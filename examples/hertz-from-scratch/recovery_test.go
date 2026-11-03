package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/ut"
)

func TestRecoveryMiddlewarePreservesNormalResponse(t *testing.T) {
	output := captureHertzRequestLog(t)
	h := newServer()
	h.GET("/normal", func(_ context.Context, c *app.RequestContext) {
		c.Header("X-Test", "preserved")
		c.String(201, "normal response\n")
	})
	r := ut.PerformRequest(h.Engine, "GET", "/normal", nil).Result()
	if r.StatusCode() != 201 || string(r.Body()) != "normal response\n" || string(r.Header.Peek("X-Test")) != "preserved" {
		t.Fatal("normal response must remain unchanged")
	}
	if string(r.Header.Peek("Content-Type")) != "text/plain; charset=utf-8" {
		t.Fatal("normal Content-Type changed")
	}
	if strings.Contains(output.String(), "hertz panic") || strings.Count(output.String(), "hertz request") != 1 {
		t.Fatalf("normal request must not produce a panic log: %q", output.String())
	}
}

func TestRecoveryMiddlewarePanicValues(t *testing.T) {
	for _, tt := range []struct {
		name  string
		value any
		text  string
	}{
		{"string", "string panic", "string panic"},
		{"error", errors.New("error panic"), "error panic"},
		{"number", 42, "42"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			output := captureHertzRequestLog(t)
			h := newServer()
			afterPanic, pending := false, 0
			h.GET("/test-panic", func(_ context.Context, _ *app.RequestContext) {
				panic(tt.value)
			}, func(_ context.Context, c *app.RequestContext) {
				pending++
				c.String(200, "must not run")
			})
			// 通过独立函数确认 panic 后的语句不会被恢复执行。
			h.GET("/after-panic", func(_ context.Context, _ *app.RequestContext) {
				func() { panic(tt.value) }()
				afterPanic = true
			})
			for _, path := range []string{"/test-panic", "/after-panic"} {
				output.Reset()
				r := ut.PerformRequest(h.Engine, "GET", path+"?hidden=query-value", nil).Result()
				assertRecoveryResponse(t, r.StatusCode(), string(r.Body()), string(r.Header.Peek("Content-Type")))
				text := output.String()
				panicAt := strings.Index(text, "hertz panic method=GET path=\""+path+"\" value="+tt.text)
				requestAt := strings.Index(text, "hertz request method=GET path=\""+path+"\"")
				if panicAt < 0 || requestAt < panicAt || !strings.Contains(text, "recoveryMiddleware") || !strings.Contains(text, "goroutine ") {
					t.Fatalf("panic detail/stack must precede outer request log: %q", text)
				}
				if strings.Count(text, "hertz panic method=") != 1 || strings.Count(text, "hertz request method=") != 1 || strings.Contains(text, "query-value") {
					t.Fatalf("unexpected log count/content: %q", text)
				}
			}
			if pending != 0 || afterPanic {
				t.Fatalf("panic must not resume failed code or pending Handler: pending=%d after=%v", pending, afterPanic)
			}
		})
	}
}

func TestRecoveryMiddlewareClearsBufferedResponse(t *testing.T) {
	output := captureHertzRequestLog(t)
	h := newServer()
	h.GET("/buffered-panic", func(_ context.Context, c *app.RequestContext) {
		c.Header("Content-Encoding", "gzip")
		c.Header("Content-Length", "999")
		c.Header("X-Stale", "old response")
		c.String(200, "partial success")
		panic("buffered panic")
	})
	r := ut.PerformRequest(h.Engine, "GET", "/buffered-panic", nil).Result()
	assertRecoveryResponse(t, r.StatusCode(), string(r.Body()), string(r.Header.Peek("Content-Type")))
	if len(r.Header.Peek("Content-Encoding")) != 0 || len(r.Header.Peek("X-Stale")) != 0 || string(r.Header.Peek("Content-Length")) == "999" {
		t.Fatal("stale headers survived response reset")
	}
	if !strings.Contains(output.String(), "value=buffered panic") {
		t.Fatal("server must retain diagnostic detail")
	}
}

func TestRecoveryPanicRouteAndSubsequentPing(t *testing.T) {
	output := captureHertzRequestLog(t)
	h := newServer()
	r := ut.PerformRequest(h.Engine, "GET", "/panic", nil).Result()
	assertRecoveryResponse(t, r.StatusCode(), string(r.Body()), string(r.Header.Peek("Content-Type")))
	if !strings.Contains(output.String(), "value=demo panic") {
		t.Fatal("registered teaching panic Handler was not reached")
	}
	output.Reset()
	ping := ut.PerformRequest(h.Engine, "GET", "/ping", nil)
	if ping.Code != 200 || ping.Body.String() != "pong\n" || strings.Contains(output.String(), "hertz panic") {
		t.Fatal("following request must remain healthy")
	}
	wrongMethod := ut.PerformRequest(h.Engine, "POST", "/panic", nil)
	if wrongMethod.Code != 405 || wrongMethod.Body.String() != "Method Not Allowed" {
		t.Fatal("POST /panic must not reach the GET Handler")
	}
}

func assertRecoveryResponse(t *testing.T, status int, body, contentType string) {
	t.Helper()
	if status != 500 || body != "internal server error\n" || contentType != "text/plain; charset=utf-8" {
		t.Fatalf("want clean plain-text 500, got status=%d body=%q Content-Type=%q", status, body, contentType)
	}
}
