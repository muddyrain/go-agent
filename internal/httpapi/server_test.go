package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components"
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
	sessionManager *SessionManager,
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
		streamChatHandler(ctx, c, g, systemPrompt, sessionManager)
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

// --- Callback 工具事件测试 ---

func TestNewToolCallbackSendsStartAndEnd(t *testing.T) {
	toolEvents := make(chan toolEvent, 4)
	cb := newToolCallback(toolEvents)

	// Tool 组件开始 → 应收到 start 事件
	cb.OnStart(context.Background(), &callbacks.RunInfo{
		Component: components.ComponentOfTool,
		Name:      "calculator__add_numbers",
	}, nil)

	select {
	case evt := <-toolEvents:
		if evt.Type != "start" || evt.Name != "calculator__add_numbers" {
			t.Fatalf("got %+v, want start/calculator__add_numbers", evt)
		}
	default:
		t.Fatal("expected start event, got none")
	}

	// Tool 组件结束 → 应收到 end 事件
	cb.OnEnd(context.Background(), &callbacks.RunInfo{
		Component: components.ComponentOfTool,
		Name:      "calculator__add_numbers",
	}, nil)

	select {
	case evt := <-toolEvents:
		if evt.Type != "end" || evt.Name != "calculator__add_numbers" {
			t.Fatalf("got %+v, want end/calculator__add_numbers", evt)
		}
	default:
		t.Fatal("expected end event, got none")
	}
}

func TestNewToolCallbackIgnoresNonToolComponents(t *testing.T) {
	toolEvents := make(chan toolEvent, 4)
	cb := newToolCallback(toolEvents)

	// ChatModel 组件开始/结束 → 不应发送任何事件
	cb.OnStart(context.Background(), &callbacks.RunInfo{
		Component: components.ComponentOfChatModel,
		Name:      "gpt-4",
	}, nil)
	cb.OnEnd(context.Background(), &callbacks.RunInfo{
		Component: components.ComponentOfChatModel,
		Name:      "gpt-4",
	}, nil)

	select {
	case evt := <-toolEvents:
		t.Fatalf("unexpected event for non-tool component: %+v", evt)
	default:
		// 预期没有事件
	}
}

// --- 静态页面路由测试（内存测试）---

func TestIndexHandlerReturnsHTML(t *testing.T) {
	h := NewServer(fakeGenerator{}, "system prompt", NewSessionManager(NewPostgresSessionRepository(getTestDB(t)), 20))
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

	h := NewServer(generator, "system prompt", NewSessionManager(NewPostgresSessionRepository(getTestDB(t)), 20))
	resp := postJSONToEngine(t, h, "/chat", `{"message":"  hello  "}`)

	if resp.Code != 200 {
		t.Fatalf("status = %d, want 200", resp.Code)
	}
	if got := resp.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	// Hertz c.JSON 末尾不加换行符（标准库 json.Encoder.Encode 会加）
	// 响应包含随机 session_id，用 JSON 解析后检查字段
	var result chatResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if result.Answer != "world" {
		t.Fatalf("answer = %q, want world", result.Answer)
	}
	if result.SessionID == "" {
		t.Fatal("response should include session_id")
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
			h := NewServer(fakeGenerator{}, "system prompt", NewSessionManager(NewPostgresSessionRepository(getTestDB(t)), 20))
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
			h := NewServer(generator, "system prompt", NewSessionManager(NewPostgresSessionRepository(getTestDB(t)), 20))
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

	baseURL := startStreamTestServer(t, generator, "system prompt", NewSessionManager(NewPostgresSessionRepository(getTestDB(t)), 20))
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
		`data: {"session_id":"`,
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
			baseURL := startStreamTestServer(t, fakeGenerator{}, "system prompt", NewSessionManager(NewPostgresSessionRepository(getTestDB(t)), 20))
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

	baseURL := startStreamTestServer(t, generator, "system prompt", NewSessionManager(NewPostgresSessionRepository(getTestDB(t)), 20))
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

	baseURL := startStreamTestServer(t, generator, "system prompt", NewSessionManager(NewPostgresSessionRepository(getTestDB(t)), 20))
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

// --- 多轮对话与会话历史测试 ---

func TestChatHandlerMultiTurnPreservesHistory(t *testing.T) {
	// 记录每次 Generate 调用收到的消息，用于验证历史是否正确传递
	var receivedInputs [][]*schema.Message
	callCount := 0

	generator := fakeGenerator{
		generate: func(
			_ context.Context,
			input []*schema.Message,
			_ ...agent.AgentOption,
		) (*schema.Message, error) {
			copy := make([]*schema.Message, len(input))
			for i, m := range input {
				copy[i] = m
			}
			receivedInputs = append(receivedInputs, copy)
			callCount++

			if callCount == 1 {
				return schema.AssistantMessage("你好小明", nil), nil
			}
			return schema.AssistantMessage("你叫小明", nil), nil
		},
	}

	sessionManager := NewSessionManager(NewPostgresSessionRepository(getTestDB(t)), 20)
	h := NewServer(generator, "你是助手", sessionManager)

	// 第一轮：创建会话
	resp1 := postJSONToEngine(t, h, "/chat", `{"message":"我叫小明"}`)
	if resp1.Code != 200 {
		t.Fatalf("first request status = %d, body = %s", resp1.Code, resp1.Body.String())
	}

	var result1 chatResponse
	if err := json.Unmarshal(resp1.Body.Bytes(), &result1); err != nil {
		t.Fatalf("unmarshal first response: %v", err)
	}
	if result1.SessionID == "" {
		t.Fatal("first response missing session_id")
	}
	sessionID := result1.SessionID

	// 验证第一轮 Agent 收到的消息：system + user（没有历史）
	if len(receivedInputs) != 1 {
		t.Fatalf("received %d calls, want 1", len(receivedInputs))
	}
	if len(receivedInputs[0]) != 2 {
		t.Fatalf("first call got %d messages, want 2 (system + user)", len(receivedInputs[0]))
	}
	if receivedInputs[0][0].Role != schema.System {
		t.Fatalf("first message role = %s, want system", receivedInputs[0][0].Role)
	}
	if receivedInputs[0][1].Role != schema.User || receivedInputs[0][1].Content != "我叫小明" {
		t.Fatalf("second message = %+v, want user/我叫小明", receivedInputs[0][1])
	}

	// 第二轮：带上 session_id
	body2 := `{"message":"我叫什么","session_id":"` + sessionID + `"}`
	resp2 := postJSONToEngine(t, h, "/chat", body2)
	if resp2.Code != 200 {
		t.Fatalf("second request status = %d, body = %s", resp2.Code, resp2.Body.String())
	}

	var result2 chatResponse
	if err := json.Unmarshal(resp2.Body.Bytes(), &result2); err != nil {
		t.Fatalf("unmarshal second response: %v", err)
	}
	if result2.SessionID != sessionID {
		t.Fatalf("second response session_id = %s, want %s", result2.SessionID, sessionID)
	}

	// 验证第二轮 Agent 收到的消息：system + 第一轮user + 第一轮assistant + 第二轮user
	if len(receivedInputs) != 2 {
		t.Fatalf("received %d calls, want 2", len(receivedInputs))
	}
	if len(receivedInputs[1]) != 4 {
		t.Fatalf("second call got %d messages, want 4 (system + history + user)", len(receivedInputs[1]))
	}
	if receivedInputs[1][1].Role != schema.User || receivedInputs[1][1].Content != "我叫小明" {
		t.Fatalf("history user message = %+v, want user/我叫小明", receivedInputs[1][1])
	}
	if receivedInputs[1][2].Role != schema.Assistant || receivedInputs[1][2].Content != "你好小明" {
		t.Fatalf("history assistant message = %+v, want assistant/你好小明", receivedInputs[1][2])
	}
	if receivedInputs[1][3].Role != schema.User || receivedInputs[1][3].Content != "我叫什么" {
		t.Fatalf("current user message = %+v, want user/我叫什么", receivedInputs[1][3])
	}
}

func TestChatHandlerNewSessionWithoutID(t *testing.T) {
	generator := fakeGenerator{
		generate: func(
			_ context.Context,
			_ []*schema.Message,
			_ ...agent.AgentOption,
		) (*schema.Message, error) {
			return schema.AssistantMessage("回答", nil), nil
		},
	}

	h := NewServer(generator, "system", NewSessionManager(NewPostgresSessionRepository(getTestDB(t)), 20))

	resp := postJSONToEngine(t, h, "/chat", `{"message":"hello"}`)
	if resp.Code != 200 {
		t.Fatalf("status = %d", resp.Code)
	}

	var result chatResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.SessionID == "" {
		t.Fatal("response should include session_id when none was provided")
	}
}

func TestStreamChatHandlerDoneEventIncludesSessionID(t *testing.T) {
	generator := fakeGenerator{
		stream: func(
			_ context.Context,
			_ []*schema.Message,
			_ ...agent.AgentOption,
		) (*schema.StreamReader[*schema.Message], error) {
			return newMessageStream(
				schema.AssistantMessage("hello", nil),
			), nil
		},
	}

	baseURL := startStreamTestServer(t, generator, "system", NewSessionManager(NewPostgresSessionRepository(getTestDB(t)), 20))
	status, body := postJSONAndRead(t, baseURL, "/chat/stream", `{"message":"hi"}`)

	if status != 200 {
		t.Fatalf("status = %d", status)
	}

	if !strings.Contains(body, "event: done") {
		t.Fatalf("body missing done event\nbody:\n%s", body)
	}
	if !strings.Contains(body, `"session_id":"`) {
		t.Fatalf("done event missing session_id\nbody:\n%s", body)
	}
}
