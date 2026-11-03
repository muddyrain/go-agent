package main

import (
	"context"
	"log"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/sse"
)

func eventsHandler(ctx context.Context, c *app.RequestContext) {
	events := []struct {
		name string
		data string
	}{
		{name: "message", data: "第1条消息"},
		{name: "message", data: "第2条消息"},
		{name: "message", data: "第3条消息"},
		{name: "done", data: "发送完成"},
	}

	writer := sse.NewWriter(c)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for i, event := range events {
		if i > 0 {
			select {
			case <-ctx.Done():
			case <-ticker.C:
			}
		}

		if err := ctx.Err(); err != nil {
			log.Printf("SSE request canceled: %v", err)
			return
		}

		if err := writer.WriteEvent("", event.name, []byte(event.data)); err != nil {
			log.Printf("write SSE event: %v", err)
			return
		}
	}
}
