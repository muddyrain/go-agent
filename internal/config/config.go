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

	keys := []string{
		"app.name",
		"app.env",
		"server.host",
		"server.port",
		"log.level",
		"log.format",
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

	cfg.Log.Level = level
	cfg.Log.Format = format

	return nil
}
