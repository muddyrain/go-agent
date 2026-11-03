package main

import (
	"context"
	"log"
	"net/http"
	"runtime/debug"

	"github.com/cloudwego/hertz/pkg/app"
)

// 仅用于当前教学示例的普通缓冲响应，不处理已发送的流式响应。
func recoveryMiddleware(ctx context.Context, c *app.RequestContext) {
	defer func() {
		value := recover()
		if value == nil {
			return
		}
		log.Printf(
			"hertz panic method=%s path=%q value=%v\n%s",
			c.Method(),
			c.Path(),
			value,
			debug.Stack(),
		)
		c.AbortWithMsg(
			"internal server error\n",
			http.StatusInternalServerError,
		)
		c.SetContentType("text/plain; charset=utf-8")
	}()

	c.Next(ctx)
}

// 固定故障端点仅用于本地教学，观察恢复器是否工作。
func panicHandler(_ context.Context, _ *app.RequestContext) {
	panic("demo panic")
}
