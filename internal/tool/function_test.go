package tool

import (
	"context"
	"encoding/json"
	"testing"
)

func TestNewFunctionAndExecute(t *testing.T) {
	definition := Definition{
		Name:        " echo ",
		Description: " returns the arguments ",
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

	var receivedArguments json.RawMessage

	function, err := NewFunction(
		definition,
		func(
			_ context.Context,
			arguments json.RawMessage,
		) (string, error) {
			receivedArguments = arguments
			return "executed", nil
		},
	)
	if err != nil {
		t.Fatalf("NewFunction() returned error: %v", err)
	}

	gotDefinition := function.Definition()

	if got, want := gotDefinition.Name, "echo"; got != want {
		t.Fatalf("Definition().Name = %q, want %q", got, want)
	}

	if got, want := gotDefinition.Description, "returns the arguments"; got != want {
		t.Fatalf(
			"Definition().Description = %q, want %q",
			got,
			want,
		)
	}

	arguments := json.RawMessage(`{"message":"hello"}`)

	content, err := function.Execute(
		context.Background(),
		arguments,
	)
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if got, want := content, "executed"; got != want {
		t.Fatalf("Execute() content = %q, want %q", got, want)
	}

	if got, want := string(receivedArguments), string(arguments); got != want {
		t.Fatalf(
			"handler arguments = %q, want %q",
			got,
			want,
		)
	}
}

func TestNewFunctionValidation(t *testing.T) {
	validDefinition := Definition{
		Name:        "echo",
		Description: "returns the arguments",
		Parameters:  json.RawMessage(`{"type":"object"}`),
	}

	validHandler := Handler(
		func(
			_ context.Context,
			_ json.RawMessage,
		) (string, error) {
			return "", nil
		},
	)

	tests := []struct {
		name        string
		definition  Definition
		handler     Handler
		wantMessage string
	}{
		{
			name: "empty name",
			definition: Definition{
				Description: "returns the arguments",
				Parameters:  json.RawMessage(`{"type":"object"}`),
			},
			handler:     validHandler,
			wantMessage: "tool name is required",
		},
		{
			name: "blank name",
			definition: Definition{
				Name:        "   ",
				Description: "returns the arguments",
				Parameters:  json.RawMessage(`{"type":"object"}`),
			},
			handler:     validHandler,
			wantMessage: "tool name is required",
		},
		{
			name: "empty description",
			definition: Definition{
				Name:       "echo",
				Parameters: json.RawMessage(`{"type":"object"}`),
			},
			handler:     validHandler,
			wantMessage: "tool description is required",
		},
		{
			name: "empty parameters",
			definition: Definition{
				Name:        "echo",
				Description: "returns the arguments",
			},
			handler:     validHandler,
			wantMessage: "tool parameters are required",
		},
		{
			name: "invalid parameters JSON",
			definition: Definition{
				Name:        "echo",
				Description: "returns the arguments",
				Parameters:  json.RawMessage(`not json`),
			},
			handler:     validHandler,
			wantMessage: "tool parameters must be valid JSON",
		},
		{
			name:        "nil handler",
			definition:  validDefinition,
			handler:     nil,
			wantMessage: "tool handler is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			function, err := NewFunction(
				tt.definition,
				tt.handler,
			)

			if err == nil {
				t.Fatal("NewFunction() returned nil error, want error")
			}

			if function != nil {
				t.Fatalf(
					"NewFunction() = %#v, want nil",
					function,
				)
			}

			if got, want := err.Error(), tt.wantMessage; got != want {
				t.Fatalf(
					"NewFunction() error = %q, want %q",
					got,
					want,
				)
			}
		})
	}
}
