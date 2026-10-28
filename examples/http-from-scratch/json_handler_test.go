package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGreetHandlerJSONResponse(t *testing.T) {
	for _, name := range []string{"Go", "  HTTP  ", "带\"引号\"和\\路径\n名字"} {
		t.Run(name, func(t *testing.T) {
			input, err := json.Marshal(greetRequest{Name: name})
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodPost, "/greet", strings.NewReader(string(input)))
			request.Header.Set("Content-Type", "application/json; charset=utf-8")
			response := httptest.NewRecorder()
			greetHandler(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", response.Code)
			}
			if got := response.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", got)
			}
			// 按 JSON 解析响应，而不是只比对看起来像 JSON 的文本。
			var output map[string]string
			if err := json.Unmarshal(response.Body.Bytes(), &output); err != nil {
				t.Fatalf("response is not valid JSON: %v", err)
			}
			want := "Hello, " + strings.TrimSpace(name)
			if len(output) != 2 || output["message"] != want || output["name"] != strings.TrimSpace(name) {
				t.Errorf("response = %v, want exactly message %q and lowercase name %q", output, want, strings.TrimSpace(name))
			}
		})
	}
}

func TestGreetHandlerRejectsInvalidRequests(t *testing.T) {
	for _, tt := range []struct {
		name        string
		contentType string
		body        string
		status      int
	}{
		{"invalid_json", "application/json", `{"name":`, http.StatusBadRequest},
		{"wrong_field_type", "application/json", `{"name":123}`, http.StatusBadRequest},
		{"empty_name", "application/json", `{"name":"  "}`, http.StatusBadRequest},
		{"missing_name", "application/json", `{}`, http.StatusBadRequest},
		{"null", "application/json", `null`, http.StatusBadRequest},
		{"wrong_content_type", "text/plain", `{"name":"Go"}`, http.StatusUnsupportedMediaType},
		{"missing_content_type", "", `{"name":"Go"}`, http.StatusUnsupportedMediaType},
		{"over_limit", "application/json", strings.Repeat("x", 4097), http.StatusRequestEntityTooLarge},
	} {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/greet", strings.NewReader(tt.body))
			request.Header.Set("Content-Type", tt.contentType)
			response := httptest.NewRecorder()
			greetHandler(response, request)
			if response.Code != tt.status {
				t.Errorf("status = %d, want %d; body = %q", response.Code, tt.status, response.Body.String())
			}
		})
	}
}

func TestGreetHandlerReadFailure(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/greet", nil)
	request.Header.Set("Content-Type", "application/json")
	request.Body = failingBody{}
	response := httptest.NewRecorder()
	greetHandler(response, request)
	if response.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", response.Code)
	}
}
