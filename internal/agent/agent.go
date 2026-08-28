package agent

import (
	"context"
	"fmt"

	"agenthub/internal/llm"
	"agenthub/internal/tool"
)

type Agent struct {
	model    llm.Model
	registry *tool.Registry
	executor *tool.Executor
	maxSteps int
}

type RunResult struct {
	Messages     []llm.Message
	FinalMessage llm.Message
	Steps        int
	Usage        llm.Usage
}

func New(
	model llm.Model,
	registry *tool.Registry,
	maxSteps int,
) (*Agent, error) {
	if model == nil {
		return nil, fmt.Errorf("model is required")
	}
	if registry == nil {
		return nil, fmt.Errorf("tool registry is required")
	}
	if maxSteps <= 0 {
		return nil, fmt.Errorf(
			"max steps must be greater than zero",
		)
	}
	executor, err := tool.NewExecutor(registry)
	if err != nil {
		return nil, fmt.Errorf(
			"create tool executor: %w",
			err,
		)
	}

	return &Agent{
		model:    model,
		registry: registry,
		executor: executor,
		maxSteps: maxSteps,
	}, nil
}

func (a *Agent) Run(
	ctx context.Context,
	messages []llm.Message,
) (RunResult, error) {
	if len(messages) == 0 {
		return RunResult{}, fmt.Errorf(
			"messages are required",
		)
	}
	history := make(
		[]llm.Message,
		len(messages),
	)
	copy(history, messages)

	usage := llm.Usage{}
	for step := 1; step <= a.maxSteps; step++ {
		// 每轮模型调用
		request := llm.Request{
			Messages: history,
			Tools:    a.registry.Definitions(),
		}
		response, err := a.model.Generate(
			ctx,
			request,
		)
		if err != nil {
			return RunResult{}, fmt.Errorf(
				"generate model response at step %d: %w",
				step,
				err,
			)
		}

		usage.InputTokens += response.Usage.InputTokens
		usage.OutputTokens += response.Usage.OutputTokens
		usage.TotalTokens += response.Usage.TotalTokens

		if response.Message.Role != llm.RoleAssistant {
			return RunResult{}, fmt.Errorf(
				"model response role at step %d must be %q",
				step,
				llm.RoleAssistant,
			)
		}

		history = append(
			history,
			response.Message,
		)

		if len(response.Message.ToolCalls) == 0 {
			return RunResult{
				Messages:     history,
				FinalMessage: response.Message,
				Steps:        step,
				Usage:        usage,
			}, nil
		}

		if step == a.maxSteps {
			return RunResult{}, fmt.Errorf(
				"agent exceeded maximum steps: %d",
				a.maxSteps,
			)
		}

		for _, call := range response.Message.ToolCalls {
			toolResult, err := a.executor.Execute(
				ctx,
				call,
			)
			if err != nil {
				return RunResult{}, fmt.Errorf(
					"execute tool call %q at step %d: %w",
					call.ID,
					step,
					err,
				)
			}
			toolMessage := llm.ToolMessage(
				toolResult.Name,
				toolResult.CallID,
				toolResult.Content,
			)

			history = append(
				history,
				toolMessage,
			)
		}

	}

	return RunResult{}, fmt.Errorf(
		"agent exceeded maximum steps: %d",
		a.maxSteps,
	)
}
