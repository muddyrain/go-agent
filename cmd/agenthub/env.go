package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func loadLocalEnv(path string) error {
	err := godotenv.Load(path)
	if err == nil {
		return nil
	}

	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	return fmt.Errorf("load local environment file %q: %w", path, err)
}
