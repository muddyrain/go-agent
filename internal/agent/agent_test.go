package agent

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"agenthub/internal/llm"
	"agenthub/internal/tool"
)

type fakeModel struct {
	response  llm.Response
	responses []llm.Response
	err       error
	requests  []llm.Request
}

func (m *fakeModel) Generate(
	_ context.Context,
	request llm.Request,
) (llm.Response, error) {
	requestIndex := len(m.requests)

	m.requests = append(
		m.requests,
		request,
	)

	if requestIndex < len(m.responses) {
		return m.responses[requestIndex], nil
	}

	return m.response, m.err
}

var _ llm.Model = (*fakeModel)(nil)

func newTestAgent(
	t *testing.T,
	model llm.Model,
	registry *tool.Registry,
) *Agent {
	t.Helper()

	agent, err := New(
		model,
		registry,
		4,
	)
	if err != nil {
		t.Fatalf(
			"New() returned error: %v",
			err,
		)
	}

	return agent
}

func registerEchoTool(
	t *testing.T,
	registry *tool.Registry,
) {
	t.Helper()

	echoTool, err := tool.NewFunction(
		tool.Definition{
			Name:        "echo",
			Description: "returns the provided message",
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
		func(
			_ context.Context,
			arguments json.RawMessage,
		) (string, error) {
			return string(arguments), nil
		},
	)
	if err != nil {
		t.Fatalf(
			"NewFunction() returned error: %v",
			err,
		)
	}

	if err := registry.Register(echoTool); err != nil {
		t.Fatalf(
			"Register() returned error: %v",
			err,
		)
	}
}

func TestRunDirectResponse(t *testing.T) {
	model := &fakeModel{
		response: llm.Response{
			Message: llm.AssistantMessage(
				"hello from model",
			),
			FinishReason: "stop",
			Usage: llm.Usage{
				InputTokens:  4,
				OutputTokens: 3,
				TotalTokens:  7,
			},
		},
	}

	registry := tool.NewRegistry()
	agent := newTestAgent(t, model, registry)

	input := []llm.Message{
		llm.UserMessage("hello"),
	}

	result, err := agent.Run(
		context.Background(),
		input,
	)
	if err != nil {
		t.Fatalf(
			"Run() returned error: %v",
			err,
		)
	}

	if got, want := result.FinalMessage.Content, "hello from model"; got != want {
		t.Fatalf(
			"FinalMessage.Content = %q, want %q",
			got,
			want,
		)
	}

	if got, want := result.Steps, 1; got != want {
		t.Fatalf(
			"Steps = %d, want %d",
			got,
			want,
		)
	}

	if got, want := len(result.Messages), 2; got != want {
		t.Fatalf(
			"len(Messages) = %d, want %d",
			got,
			want,
		)
	}

	if got, want := len(model.requests), 1; got != want {
		t.Fatalf(
			"model received %d requests, want %d",
			got,
			want,
		)
	}

	if !reflect.DeepEqual(
		model.requests[0].Messages,
		input,
	) {
		t.Fatalf(
			"model messages = %#v, want %#v",
			model.requests[0].Messages,
			input,
		)
	}

	if !reflect.DeepEqual(
		result.Usage,
		model.response.Usage,
	) {
		t.Fatalf(
			"Usage = %#v, want %#v",
			result.Usage,
			model.response.Usage,
		)
	}

	if got, want := len(input), 1; got != want {
		t.Fatalf(
			"input length = %d, want %d",
			got,
			want,
		)
	}
}

func TestRunRejectsEmptyMessages(t *testing.T) {
	model := &fakeModel{}
	agent := newTestAgent(
		t,
		model,
		tool.NewRegistry(),
	)

	result, err := agent.Run(
		context.Background(),
		nil,
	)

	if err == nil {
		t.Fatal(
			"Run() returned nil error, want error",
		)
	}

	if !reflect.DeepEqual(result, RunResult{}) {
		t.Fatalf(
			"Run() result = %#v, want zero value",
			result,
		)
	}

	if got, want := err.Error(), "messages are required"; got != want {
		t.Fatalf(
			"Run() error = %q, want %q",
			got,
			want,
		)
	}

	if len(model.requests) != 0 {
		t.Fatal(
			"model was called for empty messages",
		)
	}
}

func TestRunModelError(t *testing.T) {
	modelErr := errors.New("model unavailable")

	model := &fakeModel{
		err: modelErr,
	}
	agent := newTestAgent(
		t,
		model,
		tool.NewRegistry(),
	)

	result, err := agent.Run(
		context.Background(),
		[]llm.Message{
			llm.UserMessage("hello"),
		},
	)

	if !errors.Is(err, modelErr) {
		t.Fatalf(
			"Run() error = %v, want wrapped %v",
			err,
			modelErr,
		)
	}

	if !reflect.DeepEqual(result, RunResult{}) {
		t.Fatalf(
			"Run() result = %#v, want zero value",
			result,
		)
	}
}

func TestRunRejectsNonAssistantResponse(t *testing.T) {
	model := &fakeModel{
		response: llm.Response{
			Message: llm.UserMessage(
				"invalid response",
			),
		},
	}
	agent := newTestAgent(
		t,
		model,
		tool.NewRegistry(),
	)

	result, err := agent.Run(
		context.Background(),
		[]llm.Message{
			llm.UserMessage("hello"),
		},
	)

	if err == nil {
		t.Fatal(
			"Run() returned nil error, want error",
		)
	}

	if !reflect.DeepEqual(result, RunResult{}) {
		t.Fatalf(
			"Run() result = %#v, want zero value",
			result,
		)
	}
}

func TestRunSingleToolCall(t *testing.T) {
	// 第一步：创建 Registry
	registry := tool.NewRegistry()

	// 第二步：创建 Echo Tool
	echoTool, err := tool.NewFunction(
		tool.Definition{
			Name:        "echo",
			Description: "returns the provided message",
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
		func(
			_ context.Context,
			arguments json.RawMessage,
		) (string, error) {
			return string(arguments), nil
		},
	)
	if err != nil {
		t.Fatalf(
			"NewFunction() returned error: %v",
			err,
		)
	}

	// 第三步：把 Echo Tool 注册进 Registry
	if err := registry.Register(echoTool); err != nil {
		t.Fatalf(
			"Register() returned error: %v",
			err,
		)
	}

	// 第四步：配置 Fake Model 的两次响应
	model := &fakeModel{
		responses: []llm.Response{
			{
				Message: llm.AssistantToolCalls(
					tool.Call{
						ID:   "call-001",
						Name: "echo",
						Arguments: json.RawMessage(
							`{"message":"hello"}`,
						),
					},
				),
				Usage: llm.Usage{
					InputTokens:  10,
					OutputTokens: 5,
					TotalTokens:  15,
				},
			},
			{
				Message: llm.AssistantMessage(
					"the tool returned hello",
				),
				FinishReason: "stop",
				Usage: llm.Usage{
					InputTokens:  20,
					OutputTokens: 6,
					TotalTokens:  26,
				},
			},
		},
	}

	// 第五步：使用同一个 Registry 创建 Agent
	agent := newTestAgent(
		t,
		model,
		registry,
	)

	result, err := agent.Run(
		context.Background(),
		[]llm.Message{
			llm.UserMessage("echo hello"),
		},
	)
	if err != nil {
		t.Fatalf(
			"Run() returned error: %v",
			err,
		)
	}

	if got, want := result.FinalMessage.Content, "the tool returned hello"; got != want {
		t.Fatalf(
			"FinalMessage.Content = %q, want %q",
			got,
			want,
		)
	}

	if got, want := result.Steps, 2; got != want {
		t.Fatalf(
			"Steps = %d, want %d",
			got,
			want,
		)
	}

	if got, want := len(result.Messages), 4; got != want {
		t.Fatalf(
			"len(Messages) = %d, want %d",
			got,
			want,
		)
	}

	if got, want := len(model.requests), 2; got != want {
		t.Fatalf(
			"model received %d requests, want %d",
			got,
			want,
		)
	}

	toolMessage := result.Messages[2]

	if got, want := toolMessage.Role, llm.RoleTool; got != want {
		t.Fatalf(
			"tool message role = %q, want %q",
			got,
			want,
		)
	}

	if got, want := toolMessage.Name, "echo"; got != want {
		t.Fatalf(
			"tool message name = %q, want %q",
			got,
			want,
		)
	}

	if got, want := toolMessage.ToolCallID, "call-001"; got != want {
		t.Fatalf(
			"tool message call ID = %q, want %q",
			got,
			want,
		)
	}

	if got, want := toolMessage.Content, `{"message":"hello"}`; got != want {
		t.Fatalf(
			"tool message content = %q, want %q",
			got,
			want,
		)
	}

	secondRequest := model.requests[1]

	if got, want := len(secondRequest.Messages), 3; got != want {
		t.Fatalf(
			"second request received %d messages, want %d",
			got,
			want,
		)
	}

	if !reflect.DeepEqual(
		secondRequest.Messages,
		result.Messages[:3],
	) {
		t.Fatalf(
			"second request messages = %#v, want %#v",
			secondRequest.Messages,
			result.Messages[:3],
		)
	}

	for index, request := range model.requests {
		if got, want := len(request.Tools), 1; got != want {
			t.Fatalf(
				"request %d received %d tools, want %d",
				index,
				got,
				want,
			)
		}

		if got, want := request.Tools[0].Name, "echo"; got != want {
			t.Fatalf(
				"request %d tool name = %q, want %q",
				index,
				got,
				want,
			)
		}
	}

	wantUsage := llm.Usage{
		InputTokens:  30,
		OutputTokens: 11,
		TotalTokens:  41,
	}

	if !reflect.DeepEqual(result.Usage, wantUsage) {
		t.Fatalf(
			"Usage = %#v, want %#v",
			result.Usage,
			wantUsage,
		)
	}
}

func TestRunMultipleToolCallsInOneStep(t *testing.T) {
	registry := tool.NewRegistry()
	registerEchoTool(t, registry)

	model := &fakeModel{
		responses: []llm.Response{
			{
				Message: llm.AssistantToolCalls(
					tool.Call{
						ID:        "call-001",
						Name:      "echo",
						Arguments: json.RawMessage(`{"message":"first"}`),
					},
					tool.Call{
						ID:        "call-002",
						Name:      "echo",
						Arguments: json.RawMessage(`{"message":"second"}`),
					},
				),
				Usage: llm.Usage{
					InputTokens:  10,
					OutputTokens: 8,
					TotalTokens:  18,
				},
			},
			{
				Message: llm.AssistantMessage(
					"both tools completed",
				),
				FinishReason: "stop",
				Usage: llm.Usage{
					InputTokens:  20,
					OutputTokens: 4,
					TotalTokens:  24,
				},
			},
		},
	}

	agent := newTestAgent(
		t,
		model,
		registry,
	)

	result, err := agent.Run(
		context.Background(),
		[]llm.Message{
			llm.UserMessage("echo first and second"),
		},
	)
	if err != nil {
		t.Fatalf(
			"Run() returned error: %v",
			err,
		)
	}

	if got, want := result.Steps, 2; got != want {
		t.Fatalf(
			"Steps = %d, want %d",
			got,
			want,
		)
	}

	if got, want := len(result.Messages), 5; got != want {
		t.Fatalf(
			"len(Messages) = %d, want %d",
			got,
			want,
		)
	}

	firstToolMessage := result.Messages[2]
	secondToolMessage := result.Messages[3]

	if got, want := firstToolMessage.ToolCallID, "call-001"; got != want {
		t.Fatalf(
			"first tool call ID = %q, want %q",
			got,
			want,
		)
	}

	if got, want := firstToolMessage.Content, `{"message":"first"}`; got != want {
		t.Fatalf(
			"first tool content = %q, want %q",
			got,
			want,
		)
	}

	if got, want := secondToolMessage.ToolCallID, "call-002"; got != want {
		t.Fatalf(
			"second tool call ID = %q, want %q",
			got,
			want,
		)
	}

	if got, want := secondToolMessage.Content, `{"message":"second"}`; got != want {
		t.Fatalf(
			"second tool content = %q, want %q",
			got,
			want,
		)
	}

	if got, want := result.FinalMessage.Content, "both tools completed"; got != want {
		t.Fatalf(
			"FinalMessage.Content = %q, want %q",
			got,
			want,
		)
	}
}

func TestRunConsecutiveToolCalls(t *testing.T) {
	registry := tool.NewRegistry()
	registerEchoTool(t, registry)

	model := &fakeModel{
		responses: []llm.Response{
			{
				Message: llm.AssistantToolCalls(
					tool.Call{
						ID:        "call-001",
						Name:      "echo",
						Arguments: json.RawMessage(`{"message":"first"}`),
					},
				),
				Usage: llm.Usage{
					InputTokens:  10,
					OutputTokens: 5,
					TotalTokens:  15,
				},
			},
			{
				Message: llm.AssistantToolCalls(
					tool.Call{
						ID:        "call-002",
						Name:      "echo",
						Arguments: json.RawMessage(`{"message":"second"}`),
					},
				),
				Usage: llm.Usage{
					InputTokens:  20,
					OutputTokens: 6,
					TotalTokens:  26,
				},
			},
			{
				Message: llm.AssistantMessage(
					"finished after two tool rounds",
				),
				FinishReason: "stop",
				Usage: llm.Usage{
					InputTokens:  30,
					OutputTokens: 7,
					TotalTokens:  37,
				},
			},
		},
	}

	agent := newTestAgent(
		t,
		model,
		registry,
	)

	result, err := agent.Run(
		context.Background(),
		[]llm.Message{
			llm.UserMessage("run two echo steps"),
		},
	)
	if err != nil {
		t.Fatalf(
			"Run() returned error: %v",
			err,
		)
	}

	if got, want := result.Steps, 3; got != want {
		t.Fatalf(
			"Steps = %d, want %d",
			got,
			want,
		)
	}

	if got, want := len(model.requests), 3; got != want {
		t.Fatalf(
			"model received %d requests, want %d",
			got,
			want,
		)
	}

	if got, want := len(result.Messages), 6; got != want {
		t.Fatalf(
			"len(Messages) = %d, want %d",
			got,
			want,
		)
	}

	if got, want := result.Messages[2].ToolCallID, "call-001"; got != want {
		t.Fatalf(
			"first tool message call ID = %q, want %q",
			got,
			want,
		)
	}

	if got, want := result.Messages[4].ToolCallID, "call-002"; got != want {
		t.Fatalf(
			"second tool message call ID = %q, want %q",
			got,
			want,
		)
	}

	if got, want := result.FinalMessage.Content, "finished after two tool rounds"; got != want {
		t.Fatalf(
			"FinalMessage.Content = %q, want %q",
			got,
			want,
		)
	}

	wantUsage := llm.Usage{
		InputTokens:  60,
		OutputTokens: 18,
		TotalTokens:  78,
	}

	if !reflect.DeepEqual(result.Usage, wantUsage) {
		t.Fatalf(
			"Usage = %#v, want %#v",
			result.Usage,
			wantUsage,
		)
	}
}

func TestRunStopsAtMaximumSteps(t *testing.T) {
	registry := tool.NewRegistry()

	executionCount := 0

	echoTool, err := tool.NewFunction(
		tool.Definition{
			Name:        "echo",
			Description: "returns the provided message",
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
		func(
			_ context.Context,
			arguments json.RawMessage,
		) (string, error) {
			executionCount++
			return string(arguments), nil
		},
	)
	if err != nil {
		t.Fatalf(
			"NewFunction() returned error: %v",
			err,
		)
	}

	if err := registry.Register(echoTool); err != nil {
		t.Fatalf(
			"Register() returned error: %v",
			err,
		)
	}

	model := &fakeModel{
		responses: []llm.Response{
			{
				Message: llm.AssistantToolCalls(
					tool.Call{
						ID:        "call-001",
						Name:      "echo",
						Arguments: json.RawMessage(`{"message":"first"}`),
					},
				),
			},
			{
				Message: llm.AssistantToolCalls(
					tool.Call{
						ID:        "call-002",
						Name:      "echo",
						Arguments: json.RawMessage(`{"message":"second"}`),
					},
				),
			},
		},
	}

	agent, err := New(
		model,
		registry,
		2,
	)
	if err != nil {
		t.Fatalf(
			"New() returned error: %v",
			err,
		)
	}

	result, err := agent.Run(
		context.Background(),
		[]llm.Message{
			llm.UserMessage("keep using tools"),
		},
	)

	if err == nil {
		t.Fatal(
			"Run() returned nil error, want error",
		)
	}

	if got, want := err.Error(), "agent exceeded maximum steps: 2"; got != want {
		t.Fatalf(
			"Run() error = %q, want %q",
			got,
			want,
		)
	}

	if !reflect.DeepEqual(result, RunResult{}) {
		t.Fatalf(
			"Run() result = %#v, want zero value",
			result,
		)
	}

	if got, want := len(model.requests), 2; got != want {
		t.Fatalf(
			"model received %d requests, want %d",
			got,
			want,
		)
	}

	if got, want := executionCount, 1; got != want {
		t.Fatalf(
			"tool executed %d times, want %d",
			got,
			want,
		)
	}
}

func TestNewValidation(t *testing.T) {
	validModel := &fakeModel{}
	validRegistry := tool.NewRegistry()

	tests := []struct {
		name        string
		model       llm.Model
		registry    *tool.Registry
		maxSteps    int
		wantMessage string
	}{
		{
			name:        "nil model",
			model:       nil,
			registry:    validRegistry,
			maxSteps:    4,
			wantMessage: "model is required",
		},
		{
			name:        "nil registry",
			model:       validModel,
			registry:    nil,
			maxSteps:    4,
			wantMessage: "tool registry is required",
		},
		{
			name:        "zero max steps",
			model:       validModel,
			registry:    validRegistry,
			maxSteps:    0,
			wantMessage: "max steps must be greater than zero",
		},
		{
			name:        "negative max steps",
			model:       validModel,
			registry:    validRegistry,
			maxSteps:    -1,
			wantMessage: "max steps must be greater than zero",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := New(
				tt.model,
				tt.registry,
				tt.maxSteps,
			)

			if err == nil {
				t.Fatal(
					"New() returned nil error, want error",
				)
			}

			if agent != nil {
				t.Fatalf(
					"New() = %#v, want nil",
					agent,
				)
			}

			if got, want := err.Error(), tt.wantMessage; got != want {
				t.Fatalf(
					"New() error = %q, want %q",
					got,
					want,
				)
			}
		})
	}
}

func TestRunReturnsToolErrorToModel(t *testing.T) {
	registry := tool.NewRegistry()

	failingTool, err := tool.NewFunction(
		tool.Definition{
			Name:        "failing",
			Description: "always returns an error",
			Parameters: json.RawMessage(`{
				"type": "object"
			}`),
		},
		func(
			_ context.Context,
			_ json.RawMessage,
		) (string, error) {
			return "", errors.New("tool failed")
		},
	)
	if err != nil {
		t.Fatalf(
			"NewFunction() returned error: %v",
			err,
		)
	}

	if err := registry.Register(failingTool); err != nil {
		t.Fatalf(
			"Register() returned error: %v",
			err,
		)
	}

	model := &fakeModel{
		responses: []llm.Response{
			{
				Message: llm.AssistantToolCalls(
					tool.Call{
						ID:        "call-001",
						Name:      "failing",
						Arguments: json.RawMessage(`{}`),
					},
				),
			},
			{
				Message: llm.AssistantMessage(
					"the tool could not complete the request",
				),
			},
		},
	}

	agent := newTestAgent(
		t,
		model,
		registry,
	)

	result, err := agent.Run(
		context.Background(),
		[]llm.Message{
			llm.UserMessage("run failing tool"),
		},
	)
	if err != nil {
		t.Fatalf(
			"Run() returned error: %v",
			err,
		)
	}

	if got, want := result.Steps, 2; got != want {
		t.Fatalf(
			"Steps = %d, want %d",
			got,
			want,
		)
	}

	toolMessage := result.Messages[2]

	if got, want := toolMessage.Role, llm.RoleTool; got != want {
		t.Fatalf(
			"tool message role = %q, want %q",
			got,
			want,
		)
	}

	if got, want := toolMessage.ToolCallID, "call-001"; got != want {
		t.Fatalf(
			"tool message call ID = %q, want %q",
			got,
			want,
		)
	}

	if got, want := toolMessage.Content, "tool failed"; got != want {
		t.Fatalf(
			"tool message content = %q, want %q",
			got,
			want,
		)
	}

	if got, want := result.FinalMessage.Content,
		"the tool could not complete the request"; got != want {
		t.Fatalf(
			"FinalMessage.Content = %q, want %q",
			got,
			want,
		)
	}
}

func TestRunPropagatesToolCancellation(t *testing.T) {
	registry := tool.NewRegistry()

	cancelAwareTool, err := tool.NewFunction(
		tool.Definition{
			Name:        "cancel_aware",
			Description: "returns the context error",
			Parameters: json.RawMessage(`{
				"type": "object"
			}`),
		},
		func(
			ctx context.Context,
			_ json.RawMessage,
		) (string, error) {
			return "", ctx.Err()
		},
	)
	if err != nil {
		t.Fatalf(
			"NewFunction() returned error: %v",
			err,
		)
	}

	if err := registry.Register(cancelAwareTool); err != nil {
		t.Fatalf(
			"Register() returned error: %v",
			err,
		)
	}

	model := &fakeModel{
		responses: []llm.Response{
			{
				Message: llm.AssistantToolCalls(
					tool.Call{
						ID:        "call-001",
						Name:      "cancel_aware",
						Arguments: json.RawMessage(`{}`),
					},
				),
			},
		},
	}

	agent := newTestAgent(
		t,
		model,
		registry,
	)

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	cancel()

	result, err := agent.Run(
		ctx,
		[]llm.Message{
			llm.UserMessage("run tool"),
		},
	)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf(
			"Run() error = %v, want %v",
			err,
			context.Canceled,
		)
	}

	if !reflect.DeepEqual(result, RunResult{}) {
		t.Fatalf(
			"Run() result = %#v, want zero value",
			result,
		)
	}

	if got, want := len(model.requests), 1; got != want {
		t.Fatalf(
			"model received %d requests, want %d",
			got,
			want,
		)
	}
}

func TestRunPropagatesLaterModelError(t *testing.T) {
	registry := tool.NewRegistry()
	registerEchoTool(t, registry)

	laterErr := errors.New(
		"model failed after tool result",
	)

	model := &fakeModel{
		responses: []llm.Response{
			{
				Message: llm.AssistantToolCalls(
					tool.Call{
						ID:   "call-001",
						Name: "echo",
						Arguments: json.RawMessage(
							`{"message":"hello"}`,
						),
					},
				),
			},
		},
		err: laterErr,
	}

	agent := newTestAgent(
		t,
		model,
		registry,
	)

	result, err := agent.Run(
		context.Background(),
		[]llm.Message{
			llm.UserMessage("echo hello"),
		},
	)

	if !errors.Is(err, laterErr) {
		t.Fatalf(
			"Run() error = %v, want wrapped %v",
			err,
			laterErr,
		)
	}

	if !reflect.DeepEqual(result, RunResult{}) {
		t.Fatalf(
			"Run() result = %#v, want zero value",
			result,
		)
	}

	if got, want := len(model.requests), 2; got != want {
		t.Fatalf(
			"model received %d requests, want %d",
			got,
			want,
		)
	}
}
