package main

import (
	"context"
	"encoding/json"
	"errors"
	"mime"
	"strings"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/ut"
)

func chatStreamTestContext(w *eventTestWriter, body string) *app.RequestContext {
	c := app.NewContext(0)
	c.Request.SetMethod("POST")
	c.Request.SetRequestURI("/chat/stream")
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.SetBodyString(body)
	if w != nil {
		c.Response.HijackWriter(w)
	}
	return c
}

func TestChatStreamHandlerFormatAndSharedCount(t *testing.T) {
	store := newSessionStore()
	store.nextTurn("demo")
	w := &eventTestWriter{}
	c := chatStreamTestContext(w, `{"message":"/upper 你a","session_id":" demo "}`)
	newChatStreamHandler(store)(context.Background(), c)
	want := "event: delta\ndata: {\"content\":\"你\"}\n\n" +
		"event: delta\ndata: {\"content\":\"A\"}\n\n" +
		"event: done\ndata: {\"answer\":\"你A\",\"session_id\":\"demo\",\"turn\":2}\n\n"
	if w.String() != want || w.writes != 3 || w.flushes != 3 || w.finalizes != 0 {
		t.Fatalf("events=%q writes=%d flushes=%d finalizes=%d", w.String(), w.writes, w.flushes, w.finalizes)
	}
	if c.Response.StatusCode() != 200 || string(c.Response.Header.Peek("Content-Type")) != "text/event-stream; charset=utf-8" || string(c.Response.Header.Peek("Cache-Control")) != "no-cache" {
		t.Fatal("unexpected stream headers")
	}
	if store.nextTurn("demo") != 3 {
		t.Fatal("one stream must increment once, not once per delta")
	}
}

func TestChatStreamHandlerContentAndNoSession(t *testing.T) {
	for _, tt := range []struct {
		name, body, answer string
	}{
		{"ordinary", `{"message":"Go"}`, "收到：Go"},
		{"blank_id", `{"message":"/upper g","session_id":" \t "}`, "G"},
		{"null_id", `{"message":"/upper g","session_id":null}`, "G"},
		{"escaped_text", `{"message":"/upper a\"\\\nb"}`, "A\"\\\nB"},
		{"combining_code_points", `{"message":"/upper é"}`, "É"},
		{"exact_limit", `{"message":"/upper g"}` + strings.Repeat(" ", 4096-len(`{"message":"/upper g"}`)), "G"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := &eventTestWriter{}
			c := chatStreamTestContext(w, tt.body)
			newChatStreamHandler(newSessionStore())(context.Background(), c)
			frames := strings.Split(strings.TrimSuffix(w.String(), "\n\n"), "\n\n")
			wantRunes := []rune(tt.answer)
			if len(frames) != len(wantRunes)+1 || w.writes != len(frames) || w.flushes != len(frames) || w.finalizes != 0 {
				t.Fatalf("unexpected event counts: events=%q writes=%d flushes=%d", w.String(), w.writes, w.flushes)
			}
			var joined string
			for i, frame := range frames[:len(frames)-1] {
				if !strings.HasPrefix(frame, "event: delta\ndata: ") {
					t.Fatalf("unexpected delta: %q", frame)
				}
				var got map[string]string
				if err := json.Unmarshal([]byte(strings.TrimPrefix(frame, "event: delta\ndata: ")), &got); err != nil {
					t.Fatal(err)
				}
				if len(got) != 1 || got["content"] != string(wantRunes[i]) {
					t.Fatalf("delta=%v want content=%q", got, string(wantRunes[i]))
				}
				joined += got["content"]
			}
			last := frames[len(frames)-1]
			if !strings.HasPrefix(last, "event: done\ndata: ") {
				t.Fatalf("missing final done: %q", last)
			}
			var done map[string]string
			if err := json.Unmarshal([]byte(strings.TrimPrefix(last, "event: done\ndata: ")), &done); err != nil {
				t.Fatal(err)
			}
			if len(done) != 1 || done["answer"] != tt.answer || joined != tt.answer {
				t.Fatalf("without session, done must contain only full answer: %v joined=%q", done, joined)
			}
		})
	}
}

func TestChatStreamHandlerInputErrorsBeforeSSE(t *testing.T) {
	store := newSessionStore()
	h := newServer()
	for _, tt := range []struct {
		name, body, contentType, want string
		status                        int
	}{
		{"missing_type", `{"message":"Go"}`, "", "Content-Type must be application/json\n", 415},
		{"wrong_type", `{"message":"Go"}`, "text/plain", "Content-Type must be application/json\n", 415},
		{"invalid_type", `{"message":"Go"}`, "application/json; charset", "Content-Type must be application/json\n", 415},
		{"type_before_size", strings.Repeat("x", 4097), "text/plain", "Content-Type must be application/json\n", 415},
		{"too_large", strings.Repeat("x", 4097), "application/json", "request body too large\n", 413},
		{"invalid_json", `{"message":`, "application/json", "invalid JSON request\n", 400},
		{"empty_body", "", "application/json", "invalid JSON request\n", 400},
		{"wrong_message_type", `{"message":1}`, "application/json", "invalid JSON request\n", 400},
		{"wrong_session_type", `{"message":"Go","session_id":1}`, "application/json", "invalid JSON request\n", 400},
		{"two_values", `{"message":"Go"}{}`, "application/json", "invalid JSON request\n", 400},
		{"missing_message", `{}`, "application/json", "message is required\n", 400},
		{"null_body", `null`, "application/json", "message is required\n", 400},
		{"null_message", `{"message":null}`, "application/json", "message is required\n", 400},
		{"empty_message", `{"message":""}`, "application/json", "message is required\n", 400},
		{"blank_message", `{"message":" "}`, "application/json", "message is required\n", 400},
		{"bare_command", `{"message":"/upper","session_id":"error"}`, "application/json", "text is required after /upper\n", 400},
		{"blank_command", `{"message":"/upper   ","session_id":"error"}`, "application/json", "text is required after /upper\n", 400},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// 没有预先提供Writer，确认失败请求根本没有进入SSE模式。
			c := chatStreamTestContext(nil, tt.body)
			c.Request.Header.Set("Content-Type", tt.contentType)
			newChatStreamHandler(store)(context.Background(), c)
			if c.Response.GetHijackWriter() != nil || c.Response.StatusCode() != tt.status || string(c.Response.Body()) != tt.want {
				t.Fatalf("input error must be ordinary response: status=%d body=%q writer=%v", c.Response.StatusCode(), c.Response.Body(), c.Response.GetHijackWriter())
			}
			r := ut.PerformRequest(h.Engine, "POST", "/chat/stream", &ut.Body{Body: strings.NewReader(tt.body), Len: len(tt.body)}, ut.Header{Key: "Content-Type", Value: tt.contentType}).Result()
			if r.StatusCode() != tt.status || string(r.Body()) != tt.want || string(r.Header.Peek("Content-Type")) != "text/plain; charset=utf-8" {
				t.Fatalf("status=%d body=%q type=%q", r.StatusCode(), r.Body(), r.Header.Peek("Content-Type"))
			}
		})
	}
	if store.nextTurn("error") != 1 {
		t.Fatal("failed input must not increment the session")
	}
	for _, method := range []string{"GET", "HEAD"} {
		r := ut.PerformRequest(h.Engine, method, "/chat/stream", nil)
		if r.Code != 405 {
			t.Fatalf("method %s must not enter stream Handler: status=%d", method, r.Code)
		}
	}
}

func TestChatStreamHandlerCancellationAndWriteErrors(t *testing.T) {
	for _, where := range []string{"already_canceled", "after_first", "after_last", "delta_write", "delta_flush", "done_write", "done_flush"} {
		t.Run(where, func(t *testing.T) {
			output := captureHertzRequestLog(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			store := newSessionStore()
			w := &eventTestWriter{}
			body := `{"message":"Go","session_id":"failure"}`
			switch where {
			case "already_canceled":
				cancel()
			case "after_first":
				w.onFlush = cancel
			case "after_last":
				body = `{"message":"/upper g","session_id":"failure"}`
				w.onFlush = cancel
			case "delta_write":
				w.writeErr = errors.New("forced delta write failure")
			case "delta_flush":
				w.flushErr = errors.New("forced delta flush failure")
			case "done_write":
				body = `{"message":"/upper g","session_id":"failure"}`
				w.onFlush = func() { w.writeErr = errors.New("forced done write failure") }
			case "done_flush":
				body = `{"message":"/upper g","session_id":"failure"}`
				w.onFlush = func() {
					if w.flushes == 2 {
						w.flushErr = errors.New("forced done flush failure")
					}
				}
			}
			newChatStreamHandler(store)(ctx, chatStreamTestContext(w, body))
			wantWrites := 1
			if where == "already_canceled" {
				wantWrites = 0
			} else if where == "done_write" || where == "done_flush" {
				wantWrites = 2
			}
			if w.writes != wantWrites || w.finalizes != 0 || strings.Contains(w.String(), "internal server error") {
				t.Fatalf("unexpected failure output: writes=%d finalizes=%d events=%q", w.writes, w.finalizes, w.String())
			}
			// done Flush失败时，字节已写出；不能声称客户端一定没见到done。
			if where != "done_flush" && strings.Contains(w.String(), "event: done") {
				t.Fatalf("failed delta or cancellation must not append done: %q", w.String())
			}
			if strings.HasPrefix(where, "done_") {
				if !strings.Contains(output.String(), "write chat done:") {
					t.Fatal("missing done failure log")
				}
			} else if strings.HasPrefix(where, "delta_") {
				if !strings.Contains(output.String(), "write chat delta:") {
					t.Fatal("missing delta failure log")
				}
			} else if !strings.Contains(output.String(), "chat stream canceled: context canceled") {
				t.Fatal("missing cancellation log")
			}
			if store.nextTurn("failure") != 2 {
				t.Fatal("completed rule processing is counted even when transmission fails")
			}
		})
	}
}

func TestChatStreamHandlerDoneContentType(t *testing.T) {
	w := &eventTestWriter{}
	c := chatStreamTestContext(w, `{"message":"/upper g"}`)
	newChatStreamHandler(newSessionStore())(context.Background(), c)
	mediaType, _, err := mime.ParseMediaType(string(c.Response.Header.Peek("Content-Type")))
	if err != nil || mediaType != "text/event-stream" || strings.Contains(w.String(), "id:") {
		t.Fatal("stream must use SSE content type and no invented event IDs")
	}
}
