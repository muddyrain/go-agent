package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"mime"
	"strings"
	"sync"
	"testing"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/ut"
)

// 仅测试替身：嵌入完整Runnable，覆盖本节使用的Invoke并记录调用。
// 测试不运行其他流式方法，不为正式代码增加一层接口。
type proposalSpy struct {
	compose.Runnable[proposalRequest, proposalResult]
	mu    sync.Mutex
	calls int
	input proposalRequest
	ctx   context.Context
	err   error
}

func (s *proposalSpy) Invoke(ctx context.Context, input proposalRequest, opts ...compose.Option) (proposalResult, error) {
	s.mu.Lock()
	s.calls++
	s.input, s.ctx = input, ctx
	s.mu.Unlock()
	if s.err != nil {
		return proposalResult{}, s.err
	}
	return s.Runnable.Invoke(ctx, input, opts...)
}

func TestHTTPProposalValidation(t *testing.T) {
	spy := &proposalSpy{err: errors.New("must not execute")}
	h := newProposalServer(spy)
	for _, tt := range []struct {
		name, contentType, body, want string
		status                        int
	}{
		{"missing_type", "", `{"task":"Go"}`, "Content-Type must be application/json\n", 415},
		{"wrong_type", "text/plain", `{"task":"Go"}`, "Content-Type must be application/json\n", 415},
		{"invalid_type", "application/json; charset", `{}`, "Content-Type must be application/json\n", 415},
		{"type_before_size", "text/plain", strings.Repeat("x", 4097), "Content-Type must be application/json\n", 415},
		{"too_large", "application/json", strings.Repeat("x", 4097), "request body too large\n", 413},
		{"empty_body", "application/json", "", "invalid JSON request\n", 400},
		{"invalid_json", "application/json", `{"task":`, "invalid JSON request\n", 400},
		{"wrong_field_type", "application/json", `{"task":42}`, "invalid JSON request\n", 400},
		{"array_body", "application/json", `[]`, "invalid JSON request\n", 400},
		{"two_values", "application/json", `{"task":"Go"}{}`, "invalid JSON request\n", 400},
		{"missing_task", "application/json", `{}`, "task is required\n", 400},
		{"empty_task", "application/json", `{"task":""}`, "task is required\n", 400},
		{"blank_task", "application/json", `{"task":" \t\n "}`, "task is required\n", 400},
		{"null_task", "application/json", `{"task":null}`, "task is required\n", 400},
		{"null_body", "application/json", `null`, "task is required\n", 400},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := ut.PerformRequest(h.Engine, "POST", "/tech-proposal", &ut.Body{
				Body: strings.NewReader(tt.body), Len: len(tt.body),
			}, ut.Header{Key: "Content-Type", Value: tt.contentType}).Result()
			if r.StatusCode() != tt.status || string(r.Body()) != tt.want || string(r.Header.Peek("Content-Type")) != "text/plain; charset=utf-8" {
				t.Fatalf("status=%d body=%q type=%q", r.StatusCode(), r.Body(), r.Header.Peek("Content-Type"))
			}
			if spy.calls != 0 {
				t.Fatalf("invalid request invoked workflow: %d", spy.calls)
			}
		})
	}
}

func TestHTTPProposalRoutingAndRegistration(t *testing.T) {
	spy := &proposalSpy{err: errors.New("must not execute")}
	h := newProposalServer(spy)
	if spy.calls != 0 {
		t.Fatalf("registration executed workflow: %d", spy.calls)
	}
	options := h.GetOptions()
	if options.Addr != "127.0.0.1:8092" || !options.HandleMethodNotAllowed || !options.SenseClientDisconnection {
		t.Fatal("unexpected server options")
	}
	for _, method := range []string{"GET", "HEAD", "PUT"} {
		r := ut.PerformRequest(h.Engine, method, "/tech-proposal", nil).Result()
		if r.StatusCode() != 405 {
			t.Fatalf("method=%s status=%d", method, r.StatusCode())
		}
	}
	if r := ut.PerformRequest(h.Engine, "POST", "/missing", nil).Result(); r.StatusCode() != 404 {
		t.Fatalf("unknown path status=%d", r.StatusCode())
	}
	if spy.calls != 0 {
		t.Fatalf("router invoked workflow: %d", spy.calls)
	}
}

func TestHTTPProposalResults(t *testing.T) {
	workflow, err := newProposalWorkflow(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	spy := &proposalSpy{Runnable: workflow}
	h := newProposalServer(spy)
	for i, tt := range []struct{ name, body, task string }{
		{"normal", `{"task":"Go"}`, "Go"},
		{"trim", `{"task":"  中文 Go  "}`, "中文 Go"},
		{"escaping", `{"task":"Go\"\\\nHTTP"}`, "Go\"\\\nHTTP"},
		{"unknown_field", `{"task":"Go","extra":true}`, "Go"},
		{"duplicate_field", `{"task":"Old","task":"New"}`, "New"},
		{"boundary", `{"task":"Go"}` + strings.Repeat(" ", 4096-len(`{"task":"Go"}`)), "Go"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := ut.PerformRequest(h.Engine, "POST", "/tech-proposal", &ut.Body{
				Body: strings.NewReader(tt.body), Len: len(tt.body),
			}, ut.Header{Key: "Content-Type", Value: "application/json; charset=utf-8"}).Result()
			if r.StatusCode() != 200 {
				t.Fatalf("status=%d body=%q", r.StatusCode(), r.Body())
			}
			mediaType, _, err := mime.ParseMediaType(string(r.Header.Peek("Content-Type")))
			if err != nil || mediaType != "application/json" {
				t.Fatalf("mediaType=%q error=%v", mediaType, err)
			}
			// 用map检查精确小写键，避免结构体解码宽容大小写掩盖标签错误。
			var fields map[string]string
			if err := json.Unmarshal(r.Body(), &fields); err != nil {
				t.Fatal(err)
			}
			want := "任务：" + tt.task + "\n方案类型：规则模板（非模型生成）\n1. 明确输入与输出\n2. 实现最小逻辑\n3. 验证成功与错误路径\n"
			if len(fields) != 1 || fields["result"] != want || spy.calls != i+1 || spy.input.Task != tt.task {
				t.Fatalf("fields=%v calls=%d input=%v", fields, spy.calls, spy.input)
			}
		})
	}
}

func TestHTTPProposalRequestContext(t *testing.T) {
	workflow, err := newProposalWorkflow(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	spy := &proposalSpy{Runnable: workflow}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := app.NewContext(0)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.SetBodyString(`{"task":"Go"}`)
	newProposalHandler(spy)(ctx, c)
	if spy.ctx != ctx || spy.calls != 1 || c.Response.StatusCode() != 200 {
		t.Fatalf("calls=%d ctx_match=%t status=%d", spy.calls, spy.ctx == ctx, c.Response.StatusCode())
	}
}

func TestHTTPProposalFailureAndNextRequest(t *testing.T) {
	workflow, err := newProposalWorkflow(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	spy := &proposalSpy{Runnable: workflow, err: errors.New("private diagnostic")}
	h := newProposalServer(spy)
	var logs bytes.Buffer
	old := log.Writer()
	log.SetOutput(&logs)
	defer log.SetOutput(old)
	r := ut.PerformRequest(h.Engine, "POST", "/tech-proposal", &ut.Body{
		Body: strings.NewReader(`{"task":"Go"}`), Len: len(`{"task":"Go"}`),
	}, ut.Header{Key: "Content-Type", Value: "application/json"}).Result()
	if r.StatusCode() != 500 || string(r.Body()) != "workflow execution failed\n" || string(r.Header.Peek("Content-Type")) != "text/plain; charset=utf-8" || spy.calls != 1 {
		t.Fatalf("calls=%d status=%d body=%q", spy.calls, r.StatusCode(), r.Body())
	}
	if !strings.Contains(logs.String(), "proposal workflow failed: private diagnostic") || strings.Contains(string(r.Body()), "private diagnostic") {
		t.Fatalf("logs=%q body=%q", logs.String(), r.Body())
	}
	spy.err = nil
	r = ut.PerformRequest(h.Engine, "POST", "/tech-proposal", &ut.Body{
		Body: strings.NewReader(`{"task":"Next"}`), Len: len(`{"task":"Next"}`),
	}, ut.Header{Key: "Content-Type", Value: "application/json"}).Result()
	if r.StatusCode() != 200 || spy.calls != 2 {
		t.Fatalf("next request calls=%d status=%d body=%q", spy.calls, r.StatusCode(), r.Body())
	}
}

func TestHTTPProposalCanceledContext(t *testing.T) {
	workflow, err := newProposalWorkflow(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	spy := &proposalSpy{Runnable: workflow}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := app.NewContext(0)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.SetBodyString(`{"task":"Go"}`)
	var logs bytes.Buffer
	old := log.Writer()
	log.SetOutput(&logs)
	defer log.SetOutput(old)
	newProposalHandler(spy)(ctx, c)
	if spy.ctx != ctx || spy.calls != 1 || c.Response.StatusCode() != 500 || string(c.Response.Body()) != "workflow execution failed\n" || !strings.Contains(logs.String(), "context canceled") {
		t.Fatalf("ctx_match=%t status=%d body=%q logs=%q", spy.ctx == ctx, c.Response.StatusCode(), c.Response.Body(), logs.String())
	}
}

func TestHTTPProposalConcurrentTasks(t *testing.T) {
	workflow, err := newProposalWorkflow(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	spy := &proposalSpy{Runnable: workflow}
	h := newProposalServer(spy)
	var wg sync.WaitGroup
	for _, task := range []string{"A", "B", "C", "D", "E", "F", "G", "H"} {
		wg.Add(1)
		go func(task string) {
			defer wg.Done()
			body, _ := json.Marshal(map[string]string{"task": task})
			r := ut.PerformRequest(h.Engine, "POST", "/tech-proposal", &ut.Body{
				Body: strings.NewReader(string(body)), Len: len(body),
			}, ut.Header{Key: "Content-Type", Value: "application/json"}).Result()
			var fields map[string]string
			want := "任务：" + task + "\n方案类型：规则模板（非模型生成）\n1. 明确输入与输出\n2. 实现最小逻辑\n3. 验证成功与错误路径\n"
			if err := json.Unmarshal(r.Body(), &fields); err != nil || r.StatusCode() != 200 || len(fields) != 1 || fields["result"] != want {
				t.Errorf("task=%q status=%d body=%q error=%v", task, r.StatusCode(), r.Body(), err)
			}
		}(task)
	}
	wg.Wait()
	if spy.calls != 8 {
		t.Fatalf("calls=%d", spy.calls)
	}
}
