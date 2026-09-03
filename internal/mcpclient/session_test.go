package mcpclient

import (
	"context"
	"encoding/json"
	"testing"
)

type stubSession struct {
	tools      []ToolDefinition
	callResult CallResult
	err        error
	closed     bool
}

func (s *stubSession) ListTools(
	ctx context.Context,
) ([]ToolDefinition, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return s.tools, s.err
	}
}

func (s *stubSession) CallTool(
	ctx context.Context,
	_ string,
	_ json.RawMessage,
) (CallResult, error) {
	select {
	case <-ctx.Done():
		return CallResult{}, ctx.Err()
	default:
		return s.callResult, s.err
	}
}

func (s *stubSession) Close() error {
	s.closed = true
	return nil
}

var _ Session = (*stubSession)(nil)

func TestSessionListTools(t *testing.T) {
	session := &stubSession{
		tools: []ToolDefinition{
			{
				Name:        "echo",
				Description: "returns the provided message",
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
		},
	}

	tools, err := session.ListTools(
		context.Background(),
	)
	if err != nil {
		t.Fatalf(
			"ListTools() returned error: %v",
			err,
		)
	}

	if got, want := len(tools), 1; got != want {
		t.Fatalf(
			"len(ListTools()) = %d, want %d",
			got,
			want,
		)
	}

	if got, want := tools[0].Name, "echo"; got != want {
		t.Fatalf(
			"tool name = %q, want %q",
			got,
			want,
		)
	}

	if !json.Valid(tools[0].InputSchema) {
		t.Fatal("tool input schema is not valid JSON")
	}
}

func TestSessionCallTool(t *testing.T) {
	session := &stubSession{
		callResult: CallResult{
			Content: []Content{
				{
					Type: ContentTypeText,
					Text: "hello",
				},
			},
			IsError: false,
		},
	}

	result, err := session.CallTool(
		context.Background(),
		"echo",
		json.RawMessage(`{"message":"hello"}`),
	)
	if err != nil {
		t.Fatalf(
			"CallTool() returned error: %v",
			err,
		)
	}

	if result.IsError {
		t.Fatal("CallTool() IsError = true, want false")
	}

	if got, want := len(result.Content), 1; got != want {
		t.Fatalf(
			"len(CallTool().Content) = %d, want %d",
			got,
			want,
		)
	}

	if got, want := result.Content[0].Text,
		"hello"; got != want {
		t.Fatalf(
			"result text = %q, want %q",
			got,
			want,
		)
	}
}

func TestSessionCallToolCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	cancel()

	session := &stubSession{}

	_, err := session.CallTool(
		ctx,
		"echo",
		json.RawMessage(`{"message":"hello"}`),
	)
	if err != context.Canceled {
		t.Fatalf(
			"CallTool() error = %v, want %v",
			err,
			context.Canceled,
		)
	}
}

func TestSessionClose(t *testing.T) {
	session := &stubSession{}

	if err := session.Close(); err != nil {
		t.Fatalf(
			"Close() returned error: %v",
			err,
		)
	}

	if !session.closed {
		t.Fatal("Close() did not mark session as closed")
	}
}
