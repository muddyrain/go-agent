package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"go-agent/internal/llm"
	"go-agent/internal/tool"
)

type Agent struct {
	model    llm.Model
	registry *tool.Registry
	maxSteps int
}

func New(model llm.Model, registry *tool.Registry, maxSteps int) *Agent {
	return &Agent{
		model:    model,
		registry: registry,
		maxSteps: maxSteps,
	}
}

func (a *Agent) Run(ctx context.Context, s *Session, userInput string) (string, error) {
	// 1. 把用户输入加入会话历史
	s.AddMessage(llm.Message{Role: "user", Content: userInput})

	tools := a.registry.ToLLMTools()

	// 循环：最多 maxSteps 次

	for step := 0; step < a.maxSteps; step++ {
		// 2. 用 session.Messages 调用模型（不再是局部 messages）

		msg, err := a.model.Chat(ctx, s.Messages, tools)

		if err != nil {
			return "", fmt.Errorf("step %d: chat: %w", step, err)
		}
		// 3. 回填 assistant 消息到 session
		s.AddMessage(msg)

		// 4. 没有 tool_calls → 最终回答
		if len(msg.ToolCalls) == 0 {
			return msg.Content, nil
		}

		// 5. 执行工具，结果回填到 session
		for _, tc := range msg.ToolCalls {
			result := a.executeTool(ctx, tc)
			// 5. 回填 tool 结果消息
			s.AddMessage(llm.Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
			})
		}
	}
	// 超过 maxSteps 还没返回文本，报错
	return "", fmt.Errorf("agent exceeded max steps (%d)", a.maxSteps)
}

func (a *Agent) executeTool(ctx context.Context, tc llm.ToolCall) string {
	// 查找工具
	t, ok := a.registry.Get(tc.Function.Name)
	if !ok {
		return fmt.Sprintf("error: tool %q not found", tc.Function.Name)
	}

	// 执行工具
	result, err := t.Execute(ctx, json.RawMessage(tc.Function.Arguments))

	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}

	return result
}
