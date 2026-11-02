package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEventsPageRoute(t *testing.T) {
	// go test 的工作目录是包目录，而本教学程序明确从仓库根启动。
	// 只在测试里模拟该启动约定，不改变生产文件路径；这些测试不并行。
	t.Chdir(filepath.Join("..", ".."))
	wantHTML, err := os.ReadFile("examples/http-from-scratch/sse_demo.html")
	if err != nil {
		t.Fatal(err)
	}
	mux := newMux()
	for _, tt := range []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{"page", http.MethodGet, "/events/demo", 200},
		{"head", http.MethodHead, "/events/demo", 200},
		{"wrong_method", http.MethodPost, "/events/demo", 405},
		{"not_directory_server", http.MethodGet, "/events/demo/main.go", 404},
	} {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			mux.ServeHTTP(recorder, httptest.NewRequest(tt.method, tt.path, nil))
			response := recorder.Result()
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if err != nil || response.StatusCode != tt.wantStatus {
				t.Fatalf("status=%d body=%q err=%v", response.StatusCode, body, err)
			}
			if tt.wantStatus == 200 {
				if response.Header.Get("Content-Type") != "text/html; charset=utf-8" || response.Header.Get("Cache-Control") != "no-cache" {
					t.Fatalf("unexpected page headers: %v", response.Header)
				}
				if tt.method == http.MethodHead {
					if len(body) != 0 {
						t.Fatal("HEAD must omit the body")
					}
				} else if string(body) != string(wantHTML) {
					t.Fatal("GET must serve the exact saved HTML file")
				}
			}
			if tt.wantStatus == 405 && response.Header.Get("Allow") != "GET, HEAD" {
				t.Fatalf("Allow=%q", response.Header.Get("Allow"))
			}
		})
	}
}

func TestEventsPageRealHTTP(t *testing.T) {
	t.Chdir(filepath.Join("..", ".."))
	output := captureRequestLog(t)
	server := httptest.NewServer(loggingMiddleware(recoveryMiddleware(newMux())))
	defer server.Close()
	client := server.Client()
	client.Timeout = 5 * time.Second
	response, err := client.Get(server.URL + "/events/demo")
	if err != nil {
		t.Fatal(err)
	}
	body, readErr := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != 200 || readErr != nil || !strings.Contains(string(body), `new EventSource("/events")`) {
		t.Fatalf("page=%d err=%v", response.StatusCode, readErr)
	}
	// 请求页面不执行 JavaScript；另一次 HTTP 请求才获取事件流。
	stream, err := client.Get(server.URL + "/events")
	if err != nil {
		t.Fatal(err)
	}
	streamBody, readErr := io.ReadAll(stream.Body)
	stream.Body.Close()
	server.Close()
	if stream.StatusCode != 200 || stream.Header.Get("Content-Type") != "text/event-stream" || readErr != nil || !strings.HasSuffix(string(streamBody), "event: done\ndata: 发送完成\n\n") {
		t.Fatal("the page and event stream must remain separate valid responses")
	}
	if strings.Count(output.String(), "http request method=") != 2 {
		t.Fatalf("expected two request lifecycles: %q", output.String())
	}
}
