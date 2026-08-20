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

func (a *Agent) Run(ctx context.Context, input string) (string, error) {
	// 初始消息
	messages := []llm.Message{
		{Role: "user", Content: input},
	}

	tools := a.registry.ToLLMTools()

	// 循环：最多 maxSteps 次

	for step := 0; step < a.maxSteps; step++ {
		// 1. 调用模型

		msg, err := a.model.Chat(ctx, messages, tools)

		if err != nil {
			return "", fmt.Errorf("step %d: chat: %w", step, err)
		}

		// 2. 回填 assistant 消息（必须，协议要求）

		messages = append(messages, msg)

		// 3. 如果没有 tool_calls，说明是最终回答，结束循环
		if len(msg.ToolCalls) == 0 {
			return msg.Content, nil
		}

		// 4. 有 tool_calls，逐个执行
		for _, tc := range msg.ToolCalls {
			result := a.executeTool(ctx, tc)
			// 5. 回填 tool 结果消息
			messages = append(messages, llm.Message{
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
