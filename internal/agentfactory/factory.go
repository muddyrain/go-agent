package agentfactory

import (
	"fmt"

	"agenthub/internal/agent"
	"agenthub/internal/llm"
	"agenthub/internal/memory"
	"agenthub/internal/tokenizer"
	"agenthub/internal/tool"
)

// MemoryType 标识 Factory 支持的 Memory 构造策略。
// 使用独立类型可以避免在组装逻辑中散落无法被编译器区分的普通字符串。
type MemoryType string

const (
	MemoryTypeSliding     MemoryType = "sliding"
	MemoryTypeTokenBudget MemoryType = "token_budget"
)

// Config 只保存构造 Agent 所需的值配置，不持有已经创建好的运行时对象。
type Config struct {
	MaxSteps int
	Memory   MemoryConfig
}

// MemoryConfig 同时容纳不同 Memory 策略的参数；实际构造时只校验并使用当前 Type 对应的字段。
type MemoryConfig struct {
	Type        MemoryType
	MaxMessages int
	TokenBudget int
}

// Dependencies 保存由应用启动层创建并注入的运行时对象。
// Tokenizer 是条件依赖：Sliding 不需要，只有 TokenBudgetMemory 构造时才要求它存在。
type Dependencies struct {
	Model     llm.Model
	Registry  *tool.Registry
	Tokenizer tokenizer.Tokenizer
}

// validateConfig 在创建组件前检查策略和值配置，尽早返回具有 Factory 语义的错误。
// 具体组件仍会维护自身的不变量，避免其他调用方绕过 Factory 后构造出非法对象。
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

// validateDependencies 只检查所有 Agent 都必须具备的基础依赖。
// 策略专属依赖由对应构造分支检查，避免 Sliding Agent 被迫提供无用的 Tokenizer。
func validateDependencies(deps Dependencies) error {
	if deps.Model == nil {
		return fmt.Errorf("model is required")
	}

	if deps.Registry == nil {
		return fmt.Errorf("tool registry is required")
	}

	return nil
}

// buildMemory 将同一份策略配置转换为统一的 Memory 接口，
// 使 Build 无需了解 Sliding 和 TokenBudgetMemory 的具体类型差异。
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

// Build 是 Agent 的统一组装入口：先验证边界输入，再选择 Memory，最后交给 agent.New 建立核心运行时。
// Model 的网络调用和具体 Tool 业务不属于 Factory，它只负责组合外部已经创建好的依赖。
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
