package main

import (
	"fmt"
	"log/slog"
	"os"

	"agenthub/internal/apperr"
	"agenthub/internal/config"
	"agenthub/internal/logger"
)

func main() {
	if err := run(); err != nil {
		slog.Error(
			"application failed",
			"code", apperr.CodeOf(err),
			"error", err,
		)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		return err
	}

	log, err := logger.New(logger.Config{
		Level:  cfg.Log.Level,
		Format: cfg.Log.Format,
	})
	if err != nil {
		return apperr.Wrap(
			apperr.CodeInternal,
			"create logger",
			err,
		)
	}

	slog.SetDefault(log)

	address := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	log.Info(
		"application starting",
		"app", cfg.App.Name,
		"env", cfg.App.Env,
		"address", address,
	)

	return nil
}
