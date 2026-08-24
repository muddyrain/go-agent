package logger

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

type Config struct {
	Level  string
	Format string
}

func New(cfg Config) (*slog.Logger, error) {
	// 1. 解析日志级别
	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("parse log level: %w", err)
	}
	// 2. 创建 HandlerOptions
	options := &slog.HandlerOptions{
		Level: level,
	}
	// 3. 根据 format 创建 Handler
	var handler slog.Handler
	format := strings.ToLower(strings.TrimSpace(cfg.Format))

	switch format {
	case "text":
		handler = slog.NewTextHandler(os.Stdout, options)
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, options)
	default:
		return nil, fmt.Errorf("unsupported log format %q", cfg.Format)
	}
	// 4. 返回 slog.Logger
	return slog.New(handler), nil
}

func parseLevel(value string) (slog.Level, error) {
	level := strings.ToLower(strings.TrimSpace(value))

	switch level {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("unsupported log level %q", value)
	}
}
