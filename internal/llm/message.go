package llm

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
