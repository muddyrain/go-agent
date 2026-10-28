package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func caughtPanic(call func()) (value any) {
	defer func() { value = recover() }()
	call()
	return nil
}

func TestRecoveryMiddlewareBeforeResponse(t *testing.T) {
	output := captureRequestLog(t)
	response := httptest.NewRecorder()
	handler := recoveryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", "999")
		w.Header().Set("Content-Encoding", "gzip")
		panic("test internal failure")
	}))
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/panic", nil))
	if response.Code != http.StatusInternalServerError || response.Body.String() != "internal server error\n" {
		t.Fatalf("expected recovered 500, got %d %q", response.Code, response.Body.String())
	}
	if response.Header().Get("Content-Encoding") != "" || response.Header().Get("Content-Length") != "" {
		t.Error("recovery retained incompatible response headers")
	}
	if response.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
		t.Error("recovery must return plain text, not the original media type")
	}
	if !strings.Contains(output.String(), "test internal failure") || !strings.Contains(output.String(), "goroutine") {
		t.Error("panic value and stack must be recorded on the server")
	}
	if strings.Contains(response.Body.String(), "test internal failure") {
		t.Error("internal failure leaked into client response")
	}
}

func TestRecoveryMiddlewareAfterWriteAborts(t *testing.T) {
	captureRequestLog(t)
	response := httptest.NewRecorder()
	handler := recoveryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := fmt.Fprintf(w, "partial"); err != nil {
			t.Fatal(err)
		}
		panic("test failure after write")
	}))
	value := caughtPanic(func() {
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/partial", nil))
	})
	if value != http.ErrAbortHandler {
		t.Fatalf("panic = %v, want ErrAbortHandler", value)
	}
	if response.Code != http.StatusOK || response.Body.String() != "partial" {
		t.Fatalf("recovery must not append a second response: %d %q", response.Code, response.Body.String())
	}
}

func TestRecoveryMiddlewarePreservesAbortSignal(t *testing.T) {
	output := captureRequestLog(t)
	for _, signal := range []error{http.ErrAbortHandler, fmt.Errorf("abort: %w", http.ErrAbortHandler)} {
		handler := recoveryMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic(signal) }))
		response := httptest.NewRecorder()
		value := caughtPanic(func() { handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/abort", nil)) })
		if value != signal || response.Body.Len() != 0 || output.Len() != 0 {
			t.Fatal("explicit abort must be rethrown without a new response or panic log")
		}
	}
}

func TestRecoveryMiddlewareRealHTTP(t *testing.T) {
	output := captureRequestLog(t)
	partial := strings.Repeat("x", 32*1024)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ping":
			pingHandler(w, r)
		case "/panic":
			panicHandler(w, r)
		default:
			// 大于普通 HTTP 写缓冲，确保客户端能观察到响应已经开始；不要求包装器支持 Flush。
			if _, err := io.WriteString(w, partial); err != nil {
				panic(err)
			}
			panic("test partial response failure")
		}
	})
	server := httptest.NewServer(loggingMiddleware(recoveryMiddleware(next)))
	defer server.Close()
	client := server.Client()
	client.Timeout = 3 * time.Second

	for _, path := range []string{"/panic", "/partial", "/ping"} {
		response, err := client.Get(server.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		body, readErr := io.ReadAll(response.Body)
		response.Body.Close()
		switch path {
		case "/panic":
			if response.StatusCode != 500 || readErr != nil || string(body) != "internal server error\n" {
				t.Fatalf("expected complete 500, got %d %q %v", response.StatusCode, body, readErr)
			}
		case "/partial":
			if response.StatusCode != 200 || readErr == nil || len(body) == 0 || strings.Contains(string(body), "internal server error") {
				t.Fatalf("expected interrupted original response, got status=%d bytes=%d err=%v", response.StatusCode, len(body), readErr)
			}
			if !errors.Is(readErr, io.ErrUnexpectedEOF) {
				t.Fatalf("expected incomplete HTTP/1.1 chunked body, got %v", readErr)
			}
		case "/ping":
			if response.StatusCode != 200 || readErr != nil || string(body) != "pong\n" {
				t.Fatal("server must remain usable after panic requests")
			}
		}
	}
	server.Close()
	if strings.Count(output.String(), "http request method=") != 3 {
		t.Error("outer logging must record all three requests")
	}
}
