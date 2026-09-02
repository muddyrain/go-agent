package tokenizer

import "agenthub/internal/llm"

// Tokenizer 抽象不同模型的消息 Token 计算规则，避免 Memory 依赖具体模型实现。
type Tokenizer interface {
	// CountMessages 返回整组消息的 Token 成本。
	CountMessages(messages []llm.Message) (int, error)
}

// FakeTokenizer 按固定的每条消息成本计数，仅用于测试和课程演示，不代表真实分词结果。
type FakeTokenizer struct {
	PerMessage int
}

var _ Tokenizer = (*FakeTokenizer)(nil)

func (f *FakeTokenizer) CountMessages(messages []llm.Message) (int, error) {
	return len(messages) * f.PerMessage, nil
}
