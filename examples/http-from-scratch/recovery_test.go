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

func TestRecoveryWriterFlushStartsResponse(t *testing.T) {
	response := httptest.NewRecorder()
	writer := &recoveryWriter{ResponseWriter: response}
	if err := http.NewResponseController(writer).Flush(); err != nil {
		t.Fatal(err)
	}
	if !writer.started || !response.Flushed || response.Code != http.StatusOK || response.Body.Len() != 0 {
		t.Fatal("flush without body must commit 200, record started and reach the underlying writer")
	}
	writer.WriteHeader(http.StatusInternalServerError)
	if response.Code != http.StatusOK {
		t.Fatal("cannot replace status after flush")
	}
}

func TestRecoveryWriterFlushErrors(t *testing.T) {
	failure := errors.New("injected flush failure")
	for _, tt := range []struct {
		name string
		next http.ResponseWriter
		want error
	}{
		{"underlying_error", &sseProbeWriter{ResponseWriter: httptest.NewRecorder(), flushErr: failure}, failure},
		{"unsupported", sseBasicWriter{httptest.NewRecorder()}, http.ErrNotSupported},
	} {
		t.Run(tt.name, func(t *testing.T) {
			writer := &recoveryWriter{ResponseWriter: tt.next}
			err := http.NewResponseController(writer).Flush()
			if !errors.Is(err, tt.want) || !writer.started {
				t.Fatalf("flush error=%v started=%v", err, writer.started)
			}
		})
	}
}

func TestRecoveryMiddlewareAfterFlushAborts(t *testing.T) {
	captureRequestLog(t)
	response := httptest.NewRecorder()
	handler := recoveryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := http.NewResponseController(w).Flush(); err != nil {
			panic(err)
		}
		panic("failure after headers flushed")
	}))
	value := caughtPanic(func() { handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/flush", nil)) })
	if value != http.ErrAbortHandler || response.Code != 200 || !response.Flushed || response.Body.Len() != 0 {
		t.Fatal("flush-only panic must abort without appending 500")
	}
}

func TestRecoveryMiddlewareRealHTTPAfterFlush(t *testing.T) {
	output := captureRequestLog(t)
	release := make(chan struct{})
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ping" {
			pingHandler(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		if _, err := io.WriteString(w, firstSSEEvent); err != nil {
			return
		}
		if err := http.NewResponseController(w).Flush(); err != nil {
			return
		}
		select {
		case <-release:
			panic("failure after SSE flush")
		case <-r.Context().Done():
		}
	})
	server := httptest.NewServer(loggingMiddleware(recoveryMiddleware(next)))
	defer server.Close()
	client := server.Client()
	client.Timeout = 3 * time.Second
	response, err := client.Get(server.URL + "/partial")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	close(release)
	body, readErr := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != 200 || string(body) != firstSSEEvent || !errors.Is(readErr, io.ErrUnexpectedEOF) {
		t.Fatalf("expected flushed event and interrupted HTTP/1.1 body, got %d %q %v", response.StatusCode, body, readErr)
	}
	ping, err := client.Get(server.URL + "/ping")
	if err != nil {
		t.Fatal(err)
	}
	pingBody, pingErr := io.ReadAll(ping.Body)
	ping.Body.Close()
	server.Close()
	if ping.StatusCode != 200 || string(pingBody) != "pong\n" || pingErr != nil {
		t.Fatal("server must survive a panic after flushing")
	}
	if !strings.Contains(output.String(), "failure after SSE flush") {
		t.Fatal("panic must still be logged on the server")
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
