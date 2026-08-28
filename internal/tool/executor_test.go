package tool

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func newExecutorWithTool(
	t *testing.T,
	handler Handler,
) *Executor {
	t.Helper()

	function, err := NewFunction(
		Definition{
			Name:        "echo",
			Description: "test echo tool",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"message": {
						"type": "string"
					}
				},
				"required": ["message"]
			}`),
		},
		handler,
	)
	if err != nil {
		t.Fatalf("NewFunction() returned error: %v", err)
	}

	registry := NewRegistry()

	if err := registry.Register(function); err != nil {
		t.Fatalf("Register() returned error: %v", err)
	}

	executor, err := NewExecutor(registry)
	if err != nil {
		t.Fatalf("NewExecutor() returned error: %v", err)
	}

	return executor
}

func TestExecutorExecuteSuccess(t *testing.T) {
	var receivedArguments json.RawMessage

	executor := newExecutorWithTool(
		t,
		func(
			_ context.Context,
			arguments json.RawMessage,
		) (string, error) {
			receivedArguments = arguments
			return "executed", nil
		},
	)

	call := Call{
		ID:        " call-001 ",
		Name:      " echo ",
		Arguments: json.RawMessage(`{"message":"hello"}`),
	}

	result, err := executor.Execute(
		context.Background(),
		call,
	)
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	want := Result{
		CallID:  "call-001",
		Name:    "echo",
		Content: "executed",
		IsError: false,
	}

	if result != want {
		t.Fatalf(
			"Execute() result = %#v, want %#v",
			result,
			want,
		)
	}

	if got, want := string(receivedArguments), string(call.Arguments); got != want {
		t.Fatalf(
			"handler arguments = %q, want %q",
			got,
			want,
		)
	}
}

func TestExecutorExecuteToolError(t *testing.T) {
	toolErr := errors.New("echo failed")

	executor := newExecutorWithTool(
		t,
		func(
			_ context.Context,
			_ json.RawMessage,
		) (string, error) {
			return "", toolErr
		},
	)

	result, err := executor.Execute(
		context.Background(),
		Call{
			ID:        "call-001",
			Name:      "echo",
			Arguments: json.RawMessage(`{"message":"hello"}`),
		},
	)
	if err != nil {
		t.Fatalf("Execute() returned platform error: %v", err)
	}

	want := Result{
		CallID:  "call-001",
		Name:    "echo",
		Content: "echo failed",
		IsError: true,
	}

	if result != want {
		t.Fatalf(
			"Execute() result = %#v, want %#v",
			result,
			want,
		)
	}
}

func TestExecutorExecuteCancellation(t *testing.T) {
	executor := newExecutorWithTool(
		t,
		func(
			ctx context.Context,
			_ json.RawMessage,
		) (string, error) {
			return "", ctx.Err()
		},
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := executor.Execute(
		ctx,
		Call{
			ID:        "call-001",
			Name:      "echo",
			Arguments: json.RawMessage(`{"message":"hello"}`),
		},
	)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf(
			"Execute() error = %v, want %v",
			err,
			context.Canceled,
		)
	}

	if result != (Result{}) {
		t.Fatalf(
			"Execute() result = %#v, want zero value",
			result,
		)
	}
}

func TestNewExecutorValidation(t *testing.T) {
	executor, err := NewExecutor(nil)

	if err == nil {
		t.Fatal("NewExecutor() returned nil error, want error")
	}

	if executor != nil {
		t.Fatalf("NewExecutor() = %#v, want nil", executor)
	}

	if got, want := err.Error(), "tool registry is required"; got != want {
		t.Fatalf(
			"NewExecutor() error = %q, want %q",
			got,
			want,
		)
	}
}

func TestExecutorExecuteValidation(t *testing.T) {
	executor := newExecutorWithTool(
		t,
		func(
			_ context.Context,
			_ json.RawMessage,
		) (string, error) {
			return "", nil
		},
	)

	tests := []struct {
		name        string
		call        Call
		wantMessage string
	}{
		{
			name: "empty call ID",
			call: Call{
				Name:      "echo",
				Arguments: json.RawMessage(`{}`),
			},
			wantMessage: "tool call ID is required",
		},
		{
			name: "empty tool name",
			call: Call{
				ID:        "call-001",
				Arguments: json.RawMessage(`{}`),
			},
			wantMessage: "tool call name is required",
		},
		{
			name: "empty arguments",
			call: Call{
				ID:   "call-001",
				Name: "echo",
			},
			wantMessage: "tool call arguments are required",
		},
		{
			name: "invalid arguments JSON",
			call: Call{
				ID:        "call-001",
				Name:      "echo",
				Arguments: json.RawMessage(`not json`),
			},
			wantMessage: "tool call arguments must be valid JSON",
		},
		{
			name: "tool not registered",
			call: Call{
				ID:        "call-001",
				Name:      "missing",
				Arguments: json.RawMessage(`{}`),
			},
			wantMessage: `tool "missing" is not registered`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executor.Execute(
				context.Background(),
				tt.call,
			)

			if err == nil {
				t.Fatal("Execute() returned nil error, want error")
			}

			if result != (Result{}) {
				t.Fatalf(
					"Execute() result = %#v, want zero value",
					result,
				)
			}

			if got, want := err.Error(), tt.wantMessage; got != want {
				t.Fatalf(
					"Execute() error = %q, want %q",
					got,
					want,
				)
			}
		})
	}
}

func TestExecutorExecuteSchemaValidation(t *testing.T) {
	handlerCalled := false

	executor := newExecutorWithTool(
		t,
		func(
			_ context.Context,
			_ json.RawMessage,
		) (string, error) {
			handlerCalled = true
			return "executed", nil
		},
	)

	result, err := executor.Execute(
		context.Background(),
		Call{
			ID:   "call-001",
			Name: "echo",
			Arguments: json.RawMessage(
				`{"message":123}`,
			),
		},
	)
	if err != nil {
		t.Fatalf(
			"Execute() returned platform error: %v",
			err,
		)
	}

	if !result.IsError {
		t.Fatalf(
			"Execute() result = %#v, want error result",
			result,
		)
	}

	if handlerCalled {
		t.Fatal(
			"handler was called with invalid arguments",
		)
	}
}
