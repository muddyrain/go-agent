package main

import (
	"context"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
)

func pingHandler(_ context.Context, c *app.RequestContext) {
	c.String(http.StatusOK, "pong\n")
}

func helloHandler(_ context.Context, c *app.RequestContext) {
	name := c.Query("name")

	if name == "" {
		c.String(http.StatusBadRequest, "name parameter is required\n")
		return
	}
	c.String(http.StatusOK, "Hello, %s\n", name)
}

func newServer() *server.Hertz {
	h := server.New(
		server.WithHostPorts("127.0.0.1:8091"),
		server.WithHandleMethodNotAllowed(true),
	)

	h.Use(loggingMiddleware)

	h.GET("/ping", pingHandler)
	h.GET("/hello", helloHandler)
	h.POST("/greet", greetHandler)

	return h
}

func main() {
	h := newServer()
	h.Spin()
}
