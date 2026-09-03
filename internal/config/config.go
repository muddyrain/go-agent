package config

import (
	"fmt"
	"strings"

	"agenthub/internal/apperr"

	"github.com/spf13/viper"
)

type Config struct {
	App    AppConfig    `mapstructure:"app"`
	Server ServerConfig `mapstructure:"server"`
	Log    LogConfig    `mapstructure:"log"`
	Agent  AgentConfig  `mapstructure:"agent"`
}

type AppConfig struct {
	Name string `mapstructure:"name"`
	Env  string `mapstructure:"env"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// AgentConfig 是从文件和环境变量读取的应用层配置。
// 它保持普通数据形态，不依赖 agentfactory，避免基础配置层反向依赖上层组装逻辑。
type AgentConfig struct {
	MaxSteps int               `mapstructure:"max_steps"`
	Memory   AgentMemoryConfig `mapstructure:"memory"`
}

// AgentMemoryConfig 保存可由部署环境覆盖的 Memory 策略参数。
// 当前策略不会使用的字段可以保留，真正的按策略构造由 agentfactory 负责。
type AgentMemoryConfig struct {
	Type        string `mapstructure:"type"`
	MaxMessages int    `mapstructure:"max_messages"`
	TokenBudget int    `mapstructure:"token_budget"`
}

func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)

	v.SetEnvPrefix("AGENTHUB")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetDefault("app.name", "AgentHub")
	v.SetDefault("app.env", "development")
	v.SetDefault("server.host", "127.0.0.1")
	v.SetDefault("server.port", 8080)

	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "text")

	v.SetDefault("agent.max_steps", 4)
	v.SetDefault("agent.memory.type", "sliding")
	v.SetDefault("agent.memory.max_messages", 20)
	v.SetDefault("agent.memory.token_budget", 4096)

	keys := []string{
		"app.name",
		"app.env",
		"server.host",
		"server.port",
		"log.level",
		"log.format",
		"agent.max_steps",
		"agent.memory.type",
		"agent.memory.max_messages",
		"agent.memory.token_budget",
	}

	for _, key := range keys {
		if err := v.BindEnv(key); err != nil {
			return nil, apperr.Wrap(
				apperr.CodeConfig,
				fmt.Sprintf("bind env %s", key),
				err,
			)
		}
	}

	if err := v.ReadInConfig(); err != nil {
		return nil, apperr.Wrap(apperr.CodeConfig, "read config", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, apperr.Wrap(apperr.CodeConfig, "unmarshal config", err)
	}

	if err := validate(&cfg); err != nil {
		return nil, apperr.Wrap(apperr.CodeConfig, "validate config", err)
	}
	return &cfg, nil
}

func validate(cfg *Config) error {
	if strings.TrimSpace(cfg.App.Name) == "" {
		return fmt.Errorf("app.name is required")
	}

	if strings.TrimSpace(cfg.Server.Host) == "" {
		return fmt.Errorf("server.host is required")
	}

	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535")
	}

	level := strings.ToLower(strings.TrimSpace(cfg.Log.Level))
	format := strings.ToLower(strings.TrimSpace(cfg.Log.Format))

	switch level {
	case "debug", "info", "warn", "error":
		// 合法，不需要做任何事
	default:
		return fmt.Errorf("log.level must be one of debug, info, warn, error")
	}

	switch format {
	case "text", "json":
		// 合法
	default:
		return fmt.Errorf("log.format must be one of text, json")
	}

	if cfg.Agent.MaxSteps <= 0 {
		return fmt.Errorf(
			"agent.max_steps must be greater than zero",
		)
	}

	// 配置层先按当前 Memory 策略校验对应参数，让 YAML 或环境变量错误在启动阶段暴露。
	// Factory 和具体 Memory 构造函数仍会再次保护各自边界。
	memoryType := strings.ToLower(
		strings.TrimSpace(cfg.Agent.Memory.Type),
	)

	switch memoryType {
	case "sliding":
		if cfg.Agent.Memory.MaxMessages <= 0 {
			return fmt.Errorf(
				"agent.memory.max_messages must be greater than zero",
			)
		}

	case "token_budget":
		if cfg.Agent.Memory.TokenBudget <= 0 {
			return fmt.Errorf(
				"agent.memory.token_budget must be greater than zero",
			)
		}

	default:
		return fmt.Errorf(
			"agent.memory.type must be one of sliding, token_budget",
		)
	}

	cfg.Agent.Memory.Type = memoryType

	cfg.Log.Level = level
	cfg.Log.Format = format

	return nil
}
