package session

import (
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestContextWindowKeepsLatestCompleteTurns(t *testing.T) {
	window, err := NewContextWindow(2)
	if err != nil {
		t.Fatalf("NewContextWindow() error = %v", err)
	}

	system := schema.SystemMessage("system")
	oldUser := schema.UserMessage("old question")
	oldAssistant := schema.AssistantMessage("old answer", nil)
	toolUser := schema.UserMessage("read file")
	toolCall := schema.AssistantMessage("", []schema.ToolCall{{
		ID: "call-1",
		Function: schema.FunctionCall{
			Name:      "read_project_file",
			Arguments: `{"path":"README.md"}`,
		},
	}})
	toolResult := schema.ToolMessage("file content", "call-1", schema.WithToolName("read_project_file"))
	toolAnswer := schema.AssistantMessage("summary", nil)
	latestUser := schema.UserMessage("latest question")

	history := []*schema.Message{
		system,
		oldUser,
		oldAssistant,
		toolUser,
		toolCall,
		toolResult,
		toolAnswer,
		latestUser,
	}

	view := window.BuildModelView(history)
	want := []*schema.Message{system, toolUser, toolCall, toolResult, toolAnswer, latestUser}

	if len(view.Messages) != len(want) {
		t.Fatalf("len(Messages) = %d, want %d", len(view.Messages), len(want))
	}
	for i := range want {
		if view.Messages[i] != want[i] {
			t.Errorf("Messages[%d] = %p, want %p", i, view.Messages[i], want[i])
		}
	}
	if view.DroppedMessages != 2 {
		t.Errorf("DroppedMessages = %d, want 2", view.DroppedMessages)
	}
	if view.KeptTurns != 2 {
		t.Errorf("KeptTurns = %d, want 2", view.KeptTurns)
	}
	if len(history) != 8 {
		t.Errorf("BuildModelView changed history length to %d, want 8", len(history))
	}
}

func TestNewContextWindowRejectsNonPositiveLimit(t *testing.T) {
	for _, maxTurns := range []int{0, -1} {
		if _, err := NewContextWindow(maxTurns); err == nil {
			t.Errorf("NewContextWindow(%d) error = nil, want error", maxTurns)
		}
	}
}
