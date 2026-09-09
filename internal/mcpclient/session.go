package mcpclient

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	mcpadapter "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino/components/tool"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

type Session struct {
	client *client.Client
	tools  []tool.BaseTool
}

func OpenStdio(
	ctx context.Context,
	command string,
	args ...string,
) (*Session, error) {
	if strings.TrimSpace(command) == "" {
		return nil, fmt.Errorf("MCP server command is empty")
	}

	mcpClient, err := client.NewStdioMCPClientWithOptions(
		command,
		nil,
		args,
		transport.WithCommandFunc(
			func(
				startCtx context.Context,
				command string,
				_ []string,
				args []string,
			) (*exec.Cmd, error) {
				cmd := exec.CommandContext(startCtx, command, args...)

				// 不把 AgentHub 进程中的模型 API Key 等环境变量
				// 默认传递给 MCP Server 子进程。
				cmd.Env = []string{}

				return cmd, nil
			},
		),
	)
	if err != nil {
		return nil, fmt.Errorf("start MCP stdio client: %w", err)
	}

	initializeRequest := mcp.InitializeRequest{}
	initializeRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initializeRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "agenthub",
		Version: "0.1.0",
	}

	if _, err := mcpClient.Initialize(ctx, initializeRequest); err != nil {
		_ = mcpClient.Close()
		return nil, fmt.Errorf("initialize MCP session: %w", err)
	}

	tools, err := mcpadapter.GetTools(
		ctx,
		&mcpadapter.Config{
			Cli: mcpClient,
		},
	)
	if err != nil {
		_ = mcpClient.Close()
		return nil, fmt.Errorf("discover MCP tools: %w", err)
	}

	return &Session{
		client: mcpClient,
		tools:  tools,
	}, nil
}

func (s *Session) Tools() []tool.BaseTool {
	return append([]tool.BaseTool(nil), s.tools...)
}

func (s *Session) Close() error {
	if s == nil || s.client == nil {
		return nil
	}

	return s.client.Close()
}
