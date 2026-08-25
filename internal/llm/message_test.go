package llm

import "testing"

func TestMessageConstructors(t *testing.T) {
	tests := []struct {
		name string
		got  Message
		want Message
	}{
		{
			name: "system message",
			got:  SystemMessage("follow the policy"),
			want: Message{
				Role:    RoleSystem,
				Content: "follow the policy",
			},
		},
		{
			name: "user message",
			got:  UserMessage("hello"),
			want: Message{
				Role:    RoleUser,
				Content: "hello",
			},
		},
		{
			name: "assistant message",
			got:  AssistantMessage("hi"),
			want: Message{
				Role:    RoleAssistant,
				Content: "hi",
			},
		},
		{
			name: "tool message",
			got:  ToolMessage("weather", "call-001", "25 degrees"),
			want: Message{
				Role:       RoleTool,
				Content:    "25 degrees",
				Name:       "weather",
				ToolCallID: "call-001",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("message = %#v, want %#v", tt.got, tt.want)
			}
		})
	}
}
