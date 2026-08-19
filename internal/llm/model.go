package llm

import "context"

type Model interface {
	Chat(ctx context.Context, messages []Message) (string, error)
}
