package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"agenthub/internal/agentfactory"
	"agenthub/internal/apperr"
	"agenthub/internal/config"
	"agenthub/internal/llm"
	"agenthub/internal/logger"
	"agenthub/internal/tokenizer"
	"agenthub/internal/tool"
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
	cfg, err := config.Load("configs/config.yaml")
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

	model := &demoModel{}

	registry := tool.NewRegistry()
	weatherTool, err := newWeatherTool()
	if err != nil {
		return apperr.Wrap(
			apperr.CodeInternal,
			"create demo weather tool",
			err,
		)
	}
	if err := registry.Register(weatherTool); err != nil {
		return apperr.Wrap(
			apperr.CodeInternal,
			"register demo weather tool",
			err,
		)
	}

	fakeTokenizer := &tokenizer.FakeTokenizer{
		PerMessage: 1,
	}

	runtimeAgent, err := agentfactory.Build(
		agentfactory.ConfigFromApp(cfg.Agent),
		agentfactory.Dependencies{
			Model:     model,
			Registry:  registry,
			Tokenizer: fakeTokenizer,
		},
	)

	if err != nil {
		return apperr.Wrap(
			apperr.CodeInternal,
			"build demo agent",
			err,
		)
	}

	result, err := runtimeAgent.Run(
		context.Background(),
		[]llm.Message{
			llm.UserMessage("杭州今天天气怎么样？"),
		},
	)
	if err != nil {
		return apperr.Wrap(
			apperr.CodeInternal,
			"run demo agent",
			err,
		)
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
