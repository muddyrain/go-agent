package main

import (
	"context"
	"log"
	"net/http"
	"runtime/debug"

	"github.com/cloudwego/hertz/pkg/app"
)

// 普通缓冲响应panic返回500；当前HTTP/1流式模式panic中断连接。
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
		// 已切换流式写入模式：不重置响应生成500，直接中断当前连接。
		if c.Response.GetHijackWriter() != nil {
			c.Abort()
			if err := c.GetConn().Close(); err != nil {
				log.Printf("close SSE connection after panic: %v", err)
			}
			return
		}
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
