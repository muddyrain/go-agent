package agentfactory

import (
	"fmt"

	"agenthub/internal/agent"
	"agenthub/internal/llm"
	"agenthub/internal/memory"
	"agenthub/internal/tokenizer"
	"agenthub/internal/tool"
)

type MemoryType string

const (
	MemoryTypeSliding     MemoryType = "sliding"
	MemoryTypeTokenBudget MemoryType = "token_budget"
)

type Config struct {
	MaxSteps int
	Memory   MemoryConfig
}

type MemoryConfig struct {
	Type        MemoryType
	MaxMessages int
	TokenBudget int
}

type Dependencies struct {
	Model     llm.Model
	Registry  *tool.Registry
	Tokenizer tokenizer.Tokenizer
}

func validateConfig(cfg Config) error {
	if cfg.MaxSteps <= 0 {
		return fmt.Errorf(
			"max steps must be greater than zero",
		)
	}

	switch cfg.Memory.Type {
	case MemoryTypeSliding:
		if cfg.Memory.MaxMessages <= 0 {
			return fmt.Errorf(
				"memory max messages must be greater than zero",
			)
		}

	case MemoryTypeTokenBudget:
		if cfg.Memory.TokenBudget <= 0 {
			return fmt.Errorf(
				"memory token budget must be greater than zero",
			)
		}

	default:
		return fmt.Errorf(
			"unsupported memory type %q",
			cfg.Memory.Type,
		)
	}

	return nil
}

func validateDependencies(deps Dependencies) error {
	if deps.Model == nil {
		return fmt.Errorf("model is required")
	}

	if deps.Registry == nil {
		return fmt.Errorf("tool registry is required")
	}

	return nil
}

func buildMemory(
	cfg MemoryConfig,
	t tokenizer.Tokenizer,
) (memory.Memory, error) {
	switch cfg.Type {
	case MemoryTypeSliding:
		mem, err := memory.NewSimpleSliding(
			cfg.MaxMessages,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"create sliding memory: %w",
				err,
			)
		}

		return mem, nil

	case MemoryTypeTokenBudget:
		mem, err := memory.NewTokenBudgetMemory(
			cfg.TokenBudget,
			t,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"create token budget memory: %w",
				err,
			)
		}

		return mem, nil
	default:
		return nil, fmt.Errorf(
			"unsupported memory type %q",
			cfg.Type,
		)
	}
}

func Build(
	cfg Config,
	deps Dependencies,
) (*agent.Agent, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf(
			"validate factory config: %w",
			err,
		)
	}

	if err := validateDependencies(deps); err != nil {
		return nil, fmt.Errorf(
			"validate factory dependencies: %w",
			err,
		)
	}

	mem, err := buildMemory(
		cfg.Memory,
		deps.Tokenizer,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"build memory: %w",
			err,
		)
	}

	builtAgent, err := agent.New(
		deps.Model,
		deps.Registry,
		cfg.MaxSteps,
		mem,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"create agent: %w",
			err,
		)
	}

	return builtAgent, nil
}
