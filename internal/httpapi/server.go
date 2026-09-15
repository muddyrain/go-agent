package httpapi

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"log"
	"strings"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/sse"
)

//go:embed web/index.html
var indexHTML []byte

const address = "127.0.0.1:8080"

type chatRequest struct {
	Message   string `json:"message"`
	SessionID string `json:"session_id,omitempty"` // 前端传的会话 ID，空表示新建
}

type chatResponse struct {
	Answer    string `json:"answer"`
	SessionID string `json:"session_id,omitempty"` // 返回给前端保存的会话 ID
}

// doneEvent 是 SSE "done" 事件的数据体，携带会话 ID 供前端保存。
type doneEvent struct {
	SessionID string `json:"session_id,omitempty"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// chunkEvent 是 SSE "chunk" 事件的数据体
type chunkEvent struct {
	Content string `json:"content"`
}

// toolEvent 表示一次工具调用的开始或结束，通过 channel 从 Callback 传递到 SSE handler。
type toolEvent struct {
	Type string // "start" 或 "end"
	Name string // 工具名称
}

// chunkResult 把 stream.Recv() 的返回值打包，便于通过 channel 传递。
type chunkResult struct {
	chunk *schema.Message
	err   error
}

// indexHandler 返回嵌入的聊天页面 HTML。
// 静态内容通过 go:embed 固化进二进制，避免运行时依赖外部文件路径。
func indexHandler(_ context.Context, c *app.RequestContext) {
	c.Data(200, "text/html; charset=utf-8", indexHTML)
}

// generator 是 HTTP Handler 所需的最小 Agent 能力。
// *react.Agent 会自然同时满足 Generate 和 Stream 两个方法；
// 测试可使用本地假实现，无需访问真实模型或 MCP Server。
type generator interface {
	Generate(
		ctx context.Context,
		input []*schema.Message,
		opts ...agent.AgentOption,
	) (*schema.Message, error)

	Stream(
		ctx context.Context,
		input []*schema.Message,
		opts ...agent.AgentOption,
	) (*schema.StreamReader[*schema.Message], error)
}

// NewServer 创建并配置 Hertz 实例，注册 /chat 和 /chat/stream 两个路由。
// 分离 NewServer 和 Run 是为了测试时可以直接拿到 *server.Hertz 发请求，
// 不需要真正监听端口。
func NewServer(
	reactAgent generator,
	systemPrompt string,
	sessionManager *SessionManager,
) *server.Hertz {
	h := server.Default(server.WithHostPorts(address))

	h.GET("/", indexHandler)

	h.POST("/chat", func(ctx context.Context, c *app.RequestContext) {
		chatHandler(ctx, c, reactAgent, systemPrompt, sessionManager)
	})

	h.POST("/chat/stream", func(ctx context.Context, c *app.RequestContext) {
		streamChatHandler(ctx, c, reactAgent, systemPrompt, sessionManager)
	})

	return h
}

func chatHandler(
	ctx context.Context,
	c *app.RequestContext,
	reactAgent generator,
	systemPrompt string,
	sessionManager *SessionManager,
) {
	var input chatRequest

	// Hertz 的 c.Request.Body() 返回 []byte，直接用标准库 json.Unmarshal
	if err := json.Unmarshal(c.Request.Body(), &input); err != nil {
		c.JSON(400, errorResponse{Error: "invalid JSON request"})
		return
	}

	input.Message = strings.TrimSpace(input.Message)
	if input.Message == "" {
		c.JSON(400, errorResponse{Error: "message is required"})
		return
	}
	// 获取或创建会话；input.SessionID 为空时自动新建
	session := sessionManager.GetOrCreate(input.SessionID)

	// 构造消息：system + 历史 + 当前用户消息
	userMsg := schema.UserMessage(input.Message)
	messages := []*schema.Message{schema.SystemMessage(systemPrompt)}
	messages = append(messages, sessionManager.GetHistory(session.ID)...)
	messages = append(messages, userMsg)

	assistantMsg, err := reactAgent.Generate(ctx, messages)

	if err != nil {
		log.Printf("execute HTTP chat: %v", err)
		c.JSON(502, errorResponse{Error: "agent execution failed"})
		return
	}
	if assistantMsg == nil {
		log.Printf("execute HTTP chat: agent returned nil message")
		c.JSON(502, errorResponse{Error: "agent execution failed"})
		return
	}

	// 把本轮对话存回历史
	sessionManager.Append(session.ID, userMsg, assistantMsg)

	c.JSON(200, chatResponse{
		Answer:    assistantMsg.Content,
		SessionID: session.ID,
	})
}

// newToolCallback 创建只观察 Tool 组件的 Callback，将工具开始/结束事件
// 发送到 toolEvents channel。非阻塞发送：channel 满时丢弃事件，避免阻塞
// Eino 执行流水线。
func newToolCallback(toolEvents chan<- toolEvent) callbacks.Handler {
	return callbacks.NewHandlerBuilder().
		OnStartFn(func(
			ctx context.Context,
			info *callbacks.RunInfo,
			_ callbacks.CallbackInput,
		) context.Context {
			if info.Component == components.ComponentOfTool {
				select {
				case toolEvents <- toolEvent{Type: "start", Name: info.Name}:
				default:
				}
			}
			return ctx
		}).
		OnEndFn(func(
			ctx context.Context,
			info *callbacks.RunInfo,
			_ callbacks.CallbackOutput,
		) context.Context {
			if info.Component == components.ComponentOfTool {
				select {
				case toolEvents <- toolEvent{Type: "end", Name: info.Name}:
				default:
				}
			}
			return ctx
		}).
		Build()
}

// streamChatHandler 以 SSE 协议推送 Agent 的增量回答。
// 请求解析阶段失败时返回普通 JSON 错误；一旦 sse.NewWriter 创建成功，
// 后续所有结果（包括错误）都必须通过 SSE 事件推送，不能再调用 c.JSON。
func streamChatHandler(
	ctx context.Context,
	c *app.RequestContext,
	reactAgent generator,
	systemPrompt string,
	sessionManager *SessionManager,
) {
	var input chatRequest

	if err := json.Unmarshal(c.Request.Body(), &input); err != nil {
		c.JSON(400, errorResponse{Error: "invalid JSON request"})
		return
	}
	input.Message = strings.TrimSpace(input.Message)
	if input.Message == "" {
		c.JSON(400, errorResponse{Error: "message is required"})
		return
	}

	// toolEvents 是 Callback 与 SSE handler 之间的桥梁。
	// Callback 运行在 Eino 内部 goroutine 中，无法直接写 SSE，
	// 所以通过 channel 把工具事件传递给 handler 的 select 循环。
	toolEvents := make(chan toolEvent, 16)
	toolCallback := newToolCallback(toolEvents)

	// 获取或创建会话
	session := sessionManager.GetOrCreate(input.SessionID)
	// 构造消息：system + 历史 + 当前用户消息
	userMsg := schema.UserMessage(input.Message)
	messages := []*schema.Message{schema.SystemMessage(systemPrompt)}
	messages = append(messages, sessionManager.GetHistory(session.ID)...)
	messages = append(messages, userMsg)
	// fullAnswer 累积所有 chunk 内容，流结束后用于存历史
	var fullAnswer strings.Builder

	// Stream 启动失败时还未写入 SSE，可以返回普通 JSON 502。
	stream, err := reactAgent.Stream(
		ctx,
		messages,
		agent.WithComposeOptions(
			compose.WithCallbacks(toolCallback),
		),
	)
	if err != nil {
		log.Printf("execute HTTP chat stream: %v", err)
		c.JSON(502, errorResponse{Error: "agent execution failed"})
		return
	}
	// StreamReader 持有内部 goroutine/channel，必须关闭以释放资源。
	defer stream.Close()

	// sse.NewWriter 自动设置 Content-Type: text/event-stream 并接管响应写入。
	writer := sse.NewWriter(c)

	// stream.Recv() 是阻塞调用，不能直接放在 select 里。
	// 用 goroutine 把 Recv 结果转发到 chunkCh，这样 select 可以同时
	// 监听工具事件和文本 chunk。
	chunkCh := make(chan chunkResult)
	go func() {
		defer close(chunkCh)
		for {
			chunk, err := stream.Recv()
			chunkCh <- chunkResult{chunk: chunk, err: err}
			if err != nil {
				return
			}
		}
	}()

	// 事件类型约定：
	//   "chunk" → {"content": "增量文本"}
	//   "done"  → {}（流正常结束）
	//   "error" → {"error": "错误信息"}（流中断）
	for {
		select {
		case evt, ok := <-toolEvents:
			if !ok {
				continue
			}
			data, _ := json.Marshal(map[string]string{"name": evt.Name})
			if evt.Type == "start" {
				writer.WriteEvent("", "tool_start", data)
			} else {
				writer.WriteEvent("", "tool_end", data)
			}

		case result, ok := <-chunkCh:
			if !ok {
				// 流结束：把本轮对话存回历史，然后发 done 事件带 session_id
				assistantMsg := schema.AssistantMessage(fullAnswer.String(), nil)
				sessionManager.Append(session.ID, userMsg, assistantMsg)

				doneData, _ := json.Marshal(doneEvent{SessionID: session.ID})
				// chunkCh 关闭意味着 goroutine 已退出（流结束或出错）
				writer.WriteEvent("", "done", doneData)
				return
			}
			if errors.Is(result.err, io.EOF) {
				// 注意：EOF 是通过 chunkResult.err 传递的，
				// goroutine 发送 EOF 后会关闭 chunkCh，下一轮 select 会走到 !ok 分支
				continue
			}
			if result.err != nil {
				log.Printf("receive chat stream: %v", result.err)
				writer.WriteEvent("", "error", []byte(`{"error":"stream interrupted"}`))
				return
			}
			if result.chunk == nil {
				continue
			}
			fullAnswer.WriteString(result.chunk.Content) // 新增
			data, err := json.Marshal(chunkEvent{Content: result.chunk.Content})
			if err != nil {
				log.Printf("marshal chunk event: %v", err)
				continue
			}
			writer.WriteEvent("", "chunk", data)
		}
	}
}

// Run 创建 Hertz 实例并启动监听。
// h.Spin() 会阻塞直到进程收到终止信号；
// 为了保持和 E.1 相同的函数签名，这里返回 nil（Spin 内部出错会 log.Fatal）。
func Run(
	reactAgent generator,
	systemPrompt string,
	sessionManager *SessionManager,
	ctx context.Context,
) error {
	log.Printf("HTTP server listening on http://%s", address)

	h := NewServer(reactAgent, systemPrompt, sessionManager)

	// SetCustomSignalWaiter 替换 Hertz 默认的信号等待逻辑。
	// 默认实现自己监听 SIGINT/SIGTERM；我们改为统一监听 ctx.Done()，
	// 这样 main.go 的 signal.NotifyContext 收到信号时，所有地方
	// （清理 goroutine、HTTP 服务器）同时收到取消通知。
	//
	// 返回值语义：
	//   - return nil → Hertz 调用 Shutdown() 优雅关闭（等待进行中请求）
	//   - return err → Hertz 直接 Close() 强制退出
	h.SetCustomSignalWaiter(func(errCh chan error) error {
		select {
		case <-ctx.Done():
			log.Printf("shutdown signal received, gracefully stopping HTTP server...")
			return nil
		case err := <-errCh:
			// 服务器内部错误（如端口占用），立即退出
			return err
		}
	})

	h.Spin()
	return nil
}
