package llm

import (
	"context"

	"agenthub/internal/tool"
)

type Request struct {
	Messages []Message
	Tools    []tool.Definition
}

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

type Stream interface {
	Recv() (StreamEvent, error)
	Close() error
}

type StreamingModel interface {
	Model

	Stream(
		ctx context.Context,
		req Request,
	) (Stream, error)
}

type StreamEventType string

const (
	// delta：模型新产生了一段增量内容
	StreamEventDelta StreamEventType = "delta"
	// done：模型已经正常结束
	StreamEventDone StreamEventType = "done"
)

type StreamEvent struct {
	// 当前事件种类
	Type StreamEventType
	// 本次新增的文本，只在 `delta` 事件中使用
	Delta string
	// 最终完整响应，只在 `done` 事件中使用。
	Response Response
}

type Model interface {
	Generate(ctx context.Context, req Request) (Response, error)
}
