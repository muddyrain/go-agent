package mcpclient

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const toolNamespaceSeparator = "__"

type namespacedInvokableTool struct {
	info     schema.ToolInfo
	delegate tool.InvokableTool
}

func newNamespacedTool(
	ctx context.Context,
	serverName string,
	remoteTool tool.BaseTool,
) (tool.BaseTool, error) {
	serverName = strings.TrimSpace(serverName)
	if serverName == "" {
		return nil, fmt.Errorf("MCP server name is required")
	}

	if remoteTool == nil {
		return nil, fmt.Errorf(
			"MCP server %q returned a nil tool",
			serverName,
		)
	}

	remoteInfo, err := remoteTool.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"read MCP server %q tool info: %w",
			serverName,
			err,
		)
	}
	if remoteInfo == nil {
		return nil, fmt.Errorf(
			"MCP server %q returned nil tool info",
			serverName,
		)
	}

	remoteName := strings.TrimSpace(remoteInfo.Name)
	if remoteName == "" {
		return nil, fmt.Errorf(
			"MCP server %q returned an empty tool name",
			serverName,
		)
	}

	invokable, ok := remoteTool.(tool.InvokableTool)
	if !ok {
		return nil, fmt.Errorf(
			"MCP server %q tool %q is not invokable",
			serverName,
			remoteName,
		)
	}

	localInfo := *remoteInfo
	localInfo.Name = serverName +
		toolNamespaceSeparator +
		remoteName
	localInfo.Desc = fmt.Sprintf(
		"来自 MCP Server %s：%s",
		serverName,
		remoteInfo.Desc,
	)

	return &namespacedInvokableTool{
		info:     localInfo,
		delegate: invokable,
	}, nil
}

func (t *namespacedInvokableTool) Info(
	_ context.Context,
) (*schema.ToolInfo, error) {
	info := t.info
	return &info, nil
}

func (t *namespacedInvokableTool) InvokableRun(
	ctx context.Context,
	argumentsInJSON string,
	opts ...tool.Option,
) (string, error) {
	return t.delegate.InvokableRun(
		ctx,
		argumentsInJSON,
		opts...,
	)
}
