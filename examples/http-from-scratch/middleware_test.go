package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 这些测试临时接管标准日志输出，不能并行运行；结束时恢复原设置。
func captureRequestLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var output bytes.Buffer
	writer, flags, prefix := log.Writer(), log.Flags(), log.Prefix()
	log.SetOutput(&output)
	log.SetFlags(0)
	log.SetPrefix("")
	t.Cleanup(func() {
		log.SetOutput(writer)
		log.SetFlags(flags)
		log.SetPrefix(prefix)
	})
	return &output
}

func TestLoggingMiddlewareForwardsAndLogsAfterReturn(t *testing.T) {
	output := captureRequestLog(t)
	request := httptest.NewRequest(http.MethodPost, "/test?secret=not-for-log", strings.NewReader("input"))
	response := httptest.NewRecorder()
	calls := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r != request || w != response {
			t.Error("middleware must forward the same request and writer")
		}
		if output.Len() != 0 {
			t.Error("request log must run after downstream returns")
		}
		body, err := io.ReadAll(r.Body)
		if err != nil || string(body) != "input" {
			t.Errorf("middleware must leave request body readable: %q, %v", body, err)
		}
		w.Header().Set("X-Test", "preserved")
		w.WriteHeader(http.StatusCreated)
		if _, err := io.WriteString(w, "result"); err != nil {
			t.Fatal(err)
		}
	})

	handler := loggingMiddleware(next)
	if calls != 0 || output.Len() != 0 {
		t.Fatal("constructing middleware must not execute downstream or log a request")
	}
	handler.ServeHTTP(response, request)
	if calls != 1 {
		t.Errorf("downstream calls = %d, want 1", calls)
	}
	if response.Code != http.StatusCreated || response.Body.String() != "result" || response.Header().Get("X-Test") != "preserved" {
		t.Error("middleware changed the downstream response")
	}
	text := output.String()
	if !strings.Contains(text, `method=POST path="/test" duration=`) || strings.Count(text, "http request") != 1 {
		t.Errorf("unexpected request log: %q", text)
	}
	if strings.Contains(text, "secret") || strings.Contains(text, "input") {
		t.Errorf("log should not include query or request body: %q", text)
	}
}

func TestLoggingMiddlewareRealHTTPRequest(t *testing.T) {
	output := captureRequestLog(t)
	server := httptest.NewServer(loggingMiddleware(http.HandlerFunc(pingHandler)))
	defer server.Close()

	response, err := server.Client().Get(server.URL + "/ping")
	if err != nil {
		t.Fatal(err)
	}
	body, readErr := io.ReadAll(response.Body)
	response.Body.Close()
	// 等待服务器关闭，确保 Handler 已返回、延迟日志已结束，再读取日志缓冲。
	server.Close()
	if readErr != nil {
		t.Fatal(readErr)
	}
	if response.StatusCode != http.StatusOK || string(body) != "pong\n" {
		t.Fatalf("unexpected response: status=%d body=%q", response.StatusCode, body)
	}
	if !strings.Contains(output.String(), `method=GET path="/ping" duration=`) {
		t.Errorf("missing request log: %q", output.String())
	}
}

func TestLoggingMiddlewareCoversUnmatchedRoutes(t *testing.T) {
	output := captureRequestLog(t)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ping", pingHandler)
	handler := loggingMiddleware(mux)
	for _, tt := range []struct {
		method string
		path   string
		status int
	}{
		{http.MethodGet, "/missing", http.StatusNotFound},
		{http.MethodPost, "/ping", http.StatusMethodNotAllowed},
	} {
		output.Reset()
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(tt.method, tt.path, nil))
		if response.Code != tt.status {
			t.Errorf("%s %s status = %d, want %d", tt.method, tt.path, response.Code, tt.status)
		}
		if strings.Count(output.String(), "http request") != 1 {
			t.Errorf("unmatched route must still produce one request log: %q", output.String())
		}
	}
}
