package main

import (
	"bytes"
	"context"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/ut"
)

// 临时接管标准日志；不使用 t.Parallel，结束时恢复原配置。
func captureHertzRequestLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var output bytes.Buffer
	writer, flags, prefix := log.Writer(), log.Flags(), log.Prefix()
	log.SetOutput(&output)
	log.SetFlags(0)
	log.SetPrefix("")
	t.Cleanup(func() {
		log.SetOutput(writer)
		log.SetFlags(flags)
		log.SetPrefix(prefix)
	})
	return &output
}

func TestLoggingMiddlewareRoutes(t *testing.T) {
	output := captureHertzRequestLog(t)
	h := newServer()
	if output.Len() != 0 {
		t.Fatal("registration must not log a request")
	}
	for _, tt := range []struct {
		name, method, target, body string
		status                     int
	}{
		{"ping", "GET", "/ping?hidden=query-value", "", 200},
		{"hello", "GET", "/hello?name=Go", "", 200},
		{"missing_name", "GET", "/hello", "", 400},
		{"wrong_method", "POST", "/hello", "", 405},
		{"unknown", "GET", "/missing", "", 404},
		{"json_success", "POST", "/greet", `{"name":"body-value"}`, 200},
		{"json_error", "POST", "/greet", `{}`, 400},
	} {
		t.Run(tt.name, func(t *testing.T) {
			output.Reset()
			r := ut.PerformRequest(h.Engine, tt.method, tt.target, &ut.Body{
				Body: strings.NewReader(tt.body), Len: len(tt.body),
			}, ut.Header{Key: "Content-Type", Value: "application/json"})
			if r.Code != tt.status {
				t.Fatalf("status=%d want=%d", r.Code, tt.status)
			}
			path := strings.SplitN(tt.target, "?", 2)[0]
			want := "hertz request method=" + tt.method + " path=\"" + path + "\" duration="
			text := output.String()
			if !strings.HasPrefix(text, want) || strings.Count(text, "hertz request") != 1 {
				t.Fatalf("request must log exactly once: %q", text)
			}
			if strings.Contains(text, "query-value") || strings.Contains(text, "body-value") {
				t.Errorf("query/body must not appear in log: %q", text)
			}
			if duration, err := time.ParseDuration(strings.TrimSpace(strings.TrimPrefix(text, want))); err != nil || duration < 0 {
				t.Errorf("invalid duration in %q: %v", text, err)
			}
		})
	}
}

func TestLoggingMiddlewareForwardsAndTimesEachRequest(t *testing.T) {
	output := captureHertzRequestLog(t)
	h := newServer()
	calls := 0
	h.POST("/timed", func(_ context.Context, c *app.RequestContext) {
		calls++
		if output.Len() != 0 {
			t.Error("request log must run after the downstream Handler")
		}
		if string(c.Request.Body()) != "input" {
			t.Error("middleware must leave request body unchanged")
		}
		// 只用于测试耗时是否涵盖后续处理，非生产等待逻辑。
		time.Sleep(25 * time.Millisecond)
		c.Header("X-Test", "preserved")
		c.String(201, "result")
	})
	for i := 0; i < 2; i++ {
		output.Reset()
		r := ut.PerformRequest(h.Engine, "POST", "/timed", &ut.Body{Body: strings.NewReader("input"), Len: 5}).Result()
		if r.StatusCode() != 201 || string(r.Body()) != "result" || string(r.Header.Peek("X-Test")) != "preserved" {
			t.Fatal("middleware changed downstream response")
		}
		text := output.String()
		_, durationText, found := strings.Cut(text, " duration=")
		duration, err := time.ParseDuration(strings.TrimSpace(durationText))
		if !found || err != nil || duration < 25*time.Millisecond {
			t.Fatalf("duration must include downstream wait: %q, err=%v", text, err)
		}
		if calls != i+1 {
			t.Fatalf("downstream calls=%d want=%d", calls, i+1)
		}
	}
}

func TestLoggingMiddlewareDoesNotRecoverPanic(t *testing.T) {
	output := captureHertzRequestLog(t)
	h := newServer()
	sentinel := &struct{ label string }{"test panic"}
	h.GET("/test-panic", func(_ context.Context, _ *app.RequestContext) { panic(sentinel) })
	var got any
	// 仅测试调用方恢复 panic；不为教学服务增加 recovery 或故障路由。
	func() {
		defer func() { got = recover() }()
		ut.PerformRequest(h.Engine, "GET", "/test-panic", nil)
	}()
	if got != sentinel {
		t.Fatalf("logging must preserve panic value: got=%v", got)
	}
	text := output.String()
	if strings.Count(text, "hertz request") != 1 || !strings.Contains(text, `method=GET path="/test-panic" duration=`) {
		t.Fatalf("deferred log missing on panic unwind: %q", text)
	}
}
