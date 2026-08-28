package llm

import (
	"encoding/json"

	"agenthub/internal/tool"
)

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

type Message struct {
	Role       Role
	Content    string
	Name       string
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
