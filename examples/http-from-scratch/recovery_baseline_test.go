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

// 仅记录没有应用 recovery 时的标准库行为；测试服务器只监听随机本地端口。
func TestHTTPPanicWithoutRecoveryBaseline(t *testing.T) {
	for _, afterWrite := range []bool{false, true} {
		name := "before_response"
		if afterWrite {
			name = "after_response_flush"
		}
		t.Run(name, func(t *testing.T) {
			var serverLog bytes.Buffer
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/ping" {
					pingHandler(w, r)
					return
				}
				if afterWrite {
					w.Header().Set("Content-Type", "text/plain; charset=utf-8")
					if _, err := io.WriteString(w, "partial\n"); err != nil {
						panic(err)
					}
					if err := http.NewResponseController(w).Flush(); err != nil {
						panic(err)
					}
				}
				panic("recovery lesson baseline")
			})
			server := httptest.NewUnstartedServer(loggingMiddleware(handler))
			server.Config.ErrorLog = log.New(&serverLog, "", 0)
			server.Start()
			defer server.Close()
			client := server.Client()

			response, err := client.Get(server.URL + "/panic")
			if !afterWrite {
				if err == nil {
					response.Body.Close()
					t.Fatal("panic before response should interrupt request, not return HTTP 500")
				}
				t.Logf("before write: client received error instead of HTTP response: %v", err)
			} else {
				if err != nil {
					t.Fatalf("headers were already flushed: %v", err)
				}
				body, readErr := io.ReadAll(response.Body)
				response.Body.Close()
				if response.StatusCode != http.StatusOK || readErr == nil || !strings.Contains(string(body), "partial") {
					t.Fatalf("expected partial 200 then interrupted body; status=%d body=%q err=%v", response.StatusCode, body, readErr)
				}
				t.Logf("after write: status=%d, body=%q, read error=%v", response.StatusCode, body, readErr)
			}

			response, err = client.Get(server.URL + "/ping")
			if err != nil {
				t.Fatalf("server must remain usable after request panic: %v", err)
			}
			body, readErr := io.ReadAll(response.Body)
			response.Body.Close()
			if readErr != nil || string(body) != "pong\n" {
				t.Fatalf("follow-up ping failed: body=%q err=%v", body, readErr)
			}
			server.Close()
			if !strings.Contains(serverLog.String(), "http: panic serving") {
				t.Fatal("standard server panic log missing")
			}
			t.Log("follow-up ping: 200; standard library recovered request panic")
		})
	}
}
