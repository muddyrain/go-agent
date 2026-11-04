package main

import (
	"encoding/json"
	"mime"
	"strings"
	"sync"
	"testing"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
)

func TestChatHandlerResponse(t *testing.T) {
	h := newServer()
	for _, tt := range []struct {
		name, body, wantMessage string
	}{
		{"simple", `{"message":"Go"}`, "Go"},
		{"trim", `{"message":"  你好，Go  "}`, "你好，Go"},
		{"keep_inner_space", `{"message":"  你好 Go  "}`, "你好 Go"},
		{"escaped_text", `{"message":"Go\"\\\nHTTP"}`, "Go\"\\\nHTTP"},
		{"exact_limit", `{"message":"Go"}` + strings.Repeat(" ", 4096-len(`{"message":"Go"}`)), "Go"},
		// 保留当前Unmarshal默认行为，不额外引入严格绑定规则。
		{"unknown_field", `{"message":"Go","extra":true}`, "Go"},
		{"duplicate_field", `{"message":"Old","message":"Go"}`, "Go"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := ut.PerformRequest(h.Engine, "POST", "/chat", &ut.Body{
				Body: strings.NewReader(tt.body), Len: len(tt.body),
			}, ut.Header{Key: "Content-Type", Value: "application/json; charset=utf-8"}).Result()
			if r.StatusCode() != 200 {
				t.Fatalf("status=%d body=%q", r.StatusCode(), r.Body())
			}
			mediaType, _, err := mime.ParseMediaType(string(r.Header.Peek("Content-Type")))
			if err != nil || mediaType != "application/json" {
				t.Fatalf("unexpected Content-Type=%q err=%v", r.Header.Peek("Content-Type"), err)
			}
			// 不用结构体反解，以免大小写宽容掩盖answer标签错误。
			var got map[string]json.RawMessage
			if err := json.Unmarshal(r.Body(), &got); err != nil {
				t.Fatalf("invalid response JSON: %v; body=%q", err, r.Body())
			}
			if len(got) != 1 || got["answer"] == nil {
				t.Fatalf("response must contain exactly lowercase answer key: %s", r.Body())
			}
			var answer string
			if err := json.Unmarshal(got["answer"], &answer); err != nil {
				t.Fatal(err)
			}
			if answer != "收到："+tt.wantMessage {
				t.Errorf("answer=%q want=%q", answer, "收到："+tt.wantMessage)
			}
		})
	}
}

func TestChatHandlerValidation(t *testing.T) {
	h := newServer()
	for _, tt := range []struct {
		name, contentType, body string
		status                  int
		wantBody                string
	}{
		{"missing_type", "", `{"message":"Go"}`, 415, "Content-Type must be application/json\n"},
		{"wrong_type", "text/plain", `{"message":"Go"}`, 415, "Content-Type must be application/json\n"},
		{"invalid_type", "application/json; charset", `{"message":"Go"}`, 415, "Content-Type must be application/json\n"},
		{"too_large", "application/json", strings.Repeat("x", 4097), 413, "request body too large\n"},
		{"type_before_size", "text/plain", strings.Repeat("x", 4097), 415, "Content-Type must be application/json\n"},
		{"invalid_json", "application/json", `{"message":`, 400, "invalid JSON request\n"},
		{"wrong_field_type", "application/json", `{"message":123}`, 400, "invalid JSON request\n"},
		{"empty_body", "application/json", "", 400, "invalid JSON request\n"},
		{"two_values", "application/json", `{"message":"Go"}{}`, 400, "invalid JSON request\n"},
		{"missing_message", "application/json", `{}`, 400, "message is required\n"},
		{"empty_message", "application/json", `{"message":""}`, 400, "message is required\n"},
		{"blank_message", "application/json", `{"message":" \t\n "}`, 400, "message is required\n"},
		{"null_message", "application/json", `{"message":null}`, 400, "message is required\n"},
		{"null_body", "application/json", `null`, 400, "message is required\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := ut.PerformRequest(h.Engine, "POST", "/chat", &ut.Body{
				Body: strings.NewReader(tt.body), Len: len(tt.body),
			}, ut.Header{Key: "Content-Type", Value: tt.contentType}).Result()
			if r.StatusCode() != tt.status || string(r.Body()) != tt.wantBody {
				t.Fatalf("status=%d body=%q want=%d %q", r.StatusCode(), r.Body(), tt.status, tt.wantBody)
			}
			if string(r.Header.Peek("Content-Type")) != "text/plain; charset=utf-8" {
				t.Fatalf("error Content-Type=%q", r.Header.Peek("Content-Type"))
			}
		})
	}
}

func TestChatHandlerCommands(t *testing.T) {
	h := newServer()
	for _, tt := range []struct {
		name, message string
		status        int
		answer, body  string
	}{
		{"normal_unchanged", "  Go  ", 200, "收到：Go", ""},
		{"upper", "/upper Go http", 200, "GO HTTP", ""},
		{"upper_trim", "  /upper   你好 Go  ", 200, "你好 GO", ""},
		{"bare_upper", "/upper", 400, "", "text is required after /upper\n"},
		{"blank_upper", "/upper   ", 400, "", "text is required after /upper\n"},
		{"uppercase_is_not_command", "/uppercase go", 200, "收到：/uppercase go", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(map[string]string{"message": tt.message})
			if err != nil {
				t.Fatal(err)
			}
			r := ut.PerformRequest(h.Engine, "POST", "/chat", &ut.Body{
				Body: strings.NewReader(string(body)), Len: len(body),
			}, ut.Header{Key: "Content-Type", Value: "application/json"}).Result()
			if r.StatusCode() != tt.status {
				t.Fatalf("status=%d body=%q want=%d", r.StatusCode(), r.Body(), tt.status)
			}
			if tt.status == 400 {
				if string(r.Body()) != tt.body || string(r.Header.Peek("Content-Type")) != "text/plain; charset=utf-8" {
					t.Fatalf("unexpected error body=%q type=%q", r.Body(), r.Header.Peek("Content-Type"))
				}
				return
			}
			mediaType, _, err := mime.ParseMediaType(string(r.Header.Peek("Content-Type")))
			if err != nil || mediaType != "application/json" {
				t.Fatalf("type=%q error=%v", r.Header.Peek("Content-Type"), err)
			}
			var got map[string]string
			if err := json.Unmarshal(r.Body(), &got); err != nil {
				t.Fatal(err)
			}
			if len(got) != 1 || got["answer"] != tt.answer {
				t.Fatalf("response=%v want answer=%q", got, tt.answer)
			}
		})
	}
}

func TestChatSessionCounts(t *testing.T) {
	h := newServer()
	for _, tt := range []struct {
		name, body, contentType string
		status                  int
		answer, sessionID       string
		turn                    int
		wantError               string
	}{
		{"legacy", `{"message":"Go"}`, "application/json", 200, "收到：Go", "", 0, ""},
		{"empty_id", `{"message":"Go","session_id":""}`, "application/json", 200, "收到：Go", "", 0, ""},
		{"blank_id", `{"message":"Go","session_id":" \t\n "}`, "application/json", 200, "收到：Go", "", 0, ""},
		{"null_id", `{"message":"Go","session_id":null}`, "application/json", 200, "收到：Go", "", 0, ""},
		{"first", `{"message":"Go","session_id":" a "}`, "application/json", 200, "收到：Go", "a", 1, ""},
		{"second", `{"message":"/upper Go","session_id":"a"}`, "application/json", 200, "GO", "a", 2, ""},
		{"failed_command", `{"message":"/upper","session_id":"a"}`, "application/json", 400, "", "", 0, "text is required after /upper\n"},
		{"blank_message", `{"message":"   ","session_id":"a"}`, "application/json", 400, "", "", 0, "message is required\n"},
		{"wrong_field_type", `{"message":42,"session_id":"a"}`, "application/json", 400, "", "", 0, "invalid JSON request\n"},
		{"wrong_id_type", `{"message":"Go","session_id":42}`, "application/json", 400, "", "", 0, "invalid JSON request\n"},
		{"invalid_json", `{"message":"Go","session_id":"a"`, "application/json", 400, "", "", 0, "invalid JSON request\n"},
		{"wrong_media", `{"message":"Go","session_id":"a"}`, "text/plain", 415, "", "", 0, "Content-Type must be application/json\n"},
		{"too_large", `{"message":"Go","session_id":"a"}` + strings.Repeat(" ", 4097-len(`{"message":"Go","session_id":"a"}`)), "application/json", 413, "", "", 0, "request body too large\n"},
		{"third_after_errors", `{"message":"Next","session_id":"a"}`, "application/json", 200, "收到：Next", "a", 3, ""},
		{"other_id", `{"message":"Go","session_id":"b"}`, "application/json", 200, "收到：Go", "b", 1, ""},
		{"id_case_sensitive", `{"message":"Go","session_id":"A"}`, "application/json", 200, "收到：Go", "A", 1, ""},
		{"fourth", `{"message":"Again","session_id":"a"}`, "application/json", 200, "收到：Again", "a", 4, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := ut.PerformRequest(h.Engine, "POST", "/chat", &ut.Body{
				Body: strings.NewReader(tt.body), Len: len(tt.body),
			}, ut.Header{Key: "Content-Type", Value: tt.contentType}).Result()
			if r.StatusCode() != tt.status {
				t.Fatalf("status=%d want=%d body=%q", r.StatusCode(), tt.status, r.Body())
			}
			if tt.status != 200 {
				if string(r.Body()) != tt.wantError || string(r.Header.Peek("Content-Type")) != "text/plain; charset=utf-8" {
					t.Fatalf("unexpected error body=%q type=%q", r.Body(), r.Header.Peek("Content-Type"))
				}
				return
			}
			mediaType, _, err := mime.ParseMediaType(string(r.Header.Peek("Content-Type")))
			if err != nil || mediaType != "application/json" {
				t.Fatalf("unexpected response Content-Type=%q", r.Header.Peek("Content-Type"))
			}
			// 按实际JSON键反解，避免结构体宽容匹配掩盖字段标签错误。
			var got map[string]json.RawMessage
			if err := json.Unmarshal(r.Body(), &got); err != nil {
				t.Fatal(err)
			}
			var answer, sessionID string
			var turn int
			if err := json.Unmarshal(got["answer"], &answer); err != nil {
				t.Fatal(err)
			}
			if tt.sessionID == "" {
				if len(got) != 1 {
					t.Fatalf("without session, response must contain only answer: %s", r.Body())
				}
			} else {
				if len(got) != 3 || json.Unmarshal(got["session_id"], &sessionID) != nil || json.Unmarshal(got["turn"], &turn) != nil {
					t.Fatalf("expected exact answer/session_id/turn fields: %s", r.Body())
				}
			}
			if answer != tt.answer || sessionID != tt.sessionID || turn != tt.turn {
				t.Fatalf("response=%s want answer=%q session_id=%q turn=%d", r.Body(), tt.answer, tt.sessionID, tt.turn)
			}
		})
	}
}

func TestChatSessionServerIsolation(t *testing.T) {
	first, second := newServer(), newServer()
	for _, tt := range []struct {
		h    *server.Hertz
		want int
	}{{first, 1}, {first, 2}, {second, 1}, {first, 3}} {
		body := `{"message":"Go","session_id":"same"}`
		r := ut.PerformRequest(tt.h.Engine, "POST", "/chat", &ut.Body{
			Body: strings.NewReader(body), Len: len(body),
		}, ut.Header{Key: "Content-Type", Value: "application/json"}).Result()
		var got struct {
			Turn int `json:"turn"`
		}
		if r.StatusCode() != 200 || json.Unmarshal(r.Body(), &got) != nil || got.Turn != tt.want {
			t.Fatalf("store must persist per server, not across servers: status=%d body=%s want turn=%d", r.StatusCode(), r.Body(), tt.want)
		}
	}
}

func TestChatSessionConcurrentRequests(t *testing.T) {
	h := newServer()
	const count = 32
	turns := make(chan int, count)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			body := `{"message":"Go","session_id":"parallel"}`
			r := ut.PerformRequest(h.Engine, "POST", "/chat", &ut.Body{
				Body: strings.NewReader(body), Len: len(body),
			}, ut.Header{Key: "Content-Type", Value: "application/json"}).Result()
			var got chatResponse
			if r.StatusCode() != 200 || json.Unmarshal(r.Body(), &got) != nil || got.Answer != "收到：Go" || got.SessionID != "parallel" {
				t.Errorf("unexpected concurrent response: status=%d body=%s", r.StatusCode(), r.Body())
				return
			}
			turns <- got.Turn
		}()
	}
	close(start)
	wg.Wait()
	close(turns)
	seen := make(map[int]bool)
	for turn := range turns {
		if turn < 1 || turn > count || seen[turn] {
			t.Fatalf("invalid or repeated turn=%d", turn)
		}
		seen[turn] = true
	}
	if len(seen) != count {
		t.Fatalf("unique turns=%d want=%d", len(seen), count)
	}
}

func TestChatRouteMethod(t *testing.T) {
	r := ut.PerformRequest(newServer().Engine, "GET", "/chat", nil)
	if r.Code != 405 || r.Body.String() != "Method Not Allowed" {
		t.Fatalf("GET /chat must not execute the POST Handler: %d %q", r.Code, r.Body.String())
	}
}
