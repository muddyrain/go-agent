package main

import (
	"encoding/json"
	"mime"
	"strings"
	"testing"

	"github.com/cloudwego/hertz/pkg/common/ut"
)

func TestGreetHandlerJSONContract(t *testing.T) {
	h := newServer()
	for _, tt := range []struct {
		name     string
		body     string
		wantName string
	}{
		{"simple", `{"name":"Go"}`, "Go"},
		{"trim_space", `{"name":"  Go  "}`, "Go"},
		{"escaped_text", `{"name":"Go\"\\\nHTTP"}`, "Go\"\\\nHTTP"},
		{"chinese", `{"name":"你好"}`, "你好"},
		{"exact_limit", `{"name":"Go"}` + strings.Repeat(" ", 4096-len(`{"name":"Go"}`)), "Go"},
		// 保留标准库默认语义：未知字段忽略，重复字段以最后一个为准。
		{"unknown_field", `{"name":"Go","extra":true}`, "Go"},
		{"duplicate_field", `{"name":"Old","name":"Go"}`, "Go"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			response := ut.PerformRequest(h.Engine, "POST", "/greet", &ut.Body{
				Body: strings.NewReader(tt.body), Len: len(tt.body),
			}, ut.Header{Key: "Content-Type", Value: "application/json; charset=utf-8"}).Result()
			if response.StatusCode() != 200 {
				t.Fatalf("status=%d body=%q", response.StatusCode(), response.Body())
			}
			mediaType, _, err := mime.ParseMediaType(string(response.Header.Peek("Content-Type")))
			if err != nil || mediaType != "application/json" {
				t.Fatalf("Content-Type=%q err=%v", response.Header.Peek("Content-Type"), err)
			}
			// 用 map 检查精确键名，避免结构体解码大小写宽容掩盖标签错误。
			var got map[string]json.RawMessage
			if err := json.Unmarshal(response.Body(), &got); err != nil {
				t.Fatalf("response is not a JSON object: %v; body=%q", err, response.Body())
			}
			if len(got) != 2 || got["name"] == nil || got["message"] == nil {
				t.Fatalf("JSON must have exactly lowercase message/name keys: %s", response.Body())
			}
			var name, message string
			if err := json.Unmarshal(got["name"], &name); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(got["message"], &message); err != nil {
				t.Fatal(err)
			}
			if name != tt.wantName || message != "Hello, "+tt.wantName {
				t.Errorf("name=%q message=%q, want %q / %q", name, message, tt.wantName, "Hello, "+tt.wantName)
			}
		})
	}
}

func TestGreetHandlerErrors(t *testing.T) {
	h := newServer()
	for _, tt := range []struct {
		name        string
		contentType string
		body        string
		wantStatus  int
		wantBody    string
	}{
		{"missing_type", "", `{"name":"Go"}`, 415, "Content-Type must be application/json\n"},
		{"wrong_type", "text/plain", `{"name":"Go"}`, 415, "Content-Type must be application/json\n"},
		{"invalid_type", "application/json; charset", `{"name":"Go"}`, 415, "Content-Type must be application/json\n"},
		{"too_large", "application/json", strings.Repeat("x", 4097), 413, "request body too large\n"},
		{"type_checked_before_size", "text/plain", strings.Repeat("x", 4097), 415, "Content-Type must be application/json\n"},
		{"broken_json", "application/json", `{"name":`, 400, "invalid JSON request\n"},
		{"wrong_field_type", "application/json", `{"name":123}`, 400, "invalid JSON request\n"},
		{"array_instead_of_object", "application/json", `[]`, 400, "invalid JSON request\n"},
		{"two_json_values", "application/json", `{"name":"Go"}{"name":"HTTP"}`, 400, "invalid JSON request\n"},
		{"trailing_garbage", "application/json", `{"name":"Go"}x`, 400, "invalid JSON request\n"},
		{"empty_body", "application/json", "", 400, "invalid JSON request\n"},
		{"missing_name", "application/json", `{}`, 400, "name is required\n"},
		{"empty_name", "application/json", `{"name":""}`, 400, "name is required\n"},
		{"blank_name", "application/json", `{"name":" \t\n "}`, 400, "name is required\n"},
		{"null_body", "application/json", `null`, 400, "name is required\n"},
		{"null_name", "application/json", `{"name":null}`, 400, "name is required\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			response := ut.PerformRequest(h.Engine, "POST", "/greet", &ut.Body{
				Body: strings.NewReader(tt.body), Len: len(tt.body),
			}, ut.Header{Key: "Content-Type", Value: tt.contentType}).Result()
			if response.StatusCode() != tt.wantStatus || string(response.Body()) != tt.wantBody {
				t.Fatalf("status=%d body=%q, want %d %q", response.StatusCode(), response.Body(), tt.wantStatus, tt.wantBody)
			}
			if string(response.Header.Peek("Content-Type")) != "text/plain; charset=utf-8" {
				t.Errorf("error must remain plain text: %q", response.Header.Peek("Content-Type"))
			}
		})
	}
}

func TestGreetRouteMethod(t *testing.T) {
	response := ut.PerformRequest(newServer().Engine, "GET", "/greet", nil)
	if response.Code != 405 || response.Body.String() != "Method Not Allowed" {
		t.Fatalf("GET /greet must not execute POST Handler: %d %q", response.Code, response.Body.String())
	}
}
