package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSlowHandlerAlreadyCanceled(t *testing.T) {
	output := captureRequestLog(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/slow", nil).WithContext(ctx)
	start := time.Now()
	slowHandler(response, request)
	if elapsed := time.Since(start); elapsed >= time.Second {
		t.Fatalf("canceled request kept waiting: %s", elapsed)
	}
	if response.Body.Len() != 0 || response.Header().Get("Content-Type") != "" {
		t.Fatal("canceled handler must not write a success response")
	}
	if !strings.Contains(output.String(), "slow work canceled: context canceled") || strings.Contains(output.String(), "slow work completed") {
		t.Fatalf("unexpected cancellation log: %q", output.String())
	}
}

func TestSlowHandlerCanceledDuringWait(t *testing.T) {
	output := captureRequestLog(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/slow", nil).WithContext(ctx)
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		slowHandler(response, request)
	}()
	// 只在测试中触发取消；生产 Handler 无需新增 goroutine。
	time.Sleep(30 * time.Millisecond)
	cancel()
	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		// 等待 Handler 退出再读取日志或让 cleanup 恢复全局日志设置。
		<-finished
		t.Fatal("handler ignored cancellation while waiting")
	}
	if response.Body.Len() != 0 || !strings.Contains(output.String(), "slow work canceled: context canceled") {
		t.Fatal("waiting handler must return without success body when canceled")
	}
}

func TestSlowHandlerRealHTTP(t *testing.T) {
	output := captureRequestLog(t)
	type result struct {
		err     error
		elapsed time.Duration
	}
	entered := make(chan struct{}, 2)
	finished := make(chan result, 2)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ping", pingHandler)
	mux.HandleFunc("GET /slow", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		entered <- struct{}{}
		slowHandler(w, r)
		finished <- result{r.Context().Err(), time.Since(start)}
	})
	server := httptest.NewServer(loggingMiddleware(recoveryMiddleware(mux)))
	defer server.Close()
	client := server.Client()
	client.Timeout = 10 * time.Second

	response, err := client.Get(server.URL + "/slow")
	if err != nil {
		t.Fatal(err)
	}
	body, readErr := io.ReadAll(response.Body)
	response.Body.Close()
	<-entered
	normal := <-finished
	if readErr != nil || response.StatusCode != http.StatusOK || string(body) != "slow work completed\n" || response.Header.Get("Content-Type") != "text/plain; charset=utf-8" {
		t.Fatalf("normal response: status=%d body=%q err=%v", response.StatusCode, body, readErr)
	}
	if normal.err != nil || normal.elapsed < 5*time.Second {
		t.Fatalf("normal handler ended unexpectedly: %+v", normal)
	}
	t.Logf("normal request: HTTP 200 after %s", normal.elapsed)

	// 真实 TCP 连接先发送请求，确认 Handler 进入后关闭客户端连接。
	conn, err := net.DialTimeout("tcp", server.Listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := fmt.Fprintf(conn, "GET /slow HTTP/1.1\r\nHost: localhost\r\n\r\n"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("slow request did not reach the server")
	}
	closedAt := time.Now()
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case canceled := <-finished:
		if !errors.Is(canceled.err, context.Canceled) || canceled.elapsed >= 5*time.Second {
			t.Fatalf("disconnect did not cancel work: %+v", canceled)
		}
		t.Logf("client disconnect: handler returned after %s with %v", time.Since(closedAt), canceled.err)
	case <-time.After(2 * time.Second):
		t.Fatal("disconnected request did not stop promptly")
	}

	response, err = client.Get(server.URL + "/ping")
	if err != nil {
		t.Fatal(err)
	}
	body, readErr = io.ReadAll(response.Body)
	response.Body.Close()
	if readErr != nil || response.StatusCode != http.StatusOK || string(body) != "pong\n" {
		t.Fatal("server did not remain usable after cancellation")
	}
	// 关闭并等待服务器后才读取日志缓冲，避免与 Handler 的 defer 日志并发读写。
	server.Close()
	text := output.String()
	if strings.Count(text, "slow work canceled: context canceled") != 1 || strings.Count(text, "slow work completed") != 1 || strings.Contains(text, "http panic") {
		t.Fatalf("unexpected completion/cancellation logs: %q", text)
	}
}

func TestSlowTimeoutHandlerParentContext(t *testing.T) {
	for _, tt := range []struct {
		name      string
		deadline  bool
		wantBody  string
		wantLog   string
		wantError error
	}{
		{"already_canceled", false, "", "slow work canceled: context canceled", context.Canceled},
		{"earlier_deadline", true, "slow work timed out\n", "slow work timed out: context deadline exceeded", context.DeadlineExceeded},
	} {
		t.Run(tt.name, func(t *testing.T) {
			output := captureRequestLog(t)
			var ctx context.Context
			var cancel context.CancelFunc
			if tt.deadline {
				ctx, cancel = context.WithTimeout(context.Background(), 30*time.Millisecond)
			} else {
				ctx, cancel = context.WithCancel(context.Background())
				cancel()
			}
			defer cancel()
			request := httptest.NewRequest(http.MethodGet, "/slow/timeout", nil).WithContext(ctx)
			response := httptest.NewRecorder()
			start := time.Now()
			slowTimeoutHandler(response, request)
			if elapsed := time.Since(start); elapsed >= time.Second {
				t.Fatalf("parent cancellation/deadline was not inherited: %s", elapsed)
			}
			if !errors.Is(ctx.Err(), tt.wantError) || response.Body.String() != tt.wantBody || !strings.Contains(output.String(), tt.wantLog) {
				t.Fatalf("status=%d body=%q parent=%v log=%q", response.Code, response.Body.String(), ctx.Err(), output.String())
			}
			if tt.deadline && response.Code != http.StatusServiceUnavailable {
				t.Fatalf("deadline status=%d, want 503", response.Code)
			}
			if !tt.deadline && response.Header().Get("Content-Type") != "" {
				t.Fatal("cancellation must not write an error response")
			}
		})
	}
}

func TestSlowTimeoutHandlerRealHTTP(t *testing.T) {
	output := captureRequestLog(t)
	type result struct {
		parentErr error
		elapsed   time.Duration
	}
	entered := make(chan struct{}, 2)
	finished := make(chan result, 2)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ping", pingHandler)
	mux.HandleFunc("GET /slow/timeout", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		entered <- struct{}{}
		slowTimeoutHandler(w, r)
		// 在外层 Handler 返回前观察父 Context；子超时不应取消父请求。
		finished <- result{r.Context().Err(), time.Since(start)}
	})
	server := httptest.NewServer(loggingMiddleware(recoveryMiddleware(mux)))
	defer server.Close()
	client := server.Client()
	client.Timeout = 8 * time.Second

	response, err := client.Get(server.URL + "/slow/timeout")
	if err != nil {
		t.Fatal(err)
	}
	body, readErr := io.ReadAll(response.Body)
	response.Body.Close()
	<-entered
	timedOut := <-finished
	if readErr != nil || response.StatusCode != http.StatusServiceUnavailable || string(body) != "slow work timed out\n" || response.Header.Get("Content-Type") != "text/plain; charset=utf-8" {
		t.Fatalf("timeout response: status=%d body=%q err=%v", response.StatusCode, body, readErr)
	}
	if timedOut.parentErr != nil || timedOut.elapsed < 2*time.Second || timedOut.elapsed >= 5*time.Second {
		t.Fatalf("expected child timeout without parent cancellation: %+v", timedOut)
	}
	t.Logf("server timeout: complete HTTP 503 after %s; parent context still active", timedOut.elapsed)

	conn, err := net.DialTimeout("tcp", server.Listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := fmt.Fprintf(conn, "GET /slow/timeout HTTP/1.1\r\nHost: localhost\r\n\r\n"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("timeout request did not reach the server")
	}
	closedAt := time.Now()
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case canceled := <-finished:
		if !errors.Is(canceled.parentErr, context.Canceled) || canceled.elapsed >= 2*time.Second {
			t.Fatalf("parent cancellation did not stop child work: %+v", canceled)
		}
		t.Logf("client disconnect: timeout handler returned after %s", time.Since(closedAt))
	case <-time.After(time.Second):
		t.Fatal("client disconnect was ignored until server timeout")
	}

	response, err = client.Get(server.URL + "/ping")
	if err != nil {
		t.Fatal(err)
	}
	body, readErr = io.ReadAll(response.Body)
	response.Body.Close()
	if readErr != nil || response.StatusCode != http.StatusOK || string(body) != "pong\n" {
		t.Fatal("server must remain usable after timeout/cancellation")
	}
	server.Close()
	text := output.String()
	if strings.Count(text, "slow work timed out: context deadline exceeded") != 1 || strings.Count(text, "slow work canceled: context canceled") != 1 || strings.Contains(text, "slow work completed") || strings.Contains(text, "http panic") {
		t.Fatalf("unexpected timeout/cancellation log: %q", text)
	}
}
