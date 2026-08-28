package llm

import (
	"agenthub/internal/tool"
	"encoding/json"
	"reflect"
	"testing"
)

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
			if !reflect.DeepEqual(tt.got, tt.want) {
				t.Fatalf("message = %#v, want %#v", tt.got, tt.want)
			}
		})
	}
}

func TestAssistantToolCalls(t *testing.T) {
	calls := []tool.Call{
		{
			ID:        "call-001",
			Name:      "get_weather",
			Arguments: json.RawMessage(`{"city":"杭州"}`),
		},
		{
			ID:        "call-002",
			Name:      "get_weather",
			Arguments: json.RawMessage(`{"city":"上海"}`),
		},
	}

	message := AssistantToolCalls(calls...)

	if message.Role != RoleAssistant {
		t.Fatalf(
			"Role = %q, want %q",
			message.Role,
			RoleAssistant,
		)
	}

	if !reflect.DeepEqual(message.ToolCalls, calls) {
		t.Fatalf(
			"ToolCalls = %#v, want %#v",
			message.ToolCalls,
			calls,
		)
	}
}

func TestAssistantToolCallsCopiesInput(t *testing.T) {
	calls := []tool.Call{
		{
			ID:        "call-001",
			Name:      "echo",
			Arguments: json.RawMessage(`{"message":"hello"}`),
		},
	}

	message := AssistantToolCalls(calls...)

	calls[0].Name = "changed"
	calls[0].Arguments[0] = 'X'

	if got, want := message.ToolCalls[0].Name, "echo"; got != want {
		t.Fatalf(
			"ToolCalls[0].Name = %q, want %q",
			got,
			want,
		)
	}

	if !json.Valid(message.ToolCalls[0].Arguments) {
		t.Fatalf(
			"ToolCalls[0].Arguments was modified: %s",
			message.ToolCalls[0].Arguments,
		)
	}
}
