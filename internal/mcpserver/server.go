package mcpserver

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	ServerCalculator = "calculator"
	ServerAccounting = "accounting"
)

type serverProfile struct {
	name            string
	toolDescription string
	resultLabel     string
}

var serverProfiles = map[string]serverProfile{
	ServerCalculator: {
		name:            "agenthub-calculator-mcp",
		toolDescription: "计算两个普通数字的和",
		resultLabel:     "calculator",
	},
	ServerAccounting: {
		name:            "agenthub-accounting-mcp",
		toolDescription: "计算两个账务数字的和",
		resultLabel:     "accounting",
	},
}

func ServeStdio(serverID string) error {

	profile, ok := serverProfiles[serverID]
	if !ok {
		return fmt.Errorf("unknown MCP server %q", serverID)
	}

	mcpServer := server.NewMCPServer(
		profile.name,
		"0.1.0",
	)

	mcpServer.AddTool(
		mcp.NewTool(
			"add_numbers",
			mcp.WithDescription(profile.toolDescription),
			mcp.WithNumber(
				"a",
				mcp.Required(),
				mcp.Description("第一个数字"),
			),
			mcp.WithNumber(
				"b",
				mcp.Required(),
				mcp.Description("第二个数字"),
			),
		),
		newAddNumbersHandler(profile.resultLabel),
	)

	return server.ServeStdio(mcpServer)
}

func newAddNumbersHandler(
	resultLabel string,
) server.ToolHandlerFunc {
	return func(
		_ context.Context,
		request mcp.CallToolRequest,
	) (*mcp.CallToolResult, error) {
		a, err := request.RequireFloat("a")
		if err != nil {
			return nil, fmt.Errorf("read argument a: %w", err)
		}

		b, err := request.RequireFloat("b")
		if err != nil {
			return nil, fmt.Errorf("read argument b: %w", err)
		}

		return mcp.NewToolResultText(
			fmt.Sprintf(
				"[%s] %g + %g = %g",
				resultLabel,
				a,
				b,
				a+b,
			),
		), nil
	}
}
