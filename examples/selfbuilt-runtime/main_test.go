package main

import (
	"agenthub/examples/selfbuilt-runtime/internal/config"
	"agenthub/examples/selfbuilt-runtime/internal/llm"
	"context"
	"strings"
	"testing"
)

func TestRunDemoAgentToolLoop(t *testing.T) {
	cfg := config.AgentConfig{
		MaxSteps: 4,
		Memory: config.AgentMemoryConfig{
			Type:        "sliding",
			MaxMessages: 20,
		},
	}

	result, err := runDemoAgent(
		context.Background(),
		cfg,
	)

	if err != nil {
		t.Fatalf("runDemoAgent() error = %v", err)
	}

	if got, want := result.Steps, 2; got != want {
		t.Fatalf("Steps = %d, want %d", got, want)
	}

	if got, want := len(result.Messages), 4; got != want {
		t.Fatalf("len(Messages) = %d, want %d", got, want)
	}

	roleList := []llm.Role{
		llm.RoleUser,
		llm.RoleAssistant,
		llm.RoleTool,
		llm.RoleAssistant,
	}

	for i, want := range roleList {
		got := result.Messages[i].Role
		if got != want {
			t.Fatalf(
				"Messages[%d].Role = %q, want %q",
				i,
				got,
				want,
			)
		}
	}

	toolCallMessage := result.Messages[1]
	toolMessage := result.Messages[2]

	if got, want := len(toolCallMessage.ToolCalls), 1; got != want {
		t.Fatalf("len(toolCallMessage.ToolCalls) = %d, want %d", got, want)
	}

	call := toolCallMessage.ToolCalls[0]

	if got, want := call.Name, "get_weather"; got != want {
		t.Fatalf("ToolCall.Name = %q, want %q", got, want)
	}

	if got, want := toolMessage.Name, call.Name; got != want {
		t.Fatalf("ToolMessage.Name = %q, want ToolCall.Name %q", got, want)
	}

	if got, want := toolMessage.ToolCallID, call.ID; got != want {
		t.Fatalf("ToolMessage.ToolCallID = %q, want ToolCall.ID %q", got, want)
	}

	if !strings.Contains(
		result.FinalMessage.Content,
		toolMessage.Content,
	) {
		t.Fatalf(
			"FinalMessage.Content = %q, want it to contain tool result %q",
			result.FinalMessage.Content,
			toolMessage.Content,
		)
	}
}
