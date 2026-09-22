package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSystemPromptRequiresKnowledgeSourceCitation(t *testing.T) {
	requiredRules := []string{
		"必须调用 search_knowledge 工具",
		"来源：<source>",
		"只能引用工具结果中出现的来源",
		"如果工具结果显示“来源未知”，必须明确说明来源未知",
	}

	for _, rule := range requiredRules {
		if !strings.Contains(systemPrompt, rule) {
			t.Fatalf("systemPrompt does not contain required knowledge citation rule %q", rule)
		}
	}
}

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
