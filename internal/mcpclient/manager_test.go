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

func TestOpenServersNamespacesAndRoutesDuplicateToolNames(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	manager, err := mcpclient.OpenServers(
		ctx,
		[]mcpclient.ServerConfig{
			{
				Name:    mcpserver.ServerCalculator,
				Command: os.Args[0],
				Args: []string{
					"-test.run=TestMCPServerProcess",
					"--",
					"mcp-server",
					mcpserver.ServerCalculator,
				},
			},
			{
				Name:    mcpserver.ServerAccounting,
				Command: os.Args[0],
				Args: []string{
					"-test.run=TestMCPServerProcess",
					"--",
					"mcp-server",
					mcpserver.ServerAccounting,
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("OpenServers() error = %v", err)
	}
	defer func() {
		if err := manager.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	}()

	discovered := manager.Tools()
	if len(discovered) != 2 {
		t.Fatalf("Tools() length = %d, want 2", len(discovered))
	}

	tests := []struct {
		index      int
		server     string
		name       string
		arguments  string
		wantResult string
	}{
		{
			index:      0,
			server:     mcpserver.ServerCalculator,
			name:       "calculator__add_numbers",
			arguments:  `{"a":17,"b":25}`,
			wantResult: "[calculator] 17 + 25 = 42",
		},
		{
			index:      1,
			server:     mcpserver.ServerAccounting,
			name:       "accounting__add_numbers",
			arguments:  `{"a":10.5,"b":20.5}`,
			wantResult: "[accounting] 10.5 + 20.5 = 31",
		},
	}

	for _, tt := range tests {
		discoveredTool := discovered[tt.index]
		if discoveredTool.Server != tt.server {
			t.Fatalf(
				"Tools()[%d].Server = %q, want %q",
				tt.index,
				discoveredTool.Server,
				tt.server,
			)
		}

		info, err := discoveredTool.Tool.Info(ctx)
		if err != nil {
			t.Fatalf("Info(%s) error = %v", tt.name, err)
		}
		if info.Name != tt.name {
			t.Fatalf("tool name = %q, want %q", info.Name, tt.name)
		}

		invokable, ok := discoveredTool.Tool.(tool.InvokableTool)
		if !ok {
			t.Fatalf("tool %q does not implement tool.InvokableTool", tt.name)
		}

		result, err := invokable.InvokableRun(ctx, tt.arguments)
		if err != nil {
			t.Fatalf("InvokableRun(%s) error = %v", tt.name, err)
		}
		if !strings.Contains(result, tt.wantResult) {
			t.Fatalf(
				"InvokableRun(%s) result = %q, want %q",
				tt.name,
				result,
				tt.wantResult,
			)
		}
	}
}
