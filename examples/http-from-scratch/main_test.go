package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestPingHandler(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/ping", nil)
	response := httptest.NewRecorder()

	pingHandler(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if got := response.Header().Get("Content-Type"); got != "text/plain; charset=utf-8" {
		t.Errorf("Content-Type = %q, want plain UTF-8 text", got)
	}
	if got := response.Body.String(); got != "pong\n" {
		t.Errorf("body = %q, want %q", got, "pong\n")
	}
}

func TestEchoHandlerBodyLimit(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		status int
		output string
	}{
		{"text", "你好，Go HTTP", http.StatusOK, "你好，Go HTTP"},
		{"empty", "", http.StatusOK, ""},
		{"at_limit", strings.Repeat("x", 4096), http.StatusOK, strings.Repeat("x", 4096)},
		{"over_limit", strings.Repeat("x", 4097), http.StatusRequestEntityTooLarge, "request body too large\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/echo", strings.NewReader(tt.input))
			response := httptest.NewRecorder()

			echoHandler(response, request)

			if response.Code != tt.status {
				t.Fatalf("status = %d, want %d", response.Code, tt.status)
			}
			if got := response.Header().Get("Content-Type"); got != "text/plain; charset=utf-8" {
				t.Errorf("Content-Type = %q, want plain UTF-8 text", got)
			}
			if got := response.Body.String(); got != tt.output {
				t.Errorf("response body does not match expected content (got %d bytes, want %d bytes)", len(got), len(tt.output))
			}
		})
	}
}

// 模拟请求体读取失败，不需要依赖不稳定的真实网络断开时机。
type failingBody struct{}

func (failingBody) Read([]byte) (int, error) {
	return 0, errors.New("test request body read failed")
}

func (failingBody) Close() error { return nil }

var _ io.ReadCloser = failingBody{}

func TestEchoHandlerReadFailure(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/echo", nil)
	request.Body = failingBody{}
	response := httptest.NewRecorder()

	echoHandler(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if got := response.Body.String(); got != "cannot read request body\n" {
		t.Errorf("body = %q, want read failure response", got)
	}
}

func TestHelloHandler(t *testing.T) {

	tests := []struct {
		name       string
		target     string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "go",
			target:     "/hello?name=Go",
			wantStatus: http.StatusOK,
			wantBody:   "Hello, Go\n",
		},
		{
			name:       "http",
			target:     "/hello?name=HTTP",
			wantStatus: http.StatusOK,
			wantBody:   "Hello, HTTP\n",
		},
		{
			name:       "missing_name",
			target:     "/hello",
			wantStatus: http.StatusBadRequest,
			wantBody:   "name parameter is required\n",
		},
		{
			name:       "empty_name",
			target:     "/hello?name=",
			wantStatus: http.StatusBadRequest,
			wantBody:   "name parameter is required\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, tt.target, nil)
			recorder := httptest.NewRecorder()

			helloHandler(recorder, request)

			response := recorder.Result()
			defer response.Body.Close()

			body, err := io.ReadAll(response.Body)

			if err != nil {
				t.Fatalf("read response body: %v", err)
			}

			if response.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", response.StatusCode, tt.wantStatus)
			}

			if got := response.Header.Get("Content-Type"); got != "text/plain; charset=utf-8" {
				t.Errorf("Content-Type = %q, want %q", got, "text/plain; charset=utf-8")
			}

			if got := string(body); got != tt.wantBody {
				t.Errorf("body = %q, want %q", got, tt.wantBody)
			}
		})
	}
}

func TestHelloHandlerMissingNameDoesNotExit(t *testing.T) {
	// 在独立测试进程里回归验证，防止未来误用 log.Fatalf 终止整个测试套件。
	if os.Getenv("AGENTHUB_TEST_HELLO_MISSING_NAME") == "1" {
		server := httptest.NewServer(http.HandlerFunc(helloHandler))
		defer server.Close()
		client := server.Client()
		client.Timeout = 2 * time.Second
		for _, check := range []struct {
			path   string
			status int
			body   string
		}{
			{"/hello", http.StatusBadRequest, "name parameter is required\n"},
			{"/hello?name=Go", http.StatusOK, "Hello, Go\n"},
		} {
			response, err := client.Get(server.URL + check.path)
			if err != nil {
				t.Fatalf("GET %s: %v", check.path, err)
			}
			body, readErr := io.ReadAll(response.Body)
			response.Body.Close()
			if readErr != nil {
				t.Fatalf("read %s response: %v", check.path, readErr)
			}
			if response.StatusCode != check.status || string(body) != check.body {
				t.Fatalf("GET %s = (%d, %q), want (%d, %q)", check.path, response.StatusCode, body, check.status, check.body)
			}
		}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestHelloHandlerMissingNameDoesNotExit$")
	command.Env = append(os.Environ(), "AGENTHUB_TEST_HELLO_MISSING_NAME=1")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("missing-name request must return 400 without exiting: %v\n%s", err, output)
	}
}

func TestRoutes(t *testing.T) {
	mux := newMux()

	tests := []struct {
		name       string
		method     string
		target     string
		wantStatus int
		wantBody   string
		wantAllow  string
	}{
		{
			name:       "ping",
			method:     http.MethodGet,
			target:     "/ping",
			wantStatus: http.StatusOK,
			wantBody:   "pong\n",
		},
		{
			name:       "hello",
			method:     http.MethodGet,
			target:     "/hello?name=Go",
			wantStatus: http.StatusOK,
			wantBody:   "Hello, Go\n",
		},
		{
			name:       "hello_missing_name",
			method:     http.MethodGet,
			target:     "/hello",
			wantStatus: http.StatusBadRequest,
			wantBody:   "name parameter is required\n",
		},
		{
			name:       "wrong_method",
			method:     http.MethodPost,
			target:     "/hello?name=Go",
			wantStatus: http.StatusMethodNotAllowed,
			wantBody:   "Method Not Allowed\n",
			wantAllow:  "GET, HEAD",
		},
		{
			name:       "unknown_path",
			method:     http.MethodGet,
			target:     "/missing",
			wantStatus: http.StatusNotFound,
			wantBody:   "404 page not found\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, tt.target, nil)
			recorder := httptest.NewRecorder()

			mux.ServeHTTP(recorder, request)

			response := recorder.Result()
			defer response.Body.Close()

			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatalf("read response body: %v", err)
			}

			if response.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", response.StatusCode, tt.wantStatus)
			}

			if got := string(body); got != tt.wantBody {
				t.Errorf("body = %q, want %q", got, tt.wantBody)
			}

			if got := response.Header.Get("Content-Type"); got != "text/plain; charset=utf-8" {
				t.Errorf("Content-Type = %q, want %q", got, "text/plain; charset=utf-8")
			}

			if got := response.Header.Get("Allow"); got != tt.wantAllow {
				t.Errorf("Allow = %q, want %q", got, tt.wantAllow)
			}
		})
	}

}

func TestRoutesRealHTTP(t *testing.T) {
	// 使用启动入口同一份路由注册和包装，通过随机本地端口核对真实HTTP响应。
	server := httptest.NewServer(loggingMiddleware(recoveryMiddleware(newMux())))
	defer server.Close()
	client := server.Client()
	client.Timeout = 2 * time.Second

	for _, tt := range []struct {
		method string
		path   string
		status int
		body   string
		allow  string
	}{
		{http.MethodGet, "/ping", http.StatusOK, "pong\n", ""},
		{http.MethodGet, "/hello?name=Go", http.StatusOK, "Hello, Go\n", ""},
		{http.MethodGet, "/hello", http.StatusBadRequest, "name parameter is required\n", ""},
		{http.MethodPost, "/hello?name=Go", http.StatusMethodNotAllowed, "Method Not Allowed\n", "GET, HEAD"},
		{http.MethodGet, "/missing", http.StatusNotFound, "404 page not found\n", ""},
	} {
		request, err := http.NewRequest(tt.method, server.URL+tt.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		response, err := client.Do(request)
		if err != nil {
			t.Fatalf("%s %s: %v", tt.method, tt.path, err)
		}
		body, readErr := io.ReadAll(response.Body)
		response.Body.Close()
		if readErr != nil {
			t.Fatalf("read %s response: %v", tt.path, readErr)
		}
		if response.StatusCode != tt.status || string(body) != tt.body || response.Header.Get("Allow") != tt.allow || response.Header.Get("Content-Type") != "text/plain; charset=utf-8" {
			t.Errorf("%s %s: status=%d body=%q Allow=%q Content-Type=%q", tt.method, tt.path, response.StatusCode, body, response.Header.Get("Allow"), response.Header.Get("Content-Type"))
		}
	}
}
