package agentfactory

import "agenthub/internal/config"

// ConfigFromApp 将应用层配置转换为 Factory 自己的构造配置。
// 转换集中在 Factory 边界，避免 config 包依赖 Agent 的组装实现。
func ConfigFromApp(cfg config.AgentConfig) Config {
	return Config{
		MaxSteps: cfg.MaxSteps,
		Memory: MemoryConfig{
			Type: MemoryType(
				cfg.Memory.Type,
			),
			MaxMessages: cfg.Memory.MaxMessages,
			TokenBudget: cfg.Memory.TokenBudget,
		},
	}
}
