package main

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const firstSSEEvent = "event: message\ndata: 第1条消息\n\n"

// 只在测试中注入写入/刷新失败或取消，不给教学 Handler 增加故障参数。
type sseProbeWriter struct {
	http.ResponseWriter
	writeErr error
	flushErr error
	writes   int
	flushes  int
	onFlush  func()
}

func (w *sseProbeWriter) Write(p []byte) (int, error) {
	w.writes++
	if w.writeErr != nil {
		return 0, w.writeErr
	}
	return w.ResponseWriter.Write(p)
}

func (w *sseProbeWriter) FlushError() error {
	w.flushes++
	if w.onFlush != nil {
		w.onFlush()
	}
	if w.flushErr != nil {
		return w.flushErr
	}
	return http.NewResponseController(w.ResponseWriter).Flush()
}

// 故意只暴露必需接口：底层 Recorder 的刷新能力不自动成为包装器的方法。
type sseBasicWriter struct{ http.ResponseWriter }

func TestEventsHandlerAlreadyCanceled(t *testing.T) {
	output := captureRequestLog(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	response := httptest.NewRecorder()
	writer := &sseProbeWriter{ResponseWriter: response}
	start := time.Now()
	eventsHandler(writer, httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx))
	if time.Since(start) > 500*time.Millisecond || writer.writes != 0 || writer.flushes != 0 {
		t.Fatal("already canceled request must return without writing or flushing")
	}
	if !strings.Contains(output.String(), "SSE request canceled: context canceled") {
		t.Fatalf("missing cancellation log: %q", output.String())
	}
}

func TestEventsHandlerCanceledAfterFirstFlush(t *testing.T) {
	output := captureRequestLog(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	response := httptest.NewRecorder()
	writer := &sseProbeWriter{ResponseWriter: response, onFlush: cancel}
	start := time.Now()
	eventsHandler(writer, httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx))
	if time.Since(start) > 500*time.Millisecond || writer.writes != 1 || writer.flushes != 1 {
		t.Fatal("cancellation must stop the next wait and prevent later events")
	}
	if response.Body.String() != firstSSEEvent || !strings.Contains(output.String(), "SSE request canceled") {
		t.Fatalf("unexpected canceled response/log: %q / %q", response.Body.String(), output.String())
	}
}

func TestEventsHandlerWriteAndFlushFailures(t *testing.T) {
	failure := errors.New("injected SSE failure")
	for _, tt := range []struct {
		name      string
		writeErr  error
		flushErr  error
		wantBody  string
		wantFlush int
		wantLog   string
	}{
		{"write", failure, nil, "", 0, "write SSE event"},
		{"flush", nil, failure, firstSSEEvent, 1, "flush SSE event"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			output := captureRequestLog(t)
			response := httptest.NewRecorder()
			writer := &sseProbeWriter{ResponseWriter: response, writeErr: tt.writeErr, flushErr: tt.flushErr}
			start := time.Now()
			eventsHandler(writer, httptest.NewRequest(http.MethodGet, "/events", nil))
			if time.Since(start) > 500*time.Millisecond || writer.writes != 1 || writer.flushes != tt.wantFlush {
				t.Fatal("failure must return promptly without more events or flushes")
			}
			if response.Body.String() != tt.wantBody || !strings.Contains(output.String(), tt.wantLog) {
				t.Fatalf("body/log = %q / %q", response.Body.String(), output.String())
			}
		})
	}
}

func TestEventsHandlerUnsupportedFlush(t *testing.T) {
	output := captureRequestLog(t)
	response := httptest.NewRecorder()
	start := time.Now()
	eventsHandler(sseBasicWriter{response}, httptest.NewRequest(http.MethodGet, "/events", nil))
	if time.Since(start) > 500*time.Millisecond || response.Body.String() != firstSSEEvent || response.Flushed {
		t.Fatal("unsupported flush must stop after the first write")
	}
	if !strings.Contains(output.String(), "flush SSE event") {
		t.Fatal("unsupported flush must be logged")
	}
}

// 按 SSE 空行读取完整事件，不把一次 TCP 读取误当成一个事件。
func readSSEBlock(reader *bufio.Reader) (string, error) {
	var block strings.Builder
	for {
		line, err := reader.ReadString('\n')
		block.WriteString(line)
		if err != nil {
			return block.String(), err
		}
		if line == "\n" {
			return block.String(), nil
		}
	}
}

func TestEventsHandlerRealHTTPStreams(t *testing.T) {
	output := captureRequestLog(t)
	server := httptest.NewServer(loggingMiddleware(recoveryMiddleware(newMux())))
	defer server.Close()
	client := server.Client()
	client.Timeout = 6 * time.Second
	start := time.Now()
	response, err := client.Get(server.URL + "/events")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || response.Header.Get("Content-Type") != "text/event-stream" || response.Header.Get("Cache-Control") != "no-cache" {
		t.Fatalf("SSE headers/status: %d %v", response.StatusCode, response.Header)
	}
	if response.Header.Get("Content-Length") != "" {
		t.Fatal("stream must not advertise a fixed Content-Length")
	}
	reader := bufio.NewReader(response.Body)
	want := []string{
		firstSSEEvent,
		"event: message\ndata: 第2条消息\n\n",
		"event: message\ndata: 第3条消息\n\n",
		"event: done\ndata: 发送完成\n\n",
	}
	for i, expected := range want {
		block, err := readSSEBlock(reader)
		elapsed := time.Since(start)
		if err != nil || block != expected {
			t.Fatalf("event %d = %q, err=%v, want %q", i+1, block, err, expected)
		}
		// 宽松时间窗：只检测首条是否被缓存、后续是否跳过等待，不要求精确一秒。
		if i == 0 && elapsed >= 800*time.Millisecond {
			t.Fatalf("first event arrived too late: %s", elapsed)
		}
		if i > 0 && elapsed < time.Duration(i)*time.Second-200*time.Millisecond {
			t.Fatalf("event %d skipped ticker wait: %s", i+1, elapsed)
		}
		t.Logf("event %d arrived at %s", i+1, elapsed)
	}
	remainder, err := io.ReadAll(reader)
	if err != nil || len(remainder) != 0 {
		t.Fatalf("expected clean EOF after done, got %q / %v", remainder, err)
	}
	response.Body.Close()
	ping, err := client.Get(server.URL + "/ping")
	if err != nil {
		t.Fatal(err)
	}
	body, readErr := io.ReadAll(ping.Body)
	ping.Body.Close()
	server.Close() // 等待网络 Handler/日志结束后再读取 bytes.Buffer。
	if ping.StatusCode != 200 || readErr != nil || string(body) != "pong\n" {
		t.Fatal("ping must remain available after streaming")
	}
	if strings.Count(output.String(), "http request method=") != 2 {
		t.Fatalf("missing request logs: %q", output.String())
	}
}

func TestEventsHandlerRealHTTPDisconnect(t *testing.T) {
	output := captureRequestLog(t)
	finished := make(chan error, 1)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		eventsHandler(w, r)
		finished <- r.Context().Err()
	})
	server := httptest.NewServer(loggingMiddleware(recoveryMiddleware(next)))
	defer server.Close()
	client := server.Client()
	client.Timeout = 4 * time.Second
	response, err := client.Get(server.URL + "/events")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	block, err := readSSEBlock(bufio.NewReader(response.Body))
	if err != nil || block != firstSSEEvent {
		t.Fatalf("first event = %q / %v", block, err)
	}
	start := time.Now()
	response.Body.Close()
	select {
	case err := <-finished:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("request context = %v, want Canceled", err)
		}
	case <-time.After(750 * time.Millisecond):
		server.CloseClientConnections()
		t.Fatal("Handler did not promptly stop after client disconnected")
	}
	server.Close()
	if !strings.Contains(output.String(), "SSE request canceled: context canceled") || strings.Contains(output.String(), "http panic") {
		t.Fatalf("disconnect is cancellation, not panic: %q", output.String())
	}
	t.Logf("Handler returned %s after disconnect", time.Since(start))
}
