package httpapi

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/network/standard"
)

// fakeGenerator 同时实现 Generate 和 Stream，用于隔离真实模型与 MCP Server。
type fakeGenerator struct {
	generate func(
		ctx context.Context,
		input []*schema.Message,
		opts ...agent.AgentOption,
	) (*schema.Message, error)

	stream func(
		ctx context.Context,
		input []*schema.Message,
		opts ...agent.AgentOption,
	) (*schema.StreamReader[*schema.Message], error)
}

func (f fakeGenerator) Generate(
	ctx context.Context,
	input []*schema.Message,
	opts ...agent.AgentOption,
) (*schema.Message, error) {
	return f.generate(ctx, input, opts...)
}

func (f fakeGenerator) Stream(
	ctx context.Context,
	input []*schema.Message,
	opts ...agent.AgentOption,
) (*schema.StreamReader[*schema.Message], error) {
	return f.stream(ctx, input, opts...)
}

// newMessageStream 创建一个按顺序发送给定消息后正常关闭的 StreamReader。
func newMessageStream(messages ...*schema.Message) *schema.StreamReader[*schema.Message] {
	reader, writer := schema.Pipe[*schema.Message](len(messages))
	go func() {
		defer writer.Close()
		for _, msg := range messages {
			writer.Send(msg, nil)
		}
	}()
	return reader
}

// newErrorStream 先发送指定消息，再以给定错误中断流。
func newErrorStream(err error, messages ...*schema.Message) *schema.StreamReader[*schema.Message] {
	reader, writer := schema.Pipe[*schema.Message](len(messages) + 1)
	go func() {
		defer writer.Close()
		for _, msg := range messages {
			writer.Send(msg, nil)
		}
		writer.Send(nil, err)
	}()
	return reader
}

// postJSONToEngine 向 Hertz 引擎发送 POST JSON 请求（内存测试，不经过网络）。
func postJSONToEngine(
	t *testing.T,
	h *server.Hertz,
	path, body string,
) *ut.ResponseRecorder {
	t.Helper()
	return ut.PerformRequest(
		h.Engine,
		"POST",
		path,
		&ut.Body{Body: strings.NewReader(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)
}

// startStreamTestServer 在随机端口启动真实 Hertz 服务器，用于 SSE 测试。
// SSE writer 需要真实网络连接，内存测试环境无法支持。
func startStreamTestServer(
	t *testing.T,
	g generator,
	systemPrompt string,
) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	h := server.New(
		server.WithListener(ln),
		server.WithTransport(standard.NewTransporter),
	)
	h.POST("/chat/stream", func(ctx context.Context, c *app.RequestContext) {
		streamChatHandler(ctx, c, g, systemPrompt)
	})

	go h.Spin()
	// 等待服务器启动并开始监听
	time.Sleep(100 * time.Millisecond)

	t.Cleanup(func() {
		h.Close()
	})

	return "http://" + ln.Addr().String()
}

// postJSONAndRead 向真实服务器发送 POST JSON 请求并读取完整响应体。
func postJSONAndRead(
	t *testing.T,
	baseURL, path, body string,
) (int, string) {
	t.Helper()

	resp, err := http.Post(
		baseURL+path,
		"application/json",
		strings.NewReader(body),
	)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	return resp.StatusCode, string(respBody)
}

// --- 静态页面路由测试（内存测试）---

func TestIndexHandlerReturnsHTML(t *testing.T) {
	h := NewServer(fakeGenerator{}, "system prompt")
	resp := ut.PerformRequest(h.Engine, "GET", "/", nil)

	if resp.Code != 200 {
		t.Fatalf("status = %d, want 200", resp.Code)
	}
	if got := resp.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want text/html; charset=utf-8", got)
	}

	body := resp.Body.String()
	// 验证嵌入的 HTML 包含页面标识和基本结构
	if !strings.Contains(body, "AgentHub Playground") {
		t.Fatalf("body missing page title\nbody: %s", body)
	}
	if !strings.Contains(body, "<!doctype html>") {
		t.Fatalf("body missing HTML doctype\nbody: %s", body)
	}
}

// --- 同步 /chat 端点测试（内存测试）---

func TestChatHandlerReturnsAgentAnswer(t *testing.T) {
	generator := fakeGenerator{
		generate: func(
			_ context.Context,
			input []*schema.Message,
			_ ...agent.AgentOption,
		) (*schema.Message, error) {
			if len(input) != 2 {
				t.Fatalf("input messages = %d, want 2", len(input))
			}
			if input[0].Role != schema.System || input[0].Content != "system prompt" {
				t.Fatalf("system message = %#v", input[0])
			}
			if input[1].Role != schema.User || input[1].Content != "hello" {
				t.Fatalf("user message = %#v", input[1])
			}
			return schema.AssistantMessage("world", nil), nil
		},
	}

	h := NewServer(generator, "system prompt")
	resp := postJSONToEngine(t, h, "/chat", `{"message":"  hello  "}`)

	if resp.Code != 200 {
		t.Fatalf("status = %d, want 200", resp.Code)
	}
	if got := resp.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	// Hertz c.JSON 末尾不加换行符（标准库 json.Encoder.Encode 会加）
	if got := resp.Body.String(); got != `{"answer":"world"}` {
		t.Fatalf("body = %q", got)
	}
}

func TestChatHandlerRejectsInvalidRequest(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "invalid JSON",
			body:       `{"message":`,
			wantStatus: 400,
			wantBody:   `{"error":"invalid JSON request"}`,
		},
		{
			name:       "blank message",
			body:       `{"message":"   "}`,
			wantStatus: 400,
			wantBody:   `{"error":"message is required"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewServer(fakeGenerator{}, "system prompt")
			resp := postJSONToEngine(t, h, "/chat", tt.body)

			if resp.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", resp.Code, tt.wantStatus)
			}
			if got := resp.Body.String(); got != tt.wantBody {
				t.Fatalf("body = %q, want %q", got, tt.wantBody)
			}
		})
	}
}

func TestChatHandlerReturnsBadGatewayWhenAgentFails(t *testing.T) {
	tests := []struct {
		name     string
		response *schema.Message
		err      error
	}{
		{
			name: "agent error",
			err:  errors.New("model unavailable"),
		},
		{
			name: "nil message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := fakeGenerator{
				generate: func(
					context.Context,
					[]*schema.Message,
					...agent.AgentOption,
				) (*schema.Message, error) {
					return tt.response, tt.err
				},
			}
			h := NewServer(generator, "system prompt")
			resp := postJSONToEngine(t, h, "/chat", `{"message":"hello"}`)

			if resp.Code != 502 {
				t.Fatalf("status = %d, want 502", resp.Code)
			}
			if got := resp.Body.String(); got != `{"error":"agent execution failed"}` {
				t.Fatalf("body = %q", got)
			}
		})
	}
}

// --- SSE /chat/stream 端点测试（真实网络）---

func TestStreamChatHandlerSendsChunksAndDone(t *testing.T) {
	generator := fakeGenerator{
		stream: func(
			_ context.Context,
			input []*schema.Message,
			_ ...agent.AgentOption,
		) (*schema.StreamReader[*schema.Message], error) {
			if len(input) != 2 {
				t.Fatalf("input messages = %d, want 2", len(input))
			}
			if input[1].Content != "hello" {
				t.Fatalf("user message = %q", input[1].Content)
			}
			return newMessageStream(
				schema.AssistantMessage("he", nil),
				schema.AssistantMessage("llo", nil),
				schema.AssistantMessage(" world", nil),
			), nil
		},
	}

	baseURL := startStreamTestServer(t, generator, "system prompt")
	status, body := postJSONAndRead(t, baseURL, "/chat/stream", `{"message":"hello"}`)

	if status != 200 {
		t.Fatalf("status = %d, want 200", status)
	}

	// 验证三个 chunk 事件和 done 事件都出现
	for _, want := range []string{
		"event: chunk",
		`data: {"content":"he"}`,
		`data: {"content":"llo"}`,
		`data: {"content":" world"}`,
		"event: done",
		"data: {}",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q\nbody:\n%s", want, body)
		}
	}

	// 验证顺序：he 在 llo 前，llo 在 done 前
	heIdx := strings.Index(body, `"he"`)
	lloIdx := strings.Index(body, `"llo"`)
	doneIdx := strings.Index(body, "event: done")
	if heIdx >= lloIdx || lloIdx >= doneIdx {
		t.Fatalf("event order wrong: he=%d llo=%d done=%d", heIdx, lloIdx, doneIdx)
	}
}

func TestStreamChatHandlerRejectsInvalidRequest(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "invalid JSON",
			body:       `{"message":`,
			wantStatus: 400,
			wantBody:   `{"error":"invalid JSON request"}`,
		},
		{
			name:       "blank message",
			body:       `{"message":"   "}`,
			wantStatus: 400,
			wantBody:   `{"error":"message is required"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 请求校验失败时不会调用 Stream，传 nil 也安全
			baseURL := startStreamTestServer(t, fakeGenerator{}, "system prompt")
			status, body := postJSONAndRead(t, baseURL, "/chat/stream", tt.body)

			if status != tt.wantStatus {
				t.Fatalf("status = %d, want %d", status, tt.wantStatus)
			}
			if body != tt.wantBody {
				t.Fatalf("body = %q, want %q", body, tt.wantBody)
			}
		})
	}
}

func TestStreamChatHandlerReturnsBadGatewayWhenStreamFails(t *testing.T) {
	generator := fakeGenerator{
		stream: func(
			context.Context,
			[]*schema.Message,
			...agent.AgentOption,
		) (*schema.StreamReader[*schema.Message], error) {
			return nil, errors.New("model unavailable")
		},
	}

	baseURL := startStreamTestServer(t, generator, "system prompt")
	status, body := postJSONAndRead(t, baseURL, "/chat/stream", `{"message":"hello"}`)

	if status != 502 {
		t.Fatalf("status = %d, want 502", status)
	}
	if body != `{"error":"agent execution failed"}` {
		t.Fatalf("body = %q", body)
	}
}

func TestStreamChatHandlerSendsErrorEventWhenStreamInterrupts(t *testing.T) {
	generator := fakeGenerator{
		stream: func(
			context.Context,
			[]*schema.Message,
			...agent.AgentOption,
		) (*schema.StreamReader[*schema.Message], error) {
			return newErrorStream(
				errors.New("connection reset"),
				schema.AssistantMessage("partial", nil),
			), nil
		},
	}

	baseURL := startStreamTestServer(t, generator, "system prompt")
	status, body := postJSONAndRead(t, baseURL, "/chat/stream", `{"message":"hello"}`)

	if status != 200 {
		t.Fatalf("status = %d, want 200", status)
	}

	// 流中断前的 chunk 应该已经推送
	if !strings.Contains(body, `data: {"content":"partial"}`) {
		t.Fatalf("body missing partial chunk\nbody:\n%s", body)
	}
	// 中断后应该发送 error 事件，而不是 done
	if !strings.Contains(body, "event: error") {
		t.Fatalf("body missing error event\nbody:\n%s", body)
	}
	if !strings.Contains(body, `"stream interrupted"`) {
		t.Fatalf("body missing error message\nbody:\n%s", body)
	}
	if strings.Contains(body, "event: done") {
		t.Fatalf("body should not contain done event on stream interruption\nbody:\n%s", body)
	}
}
