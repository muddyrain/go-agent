package llm

import (
	"context"
	"errors"
	"testing"
)

type fakeModel struct {
	response Response
	err      error
	request  Request
}

type contextModel struct{}

func (m *fakeModel) Generate(
	_ context.Context,
	req Request,
) (Response, error) {
	m.request = req
	return m.response, m.err
}

func (contextModel) Generate(
	ctx context.Context,
	_ Request,
) (Response, error) {
	select {
	case <-ctx.Done():
		return Response{}, ctx.Err()
	case <-make(chan struct{}):
		return Response{}, nil
	}
}

var _ Model = contextModel{}

var _ Model = (*fakeModel)(nil)

func TestModelGenerate(t *testing.T) {
	wantResponse := Response{
		Message:      AssistantMessage("hello"),
		FinishReason: "stop",
		Usage: Usage{
			InputTokens:  10,
			OutputTokens: 2,
			TotalTokens:  12,
		},
	}

	model := &fakeModel{
		response: wantResponse,
	}

	req := Request{
		Messages: []Message{
			SystemMessage("you are helpful"),
			UserMessage("hello"),
		},
	}

	gotResponse, err := model.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("Generate() returned error: %v", err)
	}

	if gotResponse != wantResponse {
		t.Fatalf(
			"Generate() response = %#v, want %#v",
			gotResponse,
			wantResponse,
		)
	}

	if len(model.request.Messages) != 2 {
		t.Fatalf(
			"Generate() received %d messages, want 2",
			len(model.request.Messages),
		)
	}

	if got, want := model.request.Messages[1], UserMessage("hello"); got != want {
		t.Fatalf("second message = %#v, want %#v", got, want)
	}
}

func TestModelGenerateError(t *testing.T) {
	wantErr := errors.New("model unavailable")

	model := &fakeModel{
		err: wantErr,
	}

	response, err := model.Generate(
		context.Background(),
		Request{
			Messages: []Message{
				UserMessage("hello"),
			},
		},
	)

	if !errors.Is(err, wantErr) {
		t.Fatalf("Generate() error = %v, want %v", err, wantErr)
	}

	if response != (Response{}) {
		t.Fatalf("Generate() response = %#v, want zero value", response)
	}
}

func TestModelGenerateCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	model := contextModel{}

	response, err := model.Generate(ctx, Request{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf(
			"Generate() error = %v, want %v",
			err,
			context.Canceled,
		)
	}

	if response != (Response{}) {
		t.Fatalf("Generate() response = %#v, want zero value", response)
	}
}
