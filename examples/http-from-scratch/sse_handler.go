package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func eventsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	controller := http.NewResponseController(w)

	events := []struct {
		name string
		data string
	}{
		{name: "message", data: "第1条消息"},
		{name: "message", data: "第2条消息"},
		{name: "message", data: "第3条消息"},
		{name: "done", data: "发送完成"},
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")

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

		if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.name, event.data); err != nil {
			log.Printf("write SSE event: %v", err)
			return
		}

		if err := controller.Flush(); err != nil {
			log.Printf("flush SSE event: %v", err)
			return
		}

	}

}
