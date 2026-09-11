package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/schema"
)

type fakeGenerator struct {
	generate func(
		ctx context.Context,
		input []*schema.Message,
		opts ...agent.AgentOption,
	) (*schema.Message, error)
}

func (f fakeGenerator) Generate(
	ctx context.Context,
	input []*schema.Message,
	opts ...agent.AgentOption,
) (*schema.Message, error) {
	return f.generate(ctx, input, opts...)
}

func TestChatHandlerReturnsAgentAnswer(t *testing.T) {
	type contextKey string
	const requestIDKey contextKey = "request-id"

	generator := fakeGenerator{
		generate: func(
			ctx context.Context,
			input []*schema.Message,
			_ ...agent.AgentOption,
		) (*schema.Message, error) {
			if got := ctx.Value(requestIDKey); got != "request-1" {
				t.Fatalf("request context value = %v, want request-1", got)
			}
			if len(input) != 2 {
				t.Fatalf("input messages = %d, want 2", len(input))
			}
			if input[0].Role != schema.System || input[0].Content != "system prompt" {
				t.Fatalf("system message = %#v", input[0])
			}
			if input[1].Role != schema.User || input[1].Content != "hello" {
				t.Fatalf("user message = %#v", input[1])
			}

			return schema.AssistantMessage("world", nil), nil
		},
	}

	request := httptest.NewRequest(
		http.MethodPost,
		"/chat",
		strings.NewReader(`{"message":"  hello  "}`),
	)
	request = request.WithContext(
		context.WithValue(request.Context(), requestIDKey, "request-1"),
	)
	response := httptest.NewRecorder()

	NewHandler(generator, "system prompt").ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if got := response.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	if got := response.Body.String(); got != "{\"answer\":\"world\"}\n" {
		t.Fatalf("body = %q", got)
	}
}

func TestChatHandlerRejectsInvalidRequest(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "invalid JSON",
			body:       `{"message":`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "{\"error\":\"invalid JSON request\"}\n",
		},
		{
			name:       "blank message",
			body:       `{"message":"   "}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "{\"error\":\"message is required\"}\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(
				http.MethodPost,
				"/chat",
				strings.NewReader(tt.body),
			)
			response := httptest.NewRecorder()

			NewHandler(nil, "system prompt").ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, tt.wantStatus)
			}
			if got := response.Body.String(); got != tt.wantBody {
				t.Fatalf("body = %q, want %q", got, tt.wantBody)
			}
		})
	}
}

func TestChatHandlerReturnsBadGatewayWhenAgentFails(t *testing.T) {
	tests := []struct {
		name     string
		response *schema.Message
		err      error
	}{
		{
			name: "agent error",
			err:  errors.New("model unavailable"),
		},
		{
			name: "nil message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := fakeGenerator{
				generate: func(
					context.Context,
					[]*schema.Message,
					...agent.AgentOption,
				) (*schema.Message, error) {
					return tt.response, tt.err
				},
			}
			request := httptest.NewRequest(
				http.MethodPost,
				"/chat",
				strings.NewReader(`{"message":"hello"}`),
			)
			response := httptest.NewRecorder()

			NewHandler(generator, "system prompt").ServeHTTP(response, request)

			if response.Code != http.StatusBadGateway {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusBadGateway)
			}
			if got := response.Body.String(); got != "{\"error\":\"agent execution failed\"}\n" {
				t.Fatalf("body = %q", got)
			}
		})
	}
}
