package llm

import (
	"context"

	"agenthub/internal/tool"
)

// Request 是一次模型调用的完整输入；Tools 只包含模型可见的工具定义，不包含本地执行实现。
type Request struct {
	Messages []Message
	Tools    []tool.Definition
}

// Usage 记录单次模型调用的 Token 用量；Agent 多轮调用时会累加各轮 Usage。
type Usage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}

type Response struct {
	Message      Message
	FinishReason string
	Usage        Usage
}

// Stream 表示一次正在进行的模型流。
// Recv 每次读取一个事件，Close 负责释放网络连接、响应体等底层资源。
type Stream interface {
	Recv() (StreamEvent, error)
	Close() error
}

// StreamingModel 在同步 Model 能力之上增加流式响应；调用方可通过类型断言按需使用。
type StreamingModel interface {
	Model

	Stream(
		ctx context.Context,
		req Request,
	) (Stream, error)
}

// StreamEventType 区分流中的增量数据和最终完成事件。
// 使用独立类型可以避免业务代码散落未经约束的字符串。
type StreamEventType string

const (
	// delta：模型新产生了一段增量内容
	StreamEventDelta StreamEventType = "delta"
	// done：模型已经正常结束
	StreamEventDone StreamEventType = "done"
)

// StreamEvent 是统一的模型流事件。
// Delta 用于实时展示增量文本，Response 只在 Done 事件中携带本轮完整响应。
type StreamEvent struct {
	// 当前事件种类
	Type StreamEventType
	// 本次新增的文本，只在 `delta` 事件中使用
	Delta string
	// 最终完整响应，只在 `done` 事件中使用。
	Response Response
}

// Model 定义同步模型的最小能力，使 Agent 不依赖具体模型供应商或 SDK。
type Model interface {
	Generate(ctx context.Context, req Request) (Response, error)
}
