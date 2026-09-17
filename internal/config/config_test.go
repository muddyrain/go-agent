package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// 清空所有 AGENTHUB_ 环境变量，验证默认值
	clearTestEnv()

	cfg := Load()

	if cfg.Env != "development" {
		t.Errorf("Env = %q, want development", cfg.Env)
	}
	if cfg.HTTPPort != 8080 {
		t.Errorf("HTTPPort = %d, want 8080", cfg.HTTPPort)
	}
	if cfg.DatabaseURL != "postgres:///agenthub?sslmode=disable" {
		t.Errorf("DatabaseURL = %q, want default", cfg.DatabaseURL)
	}
	if cfg.SessionMaxHistory != 20 {
		t.Errorf("SessionMaxHistory = %d, want 20", cfg.SessionMaxHistory)
	}
}

func TestLoad_FromEnv(t *testing.T) {
	clearTestEnv()

	os.Setenv("AGENTHUB_ENV", "production")
	os.Setenv("AGENTHUB_HTTP_PORT", "9090")
	os.Setenv("AGENTHUB_DATABASE_URL", "postgres://user:pass@db.example.com:5432/agenthub")
	os.Setenv("AGENTHUB_SESSION_MAX_HISTORY", "50")
	os.Setenv("AGENTHUB_MODEL_API_KEY", "test-key")
	os.Setenv("AGENTHUB_MODEL_BASE_URL", "https://api.example.com/v1")
	os.Setenv("AGENTHUB_MODEL_NAME", "test-model")
	defer clearTestEnv()

	cfg := Load()

	if cfg.Env != "production" {
		t.Errorf("Env = %q, want production", cfg.Env)
	}
	if cfg.HTTPPort != 9090 {
		t.Errorf("HTTPPort = %d, want 9090", cfg.HTTPPort)
	}
	if cfg.DatabaseURL != "postgres://user:pass@db.example.com:5432/agenthub" {
		t.Errorf("DatabaseURL = %q, want custom", cfg.DatabaseURL)
	}
	if cfg.SessionMaxHistory != 50 {
		t.Errorf("SessionMaxHistory = %d, want 50", cfg.SessionMaxHistory)
	}
}

func TestValidate_MissingRequired(t *testing.T) {
	clearTestEnv()
	defer clearTestEnv()

	cfg := Load() // ModelAPIKey 等为空

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want missing API key error")
	}
}

func TestValidate_InvalidPort(t *testing.T) {
	clearTestEnv()
	defer clearTestEnv()

	os.Setenv("AGENTHUB_HTTP_PORT", "99999")
	os.Setenv("AGENTHUB_MODEL_API_KEY", "test-key")
	os.Setenv("AGENTHUB_MODEL_BASE_URL", "https://api.example.com/v1")
	os.Setenv("AGENTHUB_MODEL_NAME", "test-model")

	cfg := Load()
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want invalid port error")
	}
}

func TestValidate_ProductionLocalDB(t *testing.T) {
	clearTestEnv()
	defer clearTestEnv()

	os.Setenv("AGENTHUB_ENV", "production")
	os.Setenv("AGENTHUB_DATABASE_URL", "postgres:///agenthub?sslmode=disable") // 本地默认值
	os.Setenv("AGENTHUB_MODEL_API_KEY", "test-key")
	os.Setenv("AGENTHUB_MODEL_BASE_URL", "https://api.example.com/v1")
	os.Setenv("AGENTHUB_MODEL_NAME", "test-model")

	cfg := Load()
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want production local DB error")
	}
}

func TestEnvHelpers(t *testing.T) {
	cfg := &Config{Env: "development"}
	if !cfg.IsDevelopment() {
		t.Error("IsDevelopment() = false, want true")
	}
	if cfg.IsProduction() {
		t.Error("IsProduction() = true, want false")
	}
}

// clearTestEnv 清空所有 AGENTHUB_ 前缀的环境变量，避免测试间互相影响
func clearTestEnv() {
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "AGENTHUB_") { // 用 strings.HasPrefix，不用数长度
			// 找到 = 的位置
			for i := 0; i < len(env); i++ {
				if env[i] == '=' {
					os.Unsetenv(env[:i])
					break
				}
			}
		}
	}
}

func TestLoad_ConfigFile(t *testing.T) {
	clearTestEnv()
	defer clearTestEnv()

	// 创建临时配置文件
	configFile := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte(`
http_port: 9090
session_max_history: 50
db_pool_max_conns: 15
`)
	if err := os.WriteFile(configFile, content, 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	os.Setenv("AGENTHUB_CONFIG_FILE", configFile)
	os.Setenv("AGENTHUB_MODEL_API_KEY", "test-key")
	os.Setenv("AGENTHUB_MODEL_BASE_URL", "https://api.example.com/v1")
	os.Setenv("AGENTHUB_MODEL_NAME", "test-model")

	cfg := Load()

	// 配置文件里设置的值应该覆盖默认值
	if cfg.HTTPPort != 9090 {
		t.Errorf("HTTPPort = %d, want 9090 (from config file)", cfg.HTTPPort)
	}
	if cfg.SessionMaxHistory != 50 {
		t.Errorf("SessionMaxHistory = %d, want 50 (from config file)", cfg.SessionMaxHistory)
	}
	if cfg.DBPoolMaxConns != 15 {
		t.Errorf("DBPoolMaxConns = %d, want 15 (from config file)", cfg.DBPoolMaxConns)
	}

	// 配置文件里没设置的值应该保持默认值
	if cfg.DatabaseURL != "postgres:///agenthub?sslmode=disable" {
		t.Errorf("DatabaseURL = %q, want default", cfg.DatabaseURL)
	}
}

func TestLoad_EnvVarOverridesConfigFile(t *testing.T) {
	clearTestEnv()
	defer clearTestEnv()

	// 创建临时配置文件，设置 http_port: 9090
	configFile := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte("http_port: 9090\n")
	if err := os.WriteFile(configFile, content, 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	os.Setenv("AGENTHUB_CONFIG_FILE", configFile)
	os.Setenv("AGENTHUB_HTTP_PORT", "7777") // 环境变量覆盖配置文件
	os.Setenv("AGENTHUB_MODEL_API_KEY", "test-key")
	os.Setenv("AGENTHUB_MODEL_BASE_URL", "https://api.example.com/v1")
	os.Setenv("AGENTHUB_MODEL_NAME", "test-model")

	cfg := Load()

	// 环境变量优先级最高，应该覆盖配置文件
	if cfg.HTTPPort != 7777 {
		t.Errorf("HTTPPort = %d, want 7777 (env var overrides config file)", cfg.HTTPPort)
	}
}

func TestLoad_MissingConfigFile(t *testing.T) {
	clearTestEnv()
	defer clearTestEnv()

	// 指定一个不存在的配置文件，应该不报错，用默认值
	os.Setenv("AGENTHUB_CONFIG_FILE", "/nonexistent/path/config.yaml")
	os.Setenv("AGENTHUB_MODEL_API_KEY", "test-key")
	os.Setenv("AGENTHUB_MODEL_BASE_URL", "https://api.example.com/v1")
	os.Setenv("AGENTHUB_MODEL_NAME", "test-model")

	cfg := Load()

	// 应该用默认值
	if cfg.HTTPPort != 8080 {
		t.Errorf("HTTPPort = %d, want 8080 (default)", cfg.HTTPPort)
	}
}
