package mcpclient

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"agenthub/internal/tool"
)

func TestToolAdapterDefinition(t *testing.T) {
	session := &stubSession{}

	adapter, err := NewToolAdapter(
		session,
		" filesystem__echo ",
		ToolDefinition{
			Name:        " echo ",
			Description: " returns a message ",
			InputSchema: json.RawMessage(`{
				"type": "object"
			}`),
		},
	)
	if err != nil {
		t.Fatalf(
			"NewToolAdapter() returned error: %v",
			err,
		)
	}

	definition := adapter.Definition()

	if got, want := definition.Name,
		"filesystem__echo"; got != want {
		t.Fatalf(
			"Definition().Name = %q, want %q",
			got,
			want,
		)
	}

	if got, want := definition.Description,
		"returns a message"; got != want {
		t.Fatalf(
			"Definition().Description = %q, want %q",
			got,
			want,
		)
	}

	if !json.Valid(definition.Parameters) {
		t.Fatal(
			"Definition().Parameters is not valid JSON",
		)
	}
}

func TestToolAdapterExecute(t *testing.T) {
	session := &stubSession{
		callResult: CallResult{
			Content: []Content{
				{
					Type: ContentTypeText,
					Text: "first",
				},
				{
					Type: ContentTypeText,
					Text: "second",
				},
			},
		},
	}

	adapter, err := NewToolAdapter(
		session,
		"filesystem__echo",
		ToolDefinition{
			Name:        "echo",
			Description: "returns a message",
			InputSchema: json.RawMessage(`{
				"type": "object"
			}`),
		},
	)
	if err != nil {
		t.Fatalf(
			"NewToolAdapter() returned error: %v",
			err,
		)
	}

	arguments := json.RawMessage(
		`{"message":"hello"}`,
	)

	content, err := adapter.Execute(
		context.Background(),
		arguments,
	)
	if err != nil {
		t.Fatalf(
			"Execute() returned error: %v",
			err,
		)
	}

	if got, want := content,
		"first\nsecond"; got != want {
		t.Fatalf(
			"Execute() content = %q, want %q",
			got,
			want,
		)
	}

	if got, want := session.calledName,
		"echo"; got != want {
		t.Fatalf(
			"called name = %q, want %q",
			got,
			want,
		)
	}

	if got, want := string(session.calledArguments),
		string(arguments); got != want {
		t.Fatalf(
			"called arguments = %q, want %q",
			got,
			want,
		)
	}
}

func TestToolAdapterThroughExecutor(t *testing.T) {
	session := &stubSession{
		callResult: CallResult{
			Content: []Content{
				{
					Type: ContentTypeText,
					Text: "remote failure",
				},
			},
			IsError: true,
		},
	}

	adapter, err := NewToolAdapter(
		session,
		"filesystem__echo",
		ToolDefinition{
			Name:        "echo",
			Description: "returns a message",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"message": {
						"type": "string"
					}
				},
				"required": ["message"]
			}`),
		},
	)
	if err != nil {
		t.Fatalf(
			"NewToolAdapter() returned error: %v",
			err,
		)
	}

	registry := tool.NewRegistry()

	if err := registry.Register(adapter); err != nil {
		t.Fatalf(
			"Register() returned error: %v",
			err,
		)
	}

	executor, err := tool.NewExecutor(registry)
	if err != nil {
		t.Fatalf(
			"NewExecutor() returned error: %v",
			err,
		)
	}

	result, err := executor.Execute(
		context.Background(),
		tool.Call{
			ID:   "call-1",
			Name: "filesystem__echo",
			Arguments: json.RawMessage(
				`{"message":"hello"}`,
			),
		},
	)
	if err != nil {
		t.Fatalf(
			"Executor.Execute() returned error: %v",
			err,
		)
	}

	if !result.IsError {
		t.Fatal(
			"Executor result IsError = false, want true",
		)
	}

	if !strings.Contains(
		result.Content,
		"remote failure",
	) {
		t.Fatalf(
			"Executor result content = %q, want remote failure",
			result.Content,
		)
	}

	if got, want := session.calledName,
		"echo"; got != want {
		t.Fatalf(
			"called remote name = %q, want %q",
			got,
			want,
		)
	}
}

func TestToolAdapterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	cancel()

	adapter, err := NewToolAdapter(
		&stubSession{},
		"echo",
		ToolDefinition{
			Name:        "echo",
			Description: "returns a message",
			InputSchema: json.RawMessage(
				`{"type":"object"}`,
			),
		},
	)
	if err != nil {
		t.Fatalf(
			"NewToolAdapter() returned error: %v",
			err,
		)
	}

	_, err = adapter.Execute(
		ctx,
		json.RawMessage(`{}`),
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf(
			"Execute() error = %v, want context.Canceled",
			err,
		)
	}
}

func TestNewToolAdapterValidation(t *testing.T) {
	validSession := &stubSession{}

	tests := []struct {
		name       string
		session    Session
		localName  string
		definition ToolDefinition
		wantError  string
	}{
		{
			name:      "nil session",
			session:   nil,
			localName: "filesystem__echo",
			definition: ToolDefinition{
				Name:        "echo",
				Description: "returns a message",
				InputSchema: json.RawMessage(
					`{"type":"object"}`,
				),
			},
			wantError: "session is required",
		},
		{
			name:      "empty local name",
			session:   validSession,
			localName: " ",
			definition: ToolDefinition{
				Name:        "echo",
				Description: "returns a message",
				InputSchema: json.RawMessage(
					`{"type":"object"}`,
				),
			},
			wantError: "local tool name is required",
		},
		{
			name:      "empty remote tool name",
			session:   validSession,
			localName: "filesystem__echo",
			definition: ToolDefinition{
				Name:        " ",
				Description: "returns a message",
				InputSchema: json.RawMessage(
					`{"type":"object"}`,
				),
			},
			wantError: "MCP tool name is required",
		},
		{
			name:      "empty input schema",
			session:   validSession,
			localName: "filesystem__echo",
			definition: ToolDefinition{
				Name:        "echo",
				Description: "returns a message",
				InputSchema: nil,
			},
			wantError: "input schema is required",
		},
		{
			name:      "invalid input schema JSON",
			session:   validSession,
			localName: "filesystem__echo",
			definition: ToolDefinition{
				Name:        "echo",
				Description: "returns a message",
				InputSchema: json.RawMessage(
					`{`,
				),
			},
			wantError: "valid JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter, err := NewToolAdapter(
				tt.session,
				tt.localName,
				tt.definition,
			)

			if err == nil {
				t.Fatal(
					"NewToolAdapter() returned nil error",
				)
			}

			if adapter != nil {
				t.Fatal(
					"NewToolAdapter() returned non-nil adapter",
				)
			}

			if !strings.Contains(
				err.Error(),
				tt.wantError,
			) {
				t.Fatalf(
					"error = %q, want it to contain %q",
					err.Error(),
					tt.wantError,
				)
			}
		})
	}
}
