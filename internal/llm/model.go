package llm

import "context"

type Request struct {
	Messages []Message
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

type Model interface {
	Generate(ctx context.Context, req Request) (Response, error)
}
