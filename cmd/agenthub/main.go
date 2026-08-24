package main

import (
	"fmt"
	"log/slog"

	"agenthub/internal/config"
	"agenthub/internal/logger"
)

func main() {
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		slog.Error("load config failed", "error", err)
		return
	}

	log, err := logger.New(logger.Config{
		Level:  cfg.Log.Level,
		Format: cfg.Log.Format,
	})
	if err != nil {
		slog.Error("create logger failed", "error", err)
		return
	}

	slog.SetDefault(log) // 设置为全局默认日志器

	address := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	log.Info(
		"application starting",
		"app", cfg.App.Name,
		"env", cfg.App.Env,
		"address", address,
	)
}
