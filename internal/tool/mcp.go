package tool

import (
	"context"
	"encoding/json"

	"go-agent/internal/mcp"
)

// MCPTool 把 MCP Server 的工具包装成 tool.Tool interface
type MCPTool struct {
	client *mcp.Client
	info   mcp.ToolInfo
}

func NewMCPTool(client *mcp.Client, info mcp.ToolInfo) *MCPTool {
	return &MCPTool{client: client, info: info}
}

func (t *MCPTool) Name() string {
	return t.info.Name
}

func (t *MCPTool) Description() string {
	return t.info.Description
}

func (t *MCPTool) Parameters() json.RawMessage {
	return t.info.InputSchema
}

func (t *MCPTool) Execute(ctx context.Context, arguments json.RawMessage) (string, error) {
	return t.client.CallTool(t.info.Name, arguments)
}
