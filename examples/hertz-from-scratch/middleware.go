package main

import (
	"context"
	"log"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
)

func loggingMiddleware(ctx context.Context, c *app.RequestContext) {
	start := time.Now()
	defer func() {
		log.Printf(
			"hertz request method=%s path=%q duration=%s",
			c.Method(),
			c.Path(),
			time.Since(start),
		)
	}()
	c.Next(ctx)
}
