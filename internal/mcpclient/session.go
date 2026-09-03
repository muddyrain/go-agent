package mcpclient

import (
	"context"
	"encoding/json"
)

// ToolDefinition 是 MCP Server 暴露的远程工具描述。
// 后续 Adapter 会把它转换为 AgentHub 的 tool.Definition。
type ToolDefinition struct {
	Name        string
	Description string
	InputSchema json.RawMessage
}

type ContentType string

const (
	ContentTypeText ContentType = "text"
)

type Content struct {
	Type ContentType
	Text string
}

// CallResult 表示一次远程工具执行结果。
// IsError 是模型可以理解和修正的工具业务错误，不等同于 Go error。
type CallResult struct {
	Content []Content
	IsError bool
}

// Session 表示 AgentHub 与一个 MCP Server 建立的逻辑会话。
// 具体的协议版本、传输方式和初始化握手由后续 SDK Adapter 负责。
type Session interface {
	ListTools(
		ctx context.Context,
	) ([]ToolDefinition, error)

	CallTool(
		ctx context.Context,
		name string,
		arguments json.RawMessage,
	) (CallResult, error)

	Close() error
}
