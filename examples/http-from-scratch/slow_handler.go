package main

import (
	"io"
	"log"
	"net/http"
	"time"
)

func slowHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	log.Printf("slow work started")

	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		log.Printf("slow work canceled: %v", ctx.Err())
		return

	case <-timer.C:
		// 完成与取消可能同时就绪，写响应前再次检查。
		if err := ctx.Err(); err != nil {
			log.Printf("slow work canceled: %v", err)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")

		if _, err := io.WriteString(w, "slow work completed\n"); err != nil {
			log.Printf("write slow response: %v", err)
			return
		}

		log.Printf("slow work completed")
	}
}
