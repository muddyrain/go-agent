package memory

import (
	"agenthub/internal/llm"
	"agenthub/internal/tokenizer"
	"fmt"
)

// Memory 记忆组件，输入完整历史，输出经过处理可供模型使用的消息
type Memory interface {
	// Apply 输入全部消息，返回裁剪之后的消息
	Apply(messages []llm.Message) ([]llm.Message, error)
}

var _ Memory = (*SimpleSliding)(nil)
var _ Memory = (*TokenBudgetMemory)(nil)

type TokenBudgetMemory struct {
	maxTokenBudget int
	tokenizer      tokenizer.Tokenizer
}

type SimpleSliding struct {
	// MaxKeep：除system消息外，最多保留多少**条**消息，不是对话轮数
	MaxKeep int
}

func NewSimpleSliding(maxKeep int) (*SimpleSliding, error) {
	if maxKeep <= 0 {
		return nil, fmt.Errorf("max keep must > 0")
	}
	return &SimpleSliding{
		MaxKeep: maxKeep,
	}, nil
}

func NewTokenBudgetMemory(maxTokenBudget int, t tokenizer.Tokenizer) (*TokenBudgetMemory, error) {
	if maxTokenBudget <= 0 {
		return nil, fmt.Errorf("max token budget must >0")
	}
	if t == nil {
		return nil, fmt.Errorf("tokenizer is required")
	}
	return &TokenBudgetMemory{
		maxTokenBudget: maxTokenBudget,
		tokenizer:      t,
	}, nil
}

func (s *SimpleSliding) Apply(messages []llm.Message) ([]llm.Message, error) {
	if len(messages) == 0 {
		return []llm.Message{}, nil
	}
	var out []llm.Message
	// 判断第一条是不是 system
	first := messages[0]
	if first.Role == llm.RoleSystem {
		out = append(out, first)
		// 剩余消息取 messages[1:]
		rest := messages[1:]
		if len(rest) > s.MaxKeep {
			// 丢弃前面老的，保留末尾 MaxKeep
			rest = rest[len(rest)-s.MaxKeep:]
		}
		out = append(out, rest...)
	} else {
		// 没有system，直接全部消息尾部截取
		if len(messages) > s.MaxKeep {
			out = messages[len(messages)-s.MaxKeep:]
		} else {
			out = messages
		}
	}
	return out, nil
}

func (m *TokenBudgetMemory) Apply(messages []llm.Message) ([]llm.Message, error) {
	if len(messages) == 0 {
		return []llm.Message{}, nil
	}

	// 保护头部system消息
	var system *llm.Message
	var candidates []llm.Message
	if messages[0].Role == llm.RoleSystem {
		system = &messages[0]
		candidates = messages[1:]
	} else {
		candidates = messages
	}

	// 从后往前尝试：不断丢弃最旧的candidates，直到总token ≤预算
	for {
		var buf []llm.Message
		if system != nil {
			buf = append(buf, *system)
		}
		buf = append(buf, candidates...)

		total, err := m.tokenizer.CountMessages(buf)
		if err != nil {
			return nil, fmt.Errorf("count token: %w", err)
		}
		if total <= m.maxTokenBudget {
			return buf, nil
		}
		// 超预算，丢弃最旧一条对话
		if len(candidates) == 0 {
			// 只剩system还超预算，system本身就超过上限，无法处理
			return nil, fmt.Errorf("system message exceeds token budget")
		}
		candidates = candidates[1:]
	}
}
