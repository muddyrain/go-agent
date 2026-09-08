package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadLocalEnv(t *testing.T) {
	const key = "AGENTHUB_TEST_ENV_PRIORITY"
	t.Setenv(key, "system")

	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(key+"=file\n"), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	if err := loadLocalEnv(path); err != nil {
		t.Fatalf("loadLocalEnv() error = %v", err)
	}
	if got, want := os.Getenv(key), "system"; got != want {
		t.Fatalf("%s = %q, want %q", key, got, want)
	}
}

func TestLoadLocalEnvAllowsMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.env")

	if err := loadLocalEnv(path); err != nil {
		t.Fatalf("loadLocalEnv() error = %v, want nil", err)
	}
}

func TestNewOpenAIChatModelFromEnvRequiresAPIKey(t *testing.T) {
	t.Setenv("AGENTHUB_MODEL_API_KEY", "")
	t.Setenv("AGENTHUB_MODEL_BASE_URL", "https://example.com/v1")
	t.Setenv("AGENTHUB_MODEL_NAME", "example-model")

	_, err := newOpenAIChatModelFromEnv(t.Context())
	if err == nil {
		t.Fatal("newOpenAIChatModelFromEnv() error = nil, want missing API key error")
	}
	if got, want := err.Error(), "AGENTHUB_MODEL_API_KEY is required"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}
