package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const clientGoodStream = "event: delta\ndata: {\"content\":\"你\"}\n\nevent: delta\ndata: {\"content\":\"G\"}\n\nevent: done\ndata: {\"answer\":\"你G\"}\n\n"
const clientSessionStream = "event: delta\ndata: {\"content\":\"G\"}\n\nevent: done\ndata: {\"answer\":\"G\",\"session_id\":\"demo\",\"turn\":1}\n\n"

func TestReadChatStream(t *testing.T) {
	for _, tt := range []struct {
		name, input, want, wantError string
	}{
		{"normal", clientGoodStream, "你G", ""},
		{"crlf", strings.ReplaceAll(clientGoodStream, "\n", "\r\n"), "你G", ""},
		{"blank_lines", "\n\n" + clientGoodStream + "\n", "你G", ""},
		{"json_escaping", "event: delta\ndata: {\"content\":\"\\n\\\"\\\\\"}\n\nevent: done\ndata: {\"answer\":\"\\n\\\"\\\\\"}\n\n", "\n\"\\", ""},
		{"empty_stream", "", "", "SSE ended without done"},
		{"no_done", "event: delta\ndata: {\"content\":\"G\"}\n\n", "G", "SSE ended without done"},
		{"partial_frame", "event: delta\ndata: {\"content\":\"G\"}\n", "", "SSE ended before event delimiter"},
		{"partial_done", "event: delta\ndata: {\"content\":\"G\"}\n\nevent: done\ndata: {\"answer\":\"G\"}", "G", "SSE ended before event delimiter"},
		{"bad_delta_json", "event: delta\ndata: {\n\n", "", "decode delta:"},
		{"bad_done_json", "event: done\ndata: {\n\n", "", "decode done:"},
		{"empty_delta", "event: delta\ndata: {}\n\n", "", "delta content is empty"},
		{"empty_done", "event: done\ndata: {}\n\n", "", "done answer does not match deltas"},
		{"mismatch", "event: delta\ndata: {\"content\":\"G\"}\n\nevent: done\ndata: {\"answer\":\"X\"}\n\n", "G", "done answer does not match deltas"},
		{"after_done", clientGoodStream + "event: delta\ndata: {\"content\":\"X\"}\n\n", "你G", "received event after done"},
		{"missing_event", "data: {}\n\n", "", "unexpected event:"},
		{"missing_data", "event: delta\n\n", "", "decode delta:"},
		{"unknown_event", "event: other\ndata: {}\n\n", "", "unexpected event:"},
		{"unsupported_line", "retry: 500\n", "", "unexpected SSE line:"},
		{"line_too_long", strings.Repeat("x", bufio.MaxScanTokenSize+1) + "\n", "", "read SSE:"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			result, err := readChatStream(strings.NewReader(tt.input), &out)
			if out.String() != tt.want {
				t.Fatalf("output=%q want=%q", out.String(), tt.want)
			}
			if tt.wantError == "" {
				if err != nil || result.Answer != tt.want {
					t.Fatalf("result=%v error=%v", result, err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tt.wantError) || result != (streamResult{}) {
				t.Fatalf("result=%v error=%v want=%q", result, err, tt.wantError)
			}
		})
	}
}

// Read的片段边界可落在行、JSON或UTF-8编码内部；客户端不能依赖它等于事件边界。
type clientOneByteReader struct{ r io.Reader }

func (r clientOneByteReader) Read(p []byte) (int, error) {
	if len(p) > 1 {
		p = p[:1]
	}
	return r.r.Read(p)
}

func TestReadChatStreamAcrossByteBoundaries(t *testing.T) {
	var out bytes.Buffer
	result, err := readChatStream(clientOneByteReader{strings.NewReader(clientGoodStream)}, &out)
	if err != nil || result.Answer != "你G" || out.String() != "你G" {
		t.Fatalf("result=%v output=%q error=%v", result, out.String(), err)
	}
}

type clientBrokenReader struct{ err error }

func (r clientBrokenReader) Read([]byte) (int, error) { return 0, r.err }

type clientBrokenWriter struct{ err error }

func (w clientBrokenWriter) Write([]byte) (int, error) { return 0, w.err }

func TestReadChatStreamIOErrors(t *testing.T) {
	failure := errors.New("forced failure")
	for _, tt := range []struct {
		name string
		r    io.Reader
	}{
		{"before_event", clientBrokenReader{failure}},
		{"after_done", io.MultiReader(strings.NewReader(clientGoodStream), clientBrokenReader{failure})},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := readChatStream(tt.r, io.Discard)
			if !errors.Is(err, failure) || !strings.Contains(err.Error(), "read SSE:") {
				t.Fatalf("read wrapping lost: %v", err)
			}
		})
	}
	_, err := readChatStream(strings.NewReader(clientGoodStream), clientBrokenWriter{failure})
	if !errors.Is(err, failure) || !strings.Contains(err.Error(), "write delta:") {
		t.Fatalf("write wrapping lost: %v", err)
	}
}

func TestSendChatStreamHTTP(t *testing.T) {
	noSession := "event: delta\ndata: {\"content\":\"G\"}\n\nevent: done\ndata: {\"answer\":\"G\"}\n\n"
	for _, tt := range []struct {
		name, contentType, body, session, want, wantError string
		status                                            int
	}{
		{"normal", "text/event-stream; charset=utf-8", clientSessionStream, "demo", "回答：G\n完成\n会话：demo，第 1 次\n", "", 200},
		{"trim_session", "text/event-stream", clientSessionStream, " demo ", "回答：G\n完成\n会话：demo，第 1 次\n", "", 200},
		{"no_session", "text/event-stream", noSession, "", "回答：G\n完成\n", "", 200},
		{"blank_session", "text/event-stream", noSession, " \t ", "回答：G\n完成\n", "", 200},
		{"http_400", "text/plain", "message is required\n", "", "", "HTTP 400: message is required", 400},
		{"http_500", "text/plain", "internal server error\n", "", "", "HTTP 500: internal server error", 500},
		{"wrong_type", "application/json", "{}", "", "", "unexpected Content-Type:", 200},
		{"invalid_type", "text/event-stream; charset=", "", "", "", "unexpected Content-Type:", 200},
		{"wrong_session", "text/event-stream", strings.ReplaceAll(clientSessionStream, "demo", "other"), "demo", "回答：G", "done session_id does not match request", 200},
		{"zero_turn", "text/event-stream", strings.ReplaceAll(clientSessionStream, "\"turn\":1", "\"turn\":0"), "demo", "回答：G", "invalid session turn", 200},
		{"negative_turn", "text/event-stream", strings.ReplaceAll(clientSessionStream, "\"turn\":1", "\"turn\":-1"), "demo", "回答：G", "invalid session turn", 200},
		{"turn_without_session", "text/event-stream", strings.ReplaceAll(noSession, "\"answer\":\"G\"", "\"answer\":\"G\",\"turn\":1"), "", "回答：G", "invalid session turn", 200},
		{"no_done", "text/event-stream", "event: delta\ndata: {\"content\":\"G\"}\n\n", "", "回答：G", "SSE ended without done", 200},
		{"bounded_error_body", "text/plain", strings.Repeat("x", 4097), "", "", "HTTP 400: " + strings.Repeat("x", 4096), 400},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Accept") != "text/event-stream" {
					t.Error("wrong outgoing request method or headers")
				}
				var request map[string]any
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
				}
				wantFields := 1
				if tt.session != "" {
					wantFields++
				}
				if request["message"] != "Go" || len(request) != wantFields || tt.session != "" && request["session_id"] != tt.session {
					t.Errorf("request=%v", request)
				}
				w.Header().Set("Content-Type", tt.contentType)
				w.WriteHeader(tt.status)
				_, _ = io.WriteString(w, tt.body)
			}))
			defer server.Close()
			var out bytes.Buffer
			err := sendChatStream(server.Client(), server.URL, "Go", tt.session, &out)
			if out.String() != tt.want {
				t.Fatalf("output=%q want=%q", out.String(), tt.want)
			}
			if tt.wantError == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tt.wantError) || strings.Contains(out.String(), "完成") {
				t.Fatalf("output=%q error=%v want=%q", out.String(), err, tt.wantError)
			}
			if tt.name == "bounded_error_body" && err.Error() != tt.wantError {
				t.Fatalf("error body exceeded limit: %d", len(err.Error()))
			}
		})
	}
}

type clientRoundTripFunc func(*http.Request) (*http.Response, error)

func (f clientRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type clientTrackedBody struct {
	io.Reader
	closed bool
}

func (b *clientTrackedBody) Close() error { b.closed = true; return nil }

type clientFailAtWriter struct {
	calls, failAt int
	err           error
}

func (w *clientFailAtWriter) Write(p []byte) (int, error) {
	w.calls++
	if w.calls == w.failAt {
		return 0, w.err
	}
	return len(p), nil
}

func TestSendChatStreamClosesBody(t *testing.T) {
	failure := errors.New("forced failure")
	for _, tt := range []struct {
		name, mediaType, wantError string
		status                     int
		reader                     io.Reader
		failAt                     int
		wrapped                    bool
	}{
		{"success", "text/event-stream", "", 200, strings.NewReader(clientSessionStream), 0, false},
		{"http_error", "text/plain", "HTTP 400:", 400, strings.NewReader("bad input"), 0, false},
		{"error_body_read", "text/plain", "read HTTP error body:", 400, clientBrokenReader{failure}, 0, true},
		{"wrong_media", "application/json", "unexpected Content-Type:", 200, strings.NewReader("{}"), 0, false},
		{"read_failure", "text/event-stream", "read SSE:", 200, clientBrokenReader{failure}, 0, true},
		{"prefix_write", "text/event-stream", "write answer prefix:", 200, strings.NewReader(clientSessionStream), 1, true},
		{"delta_write", "text/event-stream", "write delta:", 200, strings.NewReader(clientSessionStream), 2, true},
		{"completion_write", "text/event-stream", "write completion:", 200, strings.NewReader(clientSessionStream), 3, true},
		{"session_write", "text/event-stream", "write session result:", 200, strings.NewReader(clientSessionStream), 4, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			body := &clientTrackedBody{Reader: tt.reader}
			client := &http.Client{Transport: clientRoundTripFunc(func(r *http.Request) (*http.Response, error) {
				_ = r.Body.Close()
				return &http.Response{StatusCode: tt.status, Header: http.Header{"Content-Type": {tt.mediaType}}, Body: body, Request: r}, nil
			})}
			out := &clientFailAtWriter{failAt: tt.failAt, err: failure}
			err := sendChatStream(client, "http://example.test/chat/stream", "Go", "demo", out)
			if !body.closed {
				t.Fatal("response body not closed before function returned")
			}
			if tt.wantError == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("error=%v want=%q", err, tt.wantError)
			}
			if tt.wrapped && !errors.Is(err, failure) {
				t.Fatalf("underlying error lost: %v", err)
			}
			if tt.failAt > 0 && out.calls != tt.failAt {
				t.Fatalf("continued writing after error: calls=%d", out.calls)
			}
		})
	}
}

func TestSendChatStreamRequestErrors(t *testing.T) {
	failure := errors.New("forced transport failure")
	client := &http.Client{Transport: clientRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		_ = r.Body.Close()
		return nil, failure
	})}
	var out bytes.Buffer
	err := sendChatStream(client, "http://example.test/chat/stream", "Go", "", &out)
	if !errors.Is(err, failure) || !strings.Contains(err.Error(), "send chat request:") || out.Len() != 0 {
		t.Fatalf("transport error=%v output=%q", err, out.String())
	}
	err = sendChatStream(client, "http://[invalid", "Go", "", &out)
	if err == nil || !strings.Contains(err.Error(), "create chat request:") || out.Len() != 0 {
		t.Fatalf("create error=%v output=%q", err, out.String())
	}
}

// 测试专用并发夹具：控制片段到达和EOF，不给客户端业务增加goroutine。
func TestReadChatStreamDisplaysBeforeEOFAndWaitsAfterDone(t *testing.T) {
	reader, writer := io.Pipe()
	seen := make(chan string, 2)
	finished := make(chan error, 1)
	go func() {
		_, err := readChatStream(reader, clientNotifyWriter{seen})
		finished <- err
	}()
	defer reader.Close()
	defer writer.Close()
	if _, err := io.WriteString(writer, "event: delta\ndata: {\"content\":\"G\"}\n\n"); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-seen:
		if got != "G" {
			t.Fatalf("got=%q", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("delta not displayed before EOF")
	}
	if _, err := io.WriteString(writer, "event: done\ndata: {\"answer\":\"G\"}\n\n"); err != nil {
		t.Fatal(err)
	}
	// 写入done后继续喂一个空行：若实现提前返回，不再读，Pipe写入无法完成。
	// Set定时关闭只防止错误实现把测试挂死，正常执行不会等定时器。
	guard := time.AfterFunc(2*time.Second, func() { _ = writer.CloseWithError(errors.New("reader stopped before EOF")) })
	defer guard.Stop()
	if _, err := io.WriteString(writer, "\n"); err != nil {
		t.Fatalf("reader stopped after done: %v", err)
	}
	select {
	case err := <-finished:
		t.Fatalf("returned before EOF: %v", err)
	default:
	}
	_ = writer.Close()
	select {
	case err := <-finished:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("reader did not return after EOF")
	}
}

type clientNotifyWriter struct{ seen chan string }

func (w clientNotifyWriter) Write(p []byte) (int, error) {
	w.seen <- string(p)
	return len(p), nil
}
