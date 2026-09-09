package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectFileToolReadsTextFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "README.md")

	if err := os.WriteFile(
		path,
		[]byte("# AgentHub\n"),
		0o600,
	); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	projectFileTool, err := newProjectFileTool(root)
	if err != nil {
		t.Fatalf("newProjectFileTool() error = %v", err)
	}

	result, err := projectFileTool.InvokableRun(
		t.Context(),
		`{"path":"README.md"}`,
	)
	if err != nil {
		t.Fatalf("InvokableRun() error = %v", err)
	}
	if !strings.Contains(result, "AgentHub") {
		t.Fatalf(
			"InvokableRun() result = %q, want file content",
			result,
		)
	}
}

func TestProjectFileToolReturnsReadableFailure(t *testing.T) {
	root := t.TempDir()

	projectFileTool, err := newProjectFileTool(root)
	if err != nil {
		t.Fatalf("newProjectFileTool() error = %v", err)
	}

	tests := []struct {
		name string
		path string
	}{
		{
			name: "path traversal",
			path: "../secret.txt",
		},
		{
			name: "environment file",
			path: ".env",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := projectFileTool.InvokableRun(
				t.Context(),
				`{"path":"`+tt.path+`"}`,
			)
			if err != nil {
				t.Fatalf(
					"InvokableRun() error = %v, want readable tool result",
					err,
				)
			}
			if !strings.Contains(result, "无法读取项目文件") {
				t.Fatalf(
					"InvokableRun() result = %q, want failure message",
					result,
				)
			}
		})
	}
}

func TestProjectFileToolRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outsideRoot := t.TempDir()
	outsideFile := filepath.Join(outsideRoot, "secret.md")

	if err := os.WriteFile(
		outsideFile,
		[]byte("secret"),
		0o600,
	); err != nil {
		t.Fatalf("write outside file: %v", err)
	}

	linkPath := filepath.Join(root, "linked.md")
	if err := os.Symlink(outsideFile, linkPath); err != nil {
		t.Skipf("create symlink: %v", err)
	}

	projectFileTool, err := newProjectFileTool(root)
	if err != nil {
		t.Fatalf("newProjectFileTool() error = %v", err)
	}

	result, err := projectFileTool.InvokableRun(
		t.Context(),
		`{"path":"linked.md"}`,
	)
	if err != nil {
		t.Fatalf(
			"InvokableRun() error = %v, want readable tool result",
			err,
		)
	}
	if !strings.Contains(result, "file path escapes project root") {
		t.Fatalf(
			"InvokableRun() result = %q, want symlink escape error",
			result,
		)
	}
}
