package llm

import (
	"errors"
	"io"
	"reflect"
	"testing"
)

type fakeStreamItem struct {
	event StreamEvent
	err   error
}

type fakeStream struct {
	items    []fakeStreamItem
	index    int
	closeErr error
	closed   bool
}

func (s *fakeStream) Recv() (StreamEvent, error) {
	if s.index >= len(s.items) {
		return StreamEvent{}, io.EOF
	}

	item := s.items[s.index]
	s.index++

	return item.event, item.err
}

func (s *fakeStream) Close() error {
	s.closed = true
	return s.closeErr
}

var _ Stream = (*fakeStream)(nil)

func TestConsumeStream(t *testing.T) {
	wantResponse := Response{
		Message:      AssistantMessage("你好"),
		FinishReason: "stop",
		Usage: Usage{
			InputTokens:  5,
			OutputTokens: 2,
			TotalTokens:  7,
		},
	}

	stream := &fakeStream{
		items: []fakeStreamItem{
			{
				event: StreamEvent{
					Type:  StreamEventDelta,
					Delta: "你",
				},
			},
			{
				event: StreamEvent{
					Type:  StreamEventDelta,
					Delta: "好",
				},
			},
			{
				event: StreamEvent{
					Type:     StreamEventDone,
					Response: wantResponse,
				},
			},
		},
	}

	var gotEvents []StreamEvent

	gotResponse, err := ConsumeStream(
		stream,
		func(event StreamEvent) error {
			gotEvents = append(gotEvents, event)
			return nil
		},
	)
	if err != nil {
		t.Fatalf("ConsumeStream() returned error: %v", err)
	}

	if !reflect.DeepEqual(gotResponse, wantResponse) {
		t.Fatalf(
			"ConsumeStream() response = %#v, want %#v",
			gotResponse,
			wantResponse,
		)
	}

	if got, want := len(gotEvents), 3; got != want {
		t.Fatalf(
			"handler received %d events, want %d",
			got,
			want,
		)
	}

	if !stream.closed {
		t.Fatal("ConsumeStream() did not close stream")
	}
}

func TestConsumeStreamRejectsEOFBeforeDone(t *testing.T) {
	stream := &fakeStream{
		items: []fakeStreamItem{
			{
				event: StreamEvent{
					Type:  StreamEventDelta,
					Delta: "partial",
				},
			},
		},
	}

	_, err := ConsumeStream(
		stream,
		func(StreamEvent) error {
			return nil
		},
	)
	if err == nil {
		t.Fatal("ConsumeStream() returned nil error")
	}

	if !stream.closed {
		t.Fatal("ConsumeStream() did not close stream")
	}
}

func TestConsumeStreamRecvError(t *testing.T) {
	wantErr := errors.New("connection reset")

	stream := &fakeStream{
		items: []fakeStreamItem{
			{
				err: wantErr,
			},
		},
	}

	_, err := ConsumeStream(
		stream,
		func(StreamEvent) error {
			return nil
		},
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf(
			"ConsumeStream() error = %v, want %v",
			err,
			wantErr,
		)
	}

	if !stream.closed {
		t.Fatal("ConsumeStream() did not close stream")
	}
}

func TestConsumeStreamHandlerError(t *testing.T) {
	wantErr := errors.New("output unavailable")

	stream := &fakeStream{
		items: []fakeStreamItem{
			{
				event: StreamEvent{
					Type:  StreamEventDelta,
					Delta: "hello",
				},
			},
		},
	}

	_, err := ConsumeStream(
		stream,
		func(StreamEvent) error {
			return wantErr
		},
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf(
			"ConsumeStream() error = %v, want %v",
			err,
			wantErr,
		)
	}

	if !stream.closed {
		t.Fatal("ConsumeStream() did not close stream")
	}
}

func TestConsumeStreamCloseError(t *testing.T) {
	wantErr := errors.New("close failed")

	stream := &fakeStream{
		items: []fakeStreamItem{
			{
				event: StreamEvent{
					Type: StreamEventDone,
					Response: Response{
						Message: AssistantMessage("done"),
					},
				},
			},
		},
		closeErr: wantErr,
	}

	_, err := ConsumeStream(
		stream,
		func(StreamEvent) error {
			return nil
		},
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf(
			"ConsumeStream() error = %v, want %v",
			err,
			wantErr,
		)
	}
}

func TestConsumeStreamRejectsNilStream(t *testing.T) {
	_, err := ConsumeStream(
		nil,
		func(StreamEvent) error {
			return nil
		},
	)
	if err == nil {
		t.Fatal("ConsumeStream() returned nil error")
	}
}

func TestConsumeStreamRejectsNilHandler(t *testing.T) {
	stream := &fakeStream{}

	_, err := ConsumeStream(stream, nil)
	if err == nil {
		t.Fatal("ConsumeStream() returned nil error")
	}

	if got, want := err.Error(), "stream handler is required"; got != want {
		t.Fatalf(
			"ConsumeStream() error = %q, want %q",
			got,
			want,
		)
	}

	if stream.closed {
		t.Fatal("ConsumeStream() closed stream before validation completed")
	}
}

func TestConsumeStreamRejectsUnknownEvent(t *testing.T) {
	stream := &fakeStream{
		items: []fakeStreamItem{
			{
				event: StreamEvent{
					Type: StreamEventType("unknown"),
				},
			},
		},
	}

	_, err := ConsumeStream(
		stream,
		func(StreamEvent) error {
			t.Fatal("handler should not receive unknown event")
			return nil
		},
	)
	if err == nil {
		t.Fatal("ConsumeStream() returned nil error")
	}

	if !stream.closed {
		t.Fatal("ConsumeStream() did not close stream")
	}
}
