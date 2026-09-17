// Package config 提供应用配置的集中管理。
// 所有配置从环境变量加载，有合理的默认值，并在启动时验证。
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
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

// Load 从环境变量加载配置，未设置的项使用默认值。
// 环境变量统一用 AGENTHUB_ 前缀，避免和其他程序冲突。
func Load() *Config {
	return &Config{
		// 环境
		Env: getEnv("AGENTHUB_ENV", "development"),

		// HTTP
		HTTPPort: getEnvInt("AGENTHUB_HTTP_PORT", 8080),

		// 数据库
		DatabaseURL: getEnv("AGENTHUB_DATABASE_URL", "postgres:///agenthub?sslmode=disable"),

		// 连接池
		DBPoolMaxConns:          getEnvInt("AGENTHUB_DB_POOL_MAX_CONNS", 10),
		DBPoolMinConns:          getEnvInt("AGENTHUB_DB_POOL_MIN_CONNS", 2),
		DBPoolMaxConnLifetime:   getEnvDuration("AGENTHUB_DB_POOL_MAX_CONN_LIFETIME", 30*time.Minute),
		DBPoolMaxConnIdleTime:   getEnvDuration("AGENTHUB_DB_POOL_MAX_CONN_IDLE_TIME", 5*time.Minute),
		DBPoolHealthCheckPeriod: getEnvDuration("AGENTHUB_DB_POOL_HEALTH_CHECK_PERIOD", 30*time.Second),

		// 会话
		SessionTTL:             getEnvDuration("AGENTHUB_SESSION_TTL", 30*time.Minute),
		SessionCleanupInterval: getEnvDuration("AGENTHUB_SESSION_CLEANUP_INTERVAL", 5*time.Minute),
		SessionMaxHistory:      getEnvInt("AGENTHUB_SESSION_MAX_HISTORY", 20),

		// 模型（没有默认值，必须显式配置）
		ModelAPIKey:  getEnv("AGENTHUB_MODEL_API_KEY", ""),
		ModelBaseURL: getEnv("AGENTHUB_MODEL_BASE_URL", ""),
		ModelName:    getEnv("AGENTHUB_MODEL_NAME", ""),
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
