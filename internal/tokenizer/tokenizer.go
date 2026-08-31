package tokenizer

import "agenthub/internal/llm"

// Tokenizer 统计消息token数量，不同模型实现不一样
type Tokenizer interface {
	// CountMessages 统计一组消息消耗多少token
	CountMessages(messages []llm.Message) (int, error)
}

// FakeTokenizer 测试用假实现：每条消息固定算N个token
type FakeTokenizer struct {
	PerMessage int
}

var _ Tokenizer = (*FakeTokenizer)(nil)

func (f *FakeTokenizer) CountMessages(messages []llm.Message) (int, error) {
	return len(messages) * f.PerMessage, nil
}
