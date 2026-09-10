package mcpclient

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const toolNamespaceSeparator = "__"

// namespacedInvokableTool 是模型侧的薄代理：Info 暴露 server__tool
// 唯一名称，InvokableRun 仍委托原 MCP Adapter Tool。代理不复制远程
// 执行逻辑，也不会改变 MCP Server 实际注册的原始工具名。
type namespacedInvokableTool struct {
	info     schema.ToolInfo
	delegate tool.InvokableTool
}

// newNamespacedTool 为一个远程 Eino Tool 创建模型侧命名空间代理。
// 例如 calculator Server 的 add_numbers 对模型暴露为
// calculator__add_numbers，但执行仍走原 Adapter 的 add_numbers。
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

	// 必须复制 ToolInfo 后再改名，不能原地修改 remoteInfo：Eino MCP
	// Adapter 的 InvokableRun 会使用自己保存的原始名称发起 tools/call。
	// 若把它改成命名空间名，远程 Server 找不到对应 Handler。
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

// Info 每次返回副本，避免调用者修改代理内部保存的模型侧元数据。
func (t *namespacedInvokableTool) Info(
	_ context.Context,
) (*schema.ToolInfo, error) {
	info := t.info
	return &info, nil
}

// InvokableRun 不用命名空间名重新实现 MCP 调用，而是委托原 Adapter。
// 因而 ToolsNode 按唯一名称选中此代理后，远程 tools/call 仍携带 Server
// 注册的原始工具名，并复用 Adapter 的参数解析和结果转换。
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
