package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"agenthub/examples/selfbuilt-runtime/internal/agent"
	"agenthub/examples/selfbuilt-runtime/internal/agentfactory"
	"agenthub/examples/selfbuilt-runtime/internal/apperr"
	"agenthub/examples/selfbuilt-runtime/internal/config"
	"agenthub/examples/selfbuilt-runtime/internal/llm"
	"agenthub/examples/selfbuilt-runtime/internal/logger"
	"agenthub/examples/selfbuilt-runtime/internal/tokenizer"
	"agenthub/examples/selfbuilt-runtime/internal/tool"
)

func main() {
	if err := run(); err != nil {
		slog.Error(
			"application failed",
			"code", apperr.CodeOf(err),
			"error", err,
		)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load("examples/selfbuilt-runtime/config.yaml")
	if err != nil {
		return err
	}

	log, err := logger.New(logger.Config{
		Level:  cfg.Log.Level,
		Format: cfg.Log.Format,
	})
	if err != nil {
		return apperr.Wrap(
			apperr.CodeInternal,
			"create logger",
			err,
		)
	}

	slog.SetDefault(log)

	address := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	log.Info(
		"application starting",
		"app", cfg.App.Name,
		"env", cfg.App.Env,
		"address", address,
	)

	result, err := runDemoAgent(context.Background(), cfg.Agent)
	if err != nil {
		return err
	}

	for _, message := range result.Messages {
		// tool_call: id=call-weather-001 name=get_weather arguments={"city":"杭州"}
		if message.Role == llm.RoleAssistant {
			for _, call := range message.ToolCalls {
				// 输出 ToolCall
				fmt.Printf(
					"tool_call: id=%s name=%s arguments=%s\n",
					call.ID,
					call.Name,
					string(call.Arguments),
				)
			}
		}
		// tool_result: id=call-weather-001 name=get_weather content=杭州今天晴，25°C。
		if message.Role == llm.RoleTool {
			fmt.Printf(
				"tool_result: id=%s name=%s content=%s\n",
				message.ToolCallID,
				message.Name,
				message.Content,
			)
		}
	}

	fmt.Printf(
		"assistant: %s\nsteps: %d\nusage: input=%d output=%d total=%d\n",
		result.FinalMessage.Content,
		result.Steps,
		result.Usage.InputTokens,
		result.Usage.OutputTokens,
		result.Usage.TotalTokens,
	)

	return nil
}

func runDemoAgent(
	ctx context.Context,
	cfg config.AgentConfig,
) (agent.RunResult, error) {
	// 从现有 run() 搬入：
	// 1. 创建 demoModel
	model := &demoModel{}
	// 2. 创建 Registry
	registry := tool.NewRegistry()

	// 3. 创建并注册天气工具
	weatherTool, err := newWeatherTool()
	if err != nil {
		return agent.RunResult{}, apperr.Wrap(
			apperr.CodeInternal,
			"create demo weather tool",
			err,
		)
	}
	if err := registry.Register(weatherTool); err != nil {
		return agent.RunResult{}, apperr.Wrap(
			apperr.CodeInternal,
			"register demo weather tool",
			err,
		)
	}
	// 4. 创建 FakeTokenizer
	fakeTokenizer := &tokenizer.FakeTokenizer{
		PerMessage: 1,
	}

	// 5. agentfactory.Build
	runtimeAgent, err := agentfactory.Build(
		agentfactory.ConfigFromApp(cfg),
		agentfactory.Dependencies{
			Model:     model,
			Registry:  registry,
			Tokenizer: fakeTokenizer,
		},
	)
	if err != nil {
		return agent.RunResult{}, apperr.Wrap(
			apperr.CodeInternal,
			"build demo agent",
			err,
		)
	}
	// 6. runtimeAgent.Run
	result, err := runtimeAgent.Run(
		ctx,
		[]llm.Message{
			llm.UserMessage("杭州今天天气怎么样？"),
		},
	)
	if err != nil {
		return agent.RunResult{}, apperr.Wrap(
			apperr.CodeInternal,
			"run demo agent",
			err,
		)
	}

	return result, nil
}
