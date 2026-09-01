package llm

import (
	"encoding/json"

	"agenthub/internal/tool"
)

// Role 表示消息在模型对话协议中的身份；使用独立类型和常量，避免业务代码散落角色字符串。
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message 统一承载系统、用户、助手和工具消息；具体使用哪些字段由 Role 和模型协议决定。
type Message struct {
	Role    Role
	Content string
	Name    string
	// ToolCallID 将工具结果关联到对应的 assistant tool call。
	ToolCallID string
	ToolCalls  []tool.Call
}

func SystemMessage(content string) Message {
	return Message{
		Role:    RoleSystem,
		Content: content,
	}
}

func UserMessage(content string) Message {
	return Message{
		Role:    RoleUser,
		Content: content,
	}
}

func AssistantMessage(content string) Message {
	return Message{
		Role:    RoleAssistant,
		Content: content,
	}
}

func ToolMessage(name, toolCallID, content string) Message {
	return Message{
		Role:       RoleTool,
		Content:    content,
		Name:       name,
		ToolCallID: toolCallID,
	}
}

func AssistantToolCalls(calls ...tool.Call) Message {
	// 同时复制 Call 切片和 Arguments 字节，避免调用方后续修改输入而污染消息历史。
	cloned := make([]tool.Call, len(calls))

	for index, call := range calls {
		call.Arguments = cloneRawMessage(call.Arguments)
		cloned[index] = call
	}

	return Message{
		Role:      RoleAssistant,
		ToolCalls: cloned,
	}
}

func cloneRawMessage(
	message json.RawMessage,
) json.RawMessage {
	if message == nil {
		return nil
	}

	cloned := make(json.RawMessage, len(message))
	copy(cloned, message)

	return cloned
}
