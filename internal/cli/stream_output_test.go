package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestWriteAssistantStreamCompletesAtEOF(t *testing.T) {
	stream := schema.StreamReaderFromArray([]*schema.Message{
		schema.AssistantMessage("hello ", nil),
		schema.AssistantMessage("world", nil),
	})

	var output strings.Builder
	chunks, err := writeAssistantStream(&output, stream)
	if err != nil {
		t.Fatalf("writeAssistantStream() error = %v", err)
	}
	if chunks != 2 {
		t.Fatalf("writeAssistantStream() chunks = %d, want 2", chunks)
	}
	if got := output.String(); got != "hello world" {
		t.Fatalf("writeAssistantStream() output = %q, want %q", got, "hello world")
	}
}

func TestWriteAssistantStreamReturnsPartialFailure(t *testing.T) {
	stream, writer := schema.Pipe[*schema.Message](2)
	streamErr := errors.New("upstream stream failed")

	writer.Send(schema.AssistantMessage("partial", nil), nil)
	writer.Send(nil, streamErr)

	var output strings.Builder
	chunks, err := writeAssistantStream(&output, stream)
	if !errors.Is(err, streamErr) {
		t.Fatalf("writeAssistantStream() error = %v, want %v", err, streamErr)
	}
	if chunks != 1 {
		t.Fatalf("writeAssistantStream() chunks = %d, want 1", chunks)
	}
	if got := output.String(); got != "partial" {
		t.Fatalf("writeAssistantStream() output = %q, want %q", got, "partial")
	}

	closed := writer.Send(
		schema.AssistantMessage("unexpected", nil),
		nil,
	)
	if !closed {
		t.Fatal("stream writer remained open after reader returned")
	}
}
