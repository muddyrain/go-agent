// Package config 提供应用配置的集中管理。
// 所有配置从环境变量加载，有合理的默认值，并在启动时验证。
package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 集中管理应用所有配置。
// 所有字段从环境变量加载，避免硬编码散落在各个文件中。
type Config struct {
	// Env 运行环境：development / test / production
	Env string

	// HTTP 服务配置
	HTTPPort int

	// 数据库配置
	DatabaseURL string

	// 连接池配置
	DBPoolMaxConns          int
	DBPoolMinConns          int
	DBPoolMaxConnLifetime   time.Duration
	DBPoolMaxConnIdleTime   time.Duration
	DBPoolHealthCheckPeriod time.Duration

	// 会话配置
	SessionTTL             time.Duration
	SessionCleanupInterval time.Duration
	SessionMaxHistory      int

	// 模型配置
	ModelAPIKey  string
	ModelBaseURL string
	ModelName    string
}

type fileConfig struct {
	Env                     string        `yaml:"env"`
	HTTPPort                int           `yaml:"http_port"`
	DatabaseURL             string        `yaml:"database_url"`
	DBPoolMaxConns          int           `yaml:"db_pool_max_conns"`
	DBPoolMinConns          int           `yaml:"db_pool_min_conns"`
	DBPoolMaxConnLifetime   time.Duration `yaml:"db_pool_max_conn_lifetime"`
	DBPoolMaxConnIdleTime   time.Duration `yaml:"db_pool_max_conn_idle_time"`
	DBPoolHealthCheckPeriod time.Duration `yaml:"db_pool_health_check_period"`
	SessionTTL              time.Duration `yaml:"session_ttl"`
	SessionCleanupInterval  time.Duration `yaml:"session_cleanup_interval"`
	SessionMaxHistory       int           `yaml:"session_max_history"`
	ModelAPIKey             string        `yaml:"model_api_key"`
	ModelBaseURL            string        `yaml:"model_base_url"`
	ModelName               string        `yaml:"model_name"`
}

// Load 加载配置，优先级：环境变量 > 配置文件 > 默认值。
func Load() *Config {
	// 第 1 步：先确定环境（环境变量最高优先级，所以先读）
	env := getEnv("AGENTHUB_ENV", "development")

	// 第 2 步：加载默认值
	cfg := loadDefaults()

	// 第 3 步：加载配置文件（如果存在），覆盖默认值
	loadConfigFile(cfg, env)

	// 第 4 步：加载环境变量，覆盖所有（最高优先级）
	loadEnvVars(cfg)

	return cfg
}

// 加载默认值
func loadDefaults() *Config {
	return &Config{
		Env:                     "development",
		HTTPPort:                8080,
		DatabaseURL:             "postgres:///agenthub?sslmode=disable",
		DBPoolMaxConns:          10,
		DBPoolMinConns:          2,
		DBPoolMaxConnLifetime:   30 * time.Minute,
		DBPoolMaxConnIdleTime:   5 * time.Minute,
		DBPoolHealthCheckPeriod: 30 * time.Second,
		SessionTTL:              30 * time.Minute,
		SessionCleanupInterval:  5 * time.Minute,
		SessionMaxHistory:       20,
	}
}

// 从环境变量加载配置，覆盖所有
func loadEnvVars(cfg *Config) {
	if v := os.Getenv("AGENTHUB_ENV"); v != "" {
		cfg.Env = strings.TrimSpace(v)
	}
	if v := os.Getenv("AGENTHUB_HTTP_PORT"); v != "" {
		if parsed, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			cfg.HTTPPort = parsed
		}
	}
	if v := os.Getenv("AGENTHUB_DATABASE_URL"); v != "" {
		cfg.DatabaseURL = strings.TrimSpace(v)
	}
	if v := os.Getenv("AGENTHUB_DB_POOL_MAX_CONNS"); v != "" {
		if parsed, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			cfg.DBPoolMaxConns = parsed
		}
	}
	if v := os.Getenv("AGENTHUB_DB_POOL_MIN_CONNS"); v != "" {
		if parsed, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			cfg.DBPoolMinConns = parsed
		}
	}
	if v := os.Getenv("AGENTHUB_DB_POOL_MAX_CONN_LIFETIME"); v != "" {
		if parsed, err := time.ParseDuration(strings.TrimSpace(v)); err == nil {
			cfg.DBPoolMaxConnLifetime = parsed
		}
	}
	if v := os.Getenv("AGENTHUB_DB_POOL_MAX_CONN_IDLE_TIME"); v != "" {
		if parsed, err := time.ParseDuration(strings.TrimSpace(v)); err == nil {
			cfg.DBPoolMaxConnIdleTime = parsed
		}
	}
	if v := os.Getenv("AGENTHUB_DB_POOL_HEALTH_CHECK_PERIOD"); v != "" {
		if parsed, err := time.ParseDuration(strings.TrimSpace(v)); err == nil {
			cfg.DBPoolHealthCheckPeriod = parsed
		}
	}
	if v := os.Getenv("AGENTHUB_SESSION_TTL"); v != "" {
		if parsed, err := time.ParseDuration(strings.TrimSpace(v)); err == nil {
			cfg.SessionTTL = parsed
		}
	}
	if v := os.Getenv("AGENTHUB_SESSION_CLEANUP_INTERVAL"); v != "" {
		if parsed, err := time.ParseDuration(strings.TrimSpace(v)); err == nil {
			cfg.SessionCleanupInterval = parsed
		}
	}
	if v := os.Getenv("AGENTHUB_SESSION_MAX_HISTORY"); v != "" {
		if parsed, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			cfg.SessionMaxHistory = parsed
		}
	}
	if v := os.Getenv("AGENTHUB_MODEL_API_KEY"); v != "" {
		cfg.ModelAPIKey = strings.TrimSpace(v)
	}
	if v := os.Getenv("AGENTHUB_MODEL_BASE_URL"); v != "" {
		cfg.ModelBaseURL = strings.TrimSpace(v)
	}
	if v := os.Getenv("AGENTHUB_MODEL_NAME"); v != "" {
		cfg.ModelName = strings.TrimSpace(v)
	}
}

// 加载配置文件
// 1. 优先用 AGENTHUB_CONFIG_FILE 指定的路径
// 2. 否则按环境自动查找 configs/config.<env>.yaml
// 3. 文件不存在时不报错，静默跳过（用默认值 + 环境变量）
func loadConfigFile(cfg *Config, env string) {
	// 确定配置文件路径
	configPath := os.Getenv("AGENTHUB_CONFIG_FILE")
	if configPath == "" {
		configPath = filepath.Join("configs", fmt.Sprintf("config.%s.yaml", env))
	}

	// 读取文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		// 文件不存在或读不了，静默跳过，用默认值 + 环境变量
		return
	}

	// 解析 YAML
	var fc fileConfig
	if err := yaml.Unmarshal(data, &fc); err != nil {
		log.Printf("warning: parse config file %s: %v, using defaults", configPath, err)
		return
	}

	// 合并到 cfg：只覆盖文件里明确设置了的字段（非零值）
	mergeFileConfig(cfg, &fc)
}

// 合并配置文件到 Config
// 只覆盖文件里非零值的字段，零值表示文件里没设置，保持默认值。
// 这样配置文件可以只写需要修改的项，不需要写全所有项。
func mergeFileConfig(cfg *Config, fc *fileConfig) {
	if fc.Env != "" {
		cfg.Env = fc.Env
	}
	if fc.HTTPPort != 0 {
		cfg.HTTPPort = fc.HTTPPort
	}
	if fc.DatabaseURL != "" {
		cfg.DatabaseURL = fc.DatabaseURL
	}
	if fc.DBPoolMaxConns != 0 {
		cfg.DBPoolMaxConns = fc.DBPoolMaxConns
	}
	if fc.DBPoolMinConns != 0 {
		cfg.DBPoolMinConns = fc.DBPoolMinConns
	}
	if fc.DBPoolMaxConnLifetime != 0 {
		cfg.DBPoolMaxConnLifetime = fc.DBPoolMaxConnLifetime
	}
	if fc.DBPoolMaxConnIdleTime != 0 {
		cfg.DBPoolMaxConnIdleTime = fc.DBPoolMaxConnIdleTime
	}
	if fc.DBPoolHealthCheckPeriod != 0 {
		cfg.DBPoolHealthCheckPeriod = fc.DBPoolHealthCheckPeriod
	}
	if fc.SessionTTL != 0 {
		cfg.SessionTTL = fc.SessionTTL
	}
	if fc.SessionCleanupInterval != 0 {
		cfg.SessionCleanupInterval = fc.SessionCleanupInterval
	}
	if fc.SessionMaxHistory != 0 {
		cfg.SessionMaxHistory = fc.SessionMaxHistory
	}
	if fc.ModelAPIKey != "" {
		cfg.ModelAPIKey = fc.ModelAPIKey
	}
	if fc.ModelBaseURL != "" {
		cfg.ModelBaseURL = fc.ModelBaseURL
	}
	if fc.ModelName != "" {
		cfg.ModelName = fc.ModelName
	}
}

// Validate 检查配置是否合法，在启动时调用，避免把配置问题延迟成运行时错误。
func (c *Config) Validate() error {
	// 必填项检查
	if c.ModelAPIKey == "" {
		return fmt.Errorf("AGENTHUB_MODEL_API_KEY is required")
	}
	if c.ModelBaseURL == "" {
		return fmt.Errorf("AGENTHUB_MODEL_BASE_URL is required")
	}
	if c.ModelName == "" {
		return fmt.Errorf("AGENTHUB_MODEL_NAME is required")
	}
	if c.DatabaseURL == "" {
		return fmt.Errorf("AGENTHUB_DATABASE_URL is required")
	}

	// 范围检查
	if c.HTTPPort < 1 || c.HTTPPort > 65535 {
		return fmt.Errorf("AGENTHUB_HTTP_PORT must be between 1 and 65535, got %d", c.HTTPPort)
	}
	if c.DBPoolMaxConns < 1 {
		return fmt.Errorf("AGENTHUB_DB_POOL_MAX_CONNS must be at least 1, got %d", c.DBPoolMaxConns)
	}
	if c.SessionMaxHistory < 1 {
		return fmt.Errorf("AGENTHUB_SESSION_MAX_HISTORY must be at least 1, got %d", c.SessionMaxHistory)
	}

	// 生产环境额外检查：防止生产环境连到本地数据库
	if c.Env == "production" {
		if strings.Contains(c.DatabaseURL, "localhost") || strings.Contains(c.DatabaseURL, "127.0.0.1") || strings.HasPrefix(c.DatabaseURL, "postgres:///") {
			return fmt.Errorf("production environment must not use local database: AGENTHUB_DATABASE_URL=%s", c.DatabaseURL)
		}
	}

	return nil
}

// IsDevelopment 判断是否是开发环境
func (c *Config) IsDevelopment() bool { return c.Env == "development" }

// IsTest 判断是否是测试环境
func (c *Config) IsTest() bool { return c.Env == "test" }

// IsProduction 判断是否是生产环境
func (c *Config) IsProduction() bool { return c.Env == "production" }

// --- 辅助函数：从环境变量读取，带默认值 ---

// getEnv 读取字符串环境变量，为空时返回默认值
func getEnv(key, defaultValue string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvInt 读取整数环境变量，解析失败或为空时返回默认值
func getEnvInt(key string, defaultValue int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

// getEnvDuration 读取时长环境变量（如 "30m"、"5s"），解析失败或为空时返回默认值
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}
