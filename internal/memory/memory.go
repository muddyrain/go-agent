package memory

import (
	"agenthub/internal/llm"
	"agenthub/internal/tokenizer"
	"fmt"
)

// Memory 根据完整历史生成当前模型可见的消息视图，不负责修改或持久化 Agent 历史。
type Memory interface {
	// Apply 根据具体策略裁剪消息，返回本次模型请求使用的视图。
	Apply(messages []llm.Message) ([]llm.Message, error)
}

// 编译期确认两种实现都满足 Memory 接口。
var _ Memory = (*SimpleSliding)(nil)
var _ Memory = (*TokenBudgetMemory)(nil)

// TokenBudgetMemory 通过 Tokenizer 反复计算消息成本，并从最旧的普通消息开始裁剪。
type TokenBudgetMemory struct {
	maxTokenBudget int
	tokenizer      tokenizer.Tokenizer
}

// SimpleSliding 保留第一条 System 消息，并按消息数量保留最新的普通消息。
type SimpleSliding struct {
	// MaxKeep 表示除第一条 System 消息外最多保留的消息条数，不是对话轮数。
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
	// 只有位于历史首位的 System 消息享受永久保留。
	first := messages[0]
	if first.Role == llm.RoleSystem {
		out = append(out, first)
		// 普通消息按时间顺序排列，因此从尾部保留最新消息。
		rest := messages[1:]
		if len(rest) > s.MaxKeep {
			rest = rest[len(rest)-s.MaxKeep:]
		}
		out = append(out, rest...)
	} else {
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

	// 第一条 System 消息不参与普通消息淘汰。
	var system *llm.Message
	var candidates []llm.Message
	if messages[0].Role == llm.RoleSystem {
		system = &messages[0]
		candidates = messages[1:]
	} else {
		candidates = messages
	}

	// 不断删除最旧的候选消息，直到完整模型视图满足 Token 预算。
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
		// 候选消息全部删除后仍超限，说明受保护的 System 消息自身已超过预算。
		if len(candidates) == 0 {
			return nil, fmt.Errorf("system message exceeds token budget")
		}
		// 超出预算时删除最旧的一条候选消息，再重新计算。
		candidates = candidates[1:]
	}
}
