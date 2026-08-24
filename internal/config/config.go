package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	App    AppConfig    `mapstructure:"app"`
	Server ServerConfig `mapstructure:"server"`
}

type AppConfig struct {
	Name string `mapstructure:"name"`
	Env  string `mapstructure:"env"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
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

	keys := []string{
		"app.name",
		"app.env",
		"server.host",
		"server.port",
	}

	for _, key := range keys {
		if err := v.BindEnv(key); err != nil {
			return nil, fmt.Errorf("bind env %s: %w", key, err)
		}
	}

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
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

	return nil
}
