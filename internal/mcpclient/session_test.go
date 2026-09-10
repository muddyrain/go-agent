package mcpclient_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"agenthub/internal/mcpclient"
	"agenthub/internal/mcpserver"

	"github.com/cloudwego/eino/components/tool"
)

func TestOpenStdioDiscoversAndCallsTool(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	session, err := mcpclient.OpenStdio(
		ctx,
		os.Args[0],
		"-test.run=TestMCPServerProcess",
		"--",
		"mcp-server",
		mcpserver.ServerCalculator,
	)
	if err != nil {
		t.Fatalf("OpenStdio() error = %v", err)
	}
	defer func() {
		if err := session.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	}()

	tools := session.Tools()
	if len(tools) != 1 {
		t.Fatalf("Tools() length = %d, want 1", len(tools))
	}

	info, err := tools[0].Info(ctx)
	if err != nil {
		t.Fatalf("Info() error = %v", err)
	}
	if info.Name != "add_numbers" {
		t.Fatalf("tool name = %q, want %q", info.Name, "add_numbers")
	}

	invokable, ok := tools[0].(tool.InvokableTool)
	if !ok {
		t.Fatal("discovered MCP tool does not implement tool.InvokableTool")
	}

	result, err := invokable.InvokableRun(ctx, `{"a":7,"b":5}`)
	if err != nil {
		t.Fatalf("InvokableRun() error = %v", err)
	}
	if !strings.Contains(result, "7 + 5 = 12") {
		t.Fatalf("InvokableRun() result = %q, want calculation result", result)
	}
}

func TestOpenStdioFailsWhenServerExitsBeforeInitialize(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := mcpclient.OpenStdio(
		ctx,
		os.Args[0],
		"-test.run=TestMCPServerProcess",
	)
	if err == nil {
		t.Fatal("OpenStdio() error = nil, want initialize error")
	}
	if !strings.Contains(err.Error(), "initialize MCP session") {
		t.Fatalf("OpenStdio() error = %q, want initialize context", err)
	}
}

func TestMCPServerProcess(t *testing.T) {
	for index, arg := range os.Args {
		if arg != "mcp-server" || index+1 >= len(os.Args) {
			continue
		}

		if err := mcpserver.ServeStdio(os.Args[index+1]); err != nil {
			t.Fatal(err)
		}
		return
	}
}
