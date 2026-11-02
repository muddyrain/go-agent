package main

import (
	"context"
	"net/http"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/ut"
)

func TestRoutes(t *testing.T) {
	h := newServer()
	for _, tt := range []struct {
		name       string
		method     string
		target     string
		wantStatus int
		wantBody   string
		wantType   string
	}{
		{"ping", "GET", "/ping", 200, "pong\n", "text/plain; charset=utf-8"},
		{"hello_go", "GET", "/hello?name=Go", 200, "Hello, Go\n", "text/plain; charset=utf-8"},
		{"hello_http", "GET", "/hello?name=HTTP", 200, "Hello, HTTP\n", "text/plain; charset=utf-8"},
		{"missing_name", "GET", "/hello", 400, "name parameter is required\n", "text/plain; charset=utf-8"},
		{"empty_name", "GET", "/hello?name=", 400, "name parameter is required\n", "text/plain; charset=utf-8"},
		{"wrong_method", "POST", "/hello?name=Go", 405, "Method Not Allowed", "text/plain"},
		{"head_not_auto_registered", "HEAD", "/hello?name=Go", 405, "Method Not Allowed", "text/plain"},
		{"unknown_path", "GET", "/missing", 404, "Not Found", "text/plain"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			response := ut.PerformRequest(h.Engine, tt.method, tt.target, nil).Result()
			if response.StatusCode() != tt.wantStatus {
				t.Errorf("status=%d, want %d", response.StatusCode(), tt.wantStatus)
			}
			if string(response.Body()) != tt.wantBody {
				t.Errorf("body=%q, want %q", response.Body(), tt.wantBody)
			}
			if string(response.Header.Peek("Content-Type")) != tt.wantType {
				t.Errorf("Content-Type=%q, want %q", response.Header.Peek("Content-Type"), tt.wantType)
			}
		})
	}
}

func TestMethodNotAllowedOption(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		h := newServer()
		// 仅在隔离测试实例处理请求之前调整配置；不修改生产代码或运行服务。
		h.GetOptions().HandleMethodNotAllowed = enabled
		wantStatus := http.StatusNotFound
		if enabled {
			wantStatus = http.StatusMethodNotAllowed
		}
		response := ut.PerformRequest(h.Engine, "POST", "/hello?name=Go", nil)
		if response.Code != wantStatus {
			t.Fatalf("enabled=%v: POST status=%d, want %d", enabled, response.Code, wantStatus)
		}
		t.Logf("enabled=%v: POST /hello -> %d %q", enabled, response.Code, response.Body.String())
		unknown := ut.PerformRequest(h.Engine, "POST", "/missing", nil)
		if unknown.Code != 404 {
			t.Fatal("enabling 405 must not turn unknown paths into 405")
		}
		get := ut.PerformRequest(h.Engine, "GET", "/hello?name=Go", nil)
		if get.Code != 200 || get.Body.String() != "Hello, Go\n" {
			t.Fatal("option must not change a correctly routed GET request")
		}
	}
}

func TestDirectHandlerDoesNotEnforceRouteMethod(t *testing.T) {
	c := app.NewContext(0)
	c.Request.SetMethod("POST")
	c.Request.SetRequestURI("/hello?name=Go")
	helloHandler(context.Background(), c)
	if c.Response.StatusCode() != 200 || string(c.Response.Body()) != "Hello, Go\n" {
		t.Fatal("direct call bypasses routing; method restriction belongs to the router")
	}
}
