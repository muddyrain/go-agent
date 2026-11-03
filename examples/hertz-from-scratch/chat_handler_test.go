package main

import (
	"encoding/json"
	"mime"
	"strings"
	"testing"

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

func TestChatRouteMethod(t *testing.T) {
	r := ut.PerformRequest(newServer().Engine, "GET", "/chat", nil)
	if r.Code != 405 || r.Body.String() != "Method Not Allowed" {
		t.Fatalf("GET /chat must not execute the POST Handler: %d %q", r.Code, r.Body.String())
	}
}
