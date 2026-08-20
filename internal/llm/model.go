package llm

import "context"

type Model interface {
	Chat(ctx context.Context, messages []Message, tools []Tool) (Message, error)
	ChatStream(ctx context.Context, messages []Message, tools []Tool) (<-chan StreamChunk, error)
}
