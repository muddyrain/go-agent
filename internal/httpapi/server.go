package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"strings"

	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/sse"
)

const address = "127.0.0.1:8080"

type chatRequest struct {
	Message string `json:"message"`
}

type chatResponse struct {
	Answer string `json:"answer"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// chunkEvent 是 SSE "chunk" 事件的数据体
type chunkEvent struct {
	Content string `json:"content"`
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
) *server.Hertz {
	h := server.Default(server.WithHostPorts(address))

	h.POST("/chat", func(ctx context.Context, c *app.RequestContext) {
		chatHandler(ctx, c, reactAgent, systemPrompt)
	})

	h.POST("/chat/stream", func(ctx context.Context, c *app.RequestContext) {
		streamChatHandler(ctx, c, reactAgent, systemPrompt)
	})

	return h
}

func chatHandler(
	ctx context.Context,
	c *app.RequestContext,
	reactAgent generator,
	systemPrompt string,
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

	message, err := reactAgent.Generate(
		ctx,
		[]*schema.Message{
			schema.SystemMessage(systemPrompt),
			schema.UserMessage(input.Message),
		},
	)

	if err != nil {
		log.Printf("execute HTTP chat: %v", err)
		c.JSON(502, errorResponse{Error: "agent execution failed"})
		return
	}
	if message == nil {
		log.Printf("execute HTTP chat: agent returned nil message")
		c.JSON(502, errorResponse{Error: "agent execution failed"})
		return
	}

	c.JSON(200, chatResponse{Answer: message.Content})
}

// streamChatHandler 以 SSE 协议推送 Agent 的增量回答。
// 请求解析阶段失败时返回普通 JSON 错误；一旦 sse.NewWriter 创建成功，
// 后续所有结果（包括错误）都必须通过 SSE 事件推送，不能再调用 c.JSON。
func streamChatHandler(
	ctx context.Context,
	c *app.RequestContext,
	reactAgent generator,
	systemPrompt string,
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

	// Stream 启动失败时还未写入 SSE，可以返回普通 JSON 502。
	stream, err := reactAgent.Stream(
		ctx,
		[]*schema.Message{
			schema.SystemMessage(systemPrompt),
			schema.UserMessage(input.Message),
		},
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

	// 事件类型约定：
	//   "chunk" → {"content": "增量文本"}
	//   "done"  → {}（流正常结束）
	//   "error" → {"error": "错误信息"}（流中断）
	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			writer.WriteEvent("", "done", []byte("{}"))
			return
		}
		if err != nil {
			log.Printf("receive chat stream: %v", err)
			writer.WriteEvent("", "error", []byte(`{"error":"stream interrupted"}`))
			return
		}
		if chunk == nil {
			continue
		}

		data, err := json.Marshal(chunkEvent{Content: chunk.Content})
		if err != nil {
			log.Printf("marshal chunk event: %v", err)
			continue
		}

		writer.WriteEvent("", "chunk", data)
	}
}

// Run 创建 Hertz 实例并启动监听。
// h.Spin() 会阻塞直到进程收到终止信号；
// 为了保持和 E.1 相同的函数签名，这里返回 nil（Spin 内部出错会 log.Fatal）。
func Run(
	reactAgent generator,
	systemPrompt string,
) error {
	log.Printf("HTTP server listening on http://%s", address)

	h := NewServer(reactAgent, systemPrompt)
	h.Spin()
	return nil
}
