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

	model := &demoModel{
		reply: "你好，我是 AgentHub 的演示 Agent。",
	}

	registry := tool.NewRegistry()

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
			llm.UserMessage("请介绍一下你自己"),
		},
	)
	if err != nil {
		return apperr.Wrap(
			apperr.CodeInternal,
			"run demo agent",
			err,
		)
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
