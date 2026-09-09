package mcpserver

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func ServeStdio() error {
	mcpServer := server.NewMCPServer(
		"agenthub-local-mcp",
		"0.1.0",
	)

	mcpServer.AddTool(
		mcp.NewTool(
			"add_numbers",
			mcp.WithDescription("计算两个数字的和"),
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
		addNumbers,
	)

	return server.ServeStdio(mcpServer)
}

func addNumbers(
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
		fmt.Sprintf("%g + %g = %g", a, b, a+b),
	), nil
}
