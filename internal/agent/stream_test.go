package agent

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"agenthub/internal/llm"
	"agenthub/internal/tool"
)

type streamStep struct {
	event llm.StreamEvent
	err   error
}

type scriptedStream struct {
	steps  []streamStep
	index  int
	closed bool
}

func (s *scriptedStream) Recv() (
	llm.StreamEvent,
	error,
) {
	if s.index >= len(s.steps) {
		return llm.StreamEvent{}, io.EOF
	}

	step := s.steps[s.index]
	s.index++

	return step.event, step.err
}

func (s *scriptedStream) Close() error {
	s.closed = true
	return nil
}

var _ llm.Stream = (*scriptedStream)(nil)

type fakeStreamingModel struct {
	streams        []llm.Stream
	streamErrors   []error
	requests       []llm.Request
	generateCalled bool
}

func (m *fakeStreamingModel) Generate(
	_ context.Context,
	_ llm.Request,
) (llm.Response, error) {
	m.generateCalled = true

	return llm.Response{}, errors.New(
		"Generate should not be called",
	)
}

func (m *fakeStreamingModel) Stream(
	_ context.Context,
	request llm.Request,
) (llm.Stream, error) {
	index := len(m.requests)
	m.requests = append(m.requests, request)

	if index < len(m.streamErrors) &&
		m.streamErrors[index] != nil {
		return nil, m.streamErrors[index]
	}

	if index >= len(m.streams) {
		return nil, errors.New(
			"no scripted stream available",
		)
	}

	return m.streams[index], nil
}

var _ llm.StreamingModel = (*fakeStreamingModel)(nil)

func TestRunStreamDirectResponse(t *testing.T) {
	wantResponse := llm.Response{
		Message:      llm.AssistantMessage("你好"),
		FinishReason: "stop",
		Usage: llm.Usage{
			InputTokens:  5,
			OutputTokens: 2,
			TotalTokens:  7,
		},
	}

	stream := &scriptedStream{
		steps: []streamStep{
			{
				event: llm.StreamEvent{
					Type:  llm.StreamEventDelta,
					Delta: "你",
				},
			},
			{
				event: llm.StreamEvent{
					Type:  llm.StreamEventDelta,
					Delta: "好",
				},
			},
			{
				event: llm.StreamEvent{
					Type:     llm.StreamEventDone,
					Response: wantResponse,
				},
			},
		},
	}

	model := &fakeStreamingModel{
		streams: []llm.Stream{
			stream,
		},
	}

	registry := tool.NewRegistry()
	agent := newTestAgent(t, model, registry)

	var content strings.Builder
	var eventCount int

	result, err := agent.RunStream(
		context.Background(),
		[]llm.Message{
			llm.UserMessage("你好"),
		},
		func(event llm.StreamEvent) error {
			eventCount++

			if event.Type == llm.StreamEventDelta {
				content.WriteString(event.Delta)
			}

			return nil
		},
	)
	if err != nil {
		t.Fatalf(
			"RunStream() returned error: %v",
			err,
		)
	}

	if got, want := content.String(), "你好"; got != want {
		t.Fatalf(
			"streamed content = %q, want %q",
			got,
			want,
		)
	}

	if got, want := eventCount, 3; got != want {
		t.Fatalf(
			"handler received %d events, want %d",
			got,
			want,
		)
	}

	if !reflect.DeepEqual(
		result.FinalMessage,
		wantResponse.Message,
	) {
		t.Fatalf(
			"FinalMessage = %#v, want %#v",
			result.FinalMessage,
			wantResponse.Message,
		)
	}

	if got, want := result.Steps, 1; got != want {
		t.Fatalf(
			"Steps = %d, want %d",
			got,
			want,
		)
	}

	if !reflect.DeepEqual(result.Usage, wantResponse.Usage) {
		t.Fatalf(
			"Usage = %#v, want %#v",
			result.Usage,
			wantResponse.Usage,
		)
	}

	if !stream.closed {
		t.Fatal("RunStream() did not close stream")
	}

	if model.generateCalled {
		t.Fatal("RunStream() called Generate")
	}

	if got, want := len(model.requests), 1; got != want {
		t.Fatalf(
			"Stream() received %d requests, want %d",
			got,
			want,
		)
	}
}

func TestRunStreamRejectsNilHandler(t *testing.T) {
	model := &fakeStreamingModel{}
	registry := tool.NewRegistry()
	agent := newTestAgent(t, model, registry)

	_, err := agent.RunStream(
		context.Background(),
		[]llm.Message{
			llm.UserMessage("hello"),
		},
		nil,
	)
	if err == nil {
		t.Fatal("RunStream() returned nil error")
	}

	if got, want := err.Error(), "stream handler is required"; got != want {
		t.Fatalf(
			"RunStream() error = %q, want %q",
			got,
			want,
		)
	}

	if got := len(model.requests); got != 0 {
		t.Fatalf(
			"Stream() received %d requests, want 0",
			got,
		)
	}
}

func TestRunStreamRejectsNonStreamingModel(t *testing.T) {
	model := &fakeModel{}
	registry := tool.NewRegistry()
	agent := newTestAgent(t, model, registry)

	_, err := agent.RunStream(
		context.Background(),
		[]llm.Message{
			llm.UserMessage("hello"),
		},
		func(llm.StreamEvent) error {
			return nil
		},
	)
	if err == nil {
		t.Fatal("RunStream() returned nil error")
	}

	if got, want := err.Error(),
		"model does not support streaming"; got != want {
		t.Fatalf(
			"RunStream() error = %q, want %q",
			got,
			want,
		)
	}
}

func TestRunStreamToolCall(t *testing.T) {
	firstResponse := llm.Response{
		Message: llm.AssistantToolCalls(
			tool.Call{
				ID:   "call-001",
				Name: "echo",
				Arguments: json.RawMessage(
					`{"message":"hello"}`,
				),
			},
		),
		FinishReason: "tool_calls",
		Usage: llm.Usage{
			InputTokens:  5,
			OutputTokens: 3,
			TotalTokens:  8,
		},
	}

	secondResponse := llm.Response{
		Message:      llm.AssistantMessage("echo completed"),
		FinishReason: "stop",
		Usage: llm.Usage{
			InputTokens:  8,
			OutputTokens: 2,
			TotalTokens:  10,
		},
	}

	firstStream := &scriptedStream{
		steps: []streamStep{
			{
				event: llm.StreamEvent{
					Type:     llm.StreamEventDone,
					Response: firstResponse,
				},
			},
		},
	}

	secondStream := &scriptedStream{
		steps: []streamStep{
			{
				event: llm.StreamEvent{
					Type:  llm.StreamEventDelta,
					Delta: "echo ",
				},
			},
			{
				event: llm.StreamEvent{
					Type:  llm.StreamEventDelta,
					Delta: "completed",
				},
			},
			{
				event: llm.StreamEvent{
					Type:     llm.StreamEventDone,
					Response: secondResponse,
				},
			},
		},
	}

	model := &fakeStreamingModel{
		streams: []llm.Stream{
			firstStream,
			secondStream,
		},
	}

	registry := tool.NewRegistry()
	registerEchoTool(t, registry)
	agent := newTestAgent(t, model, registry)

	var deltas strings.Builder
	var doneCount int

	result, err := agent.RunStream(
		context.Background(),
		[]llm.Message{
			llm.UserMessage("echo hello"),
		},
		func(event llm.StreamEvent) error {
			switch event.Type {
			case llm.StreamEventDelta:
				deltas.WriteString(event.Delta)
			case llm.StreamEventDone:
				doneCount++
			}

			return nil
		},
	)
	if err != nil {
		t.Fatalf(
			"RunStream() returned error: %v",
			err,
		)
	}

	if got, want := result.FinalMessage.Content,
		"echo completed"; got != want {
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

	wantUsage := llm.Usage{
		InputTokens:  13,
		OutputTokens: 5,
		TotalTokens:  18,
	}
	if !reflect.DeepEqual(result.Usage, wantUsage) {
		t.Fatalf(
			"Usage = %#v, want %#v",
			result.Usage,
			wantUsage,
		)
	}

	if got, want := deltas.String(),
		"echo completed"; got != want {
		t.Fatalf(
			"streamed deltas = %q, want %q",
			got,
			want,
		)
	}

	if got, want := doneCount, 2; got != want {
		t.Fatalf(
			"handler received %d done events, want %d",
			got,
			want,
		)
	}

	if got, want := len(model.requests), 2; got != want {
		t.Fatalf(
			"Stream() received %d requests, want %d",
			got,
			want,
		)
	}
	// 从这里添加：检查第二次模型请求
	secondRequest := model.requests[1]

	if got, want := len(secondRequest.Messages), 3; got != want {
		t.Fatalf(
			"second request contains %d messages, want %d",
			got,
			want,
		)
	}

	toolMessage := secondRequest.Messages[2]

	if got, want := toolMessage.Role, llm.RoleTool; got != want {
		t.Fatalf(
			"tool message role = %q, want %q",
			got,
			want,
		)
	}

	if got, want := toolMessage.ToolCallID,
		"call-001"; got != want {
		t.Fatalf(
			"ToolCallID = %q, want %q",
			got,
			want,
		)
	}

	// 原来的关闭检查继续保留
	if !firstStream.closed {
		t.Fatal("first stream was not closed")
	}

	if !secondStream.closed {
		t.Fatal("second stream was not closed")
	}
}

func TestRunStreamStartError(t *testing.T) {
	wantErr := errors.New("stream unavailable")

	model := &fakeStreamingModel{
		streamErrors: []error{
			wantErr,
		},
	}

	registry := tool.NewRegistry()
	agent := newTestAgent(t, model, registry)

	_, err := agent.RunStream(
		context.Background(),
		[]llm.Message{
			llm.UserMessage("hello"),
		},
		func(llm.StreamEvent) error {
			return nil
		},
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf(
			"RunStream() error = %v, want %v",
			err,
			wantErr,
		)
	}

	if got, want := len(model.requests), 1; got != want {
		t.Fatalf(
			"Stream() received %d requests, want %d",
			got,
			want,
		)
	}
}

func TestRunStreamPropagatesCancellation(t *testing.T) {
	stream := &scriptedStream{
		steps: []streamStep{
			{
				err: context.Canceled,
			},
		},
	}

	model := &fakeStreamingModel{
		streams: []llm.Stream{
			stream,
		},
	}

	registry := tool.NewRegistry()
	agent := newTestAgent(t, model, registry)

	_, err := agent.RunStream(
		context.Background(),
		[]llm.Message{
			llm.UserMessage("hello"),
		},
		func(llm.StreamEvent) error {
			return nil
		},
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf(
			"RunStream() error = %v, want %v",
			err,
			context.Canceled,
		)
	}

	if !stream.closed {
		t.Fatal("RunStream() did not close stream")
	}
}

func TestRunStreamHandlerError(t *testing.T) {
	wantErr := errors.New("client disconnected")

	stream := &scriptedStream{
		steps: []streamStep{
			{
				event: llm.StreamEvent{
					Type:  llm.StreamEventDelta,
					Delta: "partial response",
				},
			},
		},
	}

	model := &fakeStreamingModel{
		streams: []llm.Stream{
			stream,
		},
	}

	registry := tool.NewRegistry()
	agent := newTestAgent(t, model, registry)

	handlerCalls := 0

	_, err := agent.RunStream(
		context.Background(),
		[]llm.Message{
			llm.UserMessage("hello"),
		},
		func(event llm.StreamEvent) error {
			handlerCalls++
			return wantErr
		},
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf(
			"RunStream() error = %v, want %v",
			err,
			wantErr,
		)
	}

	if got, want := handlerCalls, 1; got != want {
		t.Fatalf(
			"handler called %d times, want %d",
			got,
			want,
		)
	}

	if !stream.closed {
		t.Fatal("RunStream() did not close stream")
	}
}
