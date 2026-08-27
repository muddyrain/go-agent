package tool

import (
	"context"
	"encoding/json"
	"testing"
)

type echoTool struct {
	definition Definition
}

func (t echoTool) Definition() Definition {
	return t.definition
}

func (echoTool) Execute(
	ctx context.Context,
	arguments json.RawMessage,
) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
		return string(arguments), nil
	}
}

var _ Tool = echoTool{}

func TestToolDefinitionAndExecute(t *testing.T) {
	definition := Definition{
		Name:        "echo",
		Description: "returns the provided arguments",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"message": {
					"type": "string"
				}
			},
			"required": ["message"]
		}`),
	}

	tool := echoTool{
		definition: definition,
	}

	gotDefinition := tool.Definition()

	if gotDefinition.Name != definition.Name {
		t.Fatalf(
			"Definition().Name = %q, want %q",
			gotDefinition.Name,
			definition.Name,
		)
	}

	if gotDefinition.Description != definition.Description {
		t.Fatalf(
			"Definition().Description = %q, want %q",
			gotDefinition.Description,
			definition.Description,
		)
	}

	if string(gotDefinition.Parameters) != string(definition.Parameters) {
		t.Fatalf(
			"Definition().Parameters = %s, want %s",
			gotDefinition.Parameters,
			definition.Parameters,
		)
	}

	arguments := json.RawMessage(`{"message":"hello"}`)

	gotContent, err := tool.Execute(context.Background(), arguments)
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if got, want := gotContent, string(arguments); got != want {
		t.Fatalf("Execute() content = %q, want %q", got, want)
	}
}

func TestToolExecuteCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tool := echoTool{}

	content, err := tool.Execute(
		ctx,
		json.RawMessage(`{"message":"hello"}`),
	)

	if err != context.Canceled {
		t.Fatalf(
			"Execute() error = %v, want %v",
			err,
			context.Canceled,
		)
	}

	if content != "" {
		t.Fatalf("Execute() content = %q, want empty string", content)
	}
}
