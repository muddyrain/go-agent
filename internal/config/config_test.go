package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agenthub/internal/apperr"
)

func validConfig() *Config {
	return &Config{
		App: AppConfig{
			Name: "AgentHub",
			Env:  "development",
		},
		Server: ServerConfig{
			Host: "127.0.0.1",
			Port: 8848,
		},
		Log: LogConfig{
			Level:  "info",
			Format: "text",
		},
	}
}

func writeConfigFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	return path
}

func TestValidateValidConfig(t *testing.T) {
	cfg := validConfig()
	cfg.Log.Level = " INFO "
	cfg.Log.Format = " JSON "

	err := validate(cfg)
	if err != nil {
		t.Fatalf("validate() returned error: %v", err)
	}

	if got, want := cfg.Log.Level, "info"; got != want {
		t.Fatalf("Log.Level = %q, want %q", got, want)
	}

	if got, want := cfg.Log.Format, "json"; got != want {
		t.Fatalf("Log.Format = %q, want %q", got, want)
	}
}

func TestValidateInvalidConfig(t *testing.T) {
	tests := []struct {
		name        string
		change      func(*Config)
		wantMessage string
	}{
		{
			name: "empty app name",
			change: func(cfg *Config) {
				cfg.App.Name = " "
			},
			wantMessage: "app.name is required",
		},
		{
			name: "empty server host",
			change: func(cfg *Config) {
				cfg.Server.Host = " "
			},
			wantMessage: "server.host is required",
		},
		{
			name: "port below minimum",
			change: func(cfg *Config) {
				cfg.Server.Port = 0
			},
			wantMessage: "server.port must be between 1 and 65535",
		},
		{
			name: "port above maximum",
			change: func(cfg *Config) {
				cfg.Server.Port = 65536
			},
			wantMessage: "server.port must be between 1 and 65535",
		},
		{
			name: "invalid log level",
			change: func(cfg *Config) {
				cfg.Log.Level = "verbose"
			},
			wantMessage: "log.level must be one of debug, info, warn, error",
		},
		{
			name: "invalid log format",
			change: func(cfg *Config) {
				cfg.Log.Format = "xml"
			},
			wantMessage: "log.format must be one of text, json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.change(cfg)

			err := validate(cfg)
			if err == nil {
				t.Fatal("validate() returned nil, want error")
			}

			if !strings.Contains(err.Error(), tt.wantMessage) {
				t.Fatalf(
					"validate() error = %q, want it to contain %q",
					err.Error(),
					tt.wantMessage,
				)
			}
		})
	}
}

func TestLoadFromFile(t *testing.T) {
	path := writeConfigFile(t, `
app:
  name: TestAgentHub
  env: test
server:
  host: 127.0.0.1
  port: 9000
log:
  level: warn
  format: json
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if got, want := cfg.App.Name, "TestAgentHub"; got != want {
		t.Fatalf("App.Name = %q, want %q", got, want)
	}

	if got, want := cfg.App.Env, "test"; got != want {
		t.Fatalf("App.Env = %q, want %q", got, want)
	}

	if got, want := cfg.Server.Host, "127.0.0.1"; got != want {
		t.Fatalf("Server.Host = %q, want %q", got, want)
	}

	if got, want := cfg.Server.Port, 9000; got != want {
		t.Fatalf("Server.Port = %d, want %d", got, want)
	}

	if got, want := cfg.Log.Level, "warn"; got != want {
		t.Fatalf("Log.Level = %q, want %q", got, want)
	}

	if got, want := cfg.Log.Format, "json"; got != want {
		t.Fatalf("Log.Format = %q, want %q", got, want)
	}
}

func TestLoadEnvironmentOverride(t *testing.T) {
	path := writeConfigFile(t, `
app:
  name: AgentHub
  env: development
server:
  host: 127.0.0.1
  port: 8848
log:
  level: info
  format: text
`)

	t.Setenv("AGENTHUB_APP_ENV", "production")
	t.Setenv("AGENTHUB_SERVER_PORT", "9000")
	t.Setenv("AGENTHUB_LOG_FORMAT", "json")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if got, want := cfg.App.Env, "production"; got != want {
		t.Fatalf("App.Env = %q, want %q", got, want)
	}

	if got, want := cfg.Server.Port, 9000; got != want {
		t.Fatalf("Server.Port = %d, want %d", got, want)
	}

	if got, want := cfg.Log.Format, "json"; got != want {
		t.Fatalf("Log.Format = %q, want %q", got, want)
	}
}

func TestLoadMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.yaml")

	cfg, err := Load(path)
	if err == nil {
		t.Fatal("Load() returned nil error, want error")
	}

	if cfg != nil {
		t.Fatalf("Load() config = %#v, want nil", cfg)
	}

	if got, want := apperr.CodeOf(err), apperr.CodeConfig; got != want {
		t.Fatalf("CodeOf() = %q, want %q", got, want)
	}

	if !strings.Contains(err.Error(), "read config") {
		t.Fatalf(
			"Load() error = %q, want it to contain %q",
			err.Error(),
			"read config",
		)
	}
}
