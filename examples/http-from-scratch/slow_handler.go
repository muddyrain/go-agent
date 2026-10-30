package main

import (
	"context"
	"errors"
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
	case <-timer.C:
	}

	// 工作完成与取消可能同时就绪，等待结束后统一检查。
	if err := ctx.Err(); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("slow work timed out: %v", err)
			http.Error(w, "slow work timed out", http.StatusServiceUnavailable)
			return
		}

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

func slowTimeoutHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	slowHandler(w, r.WithContext(ctx))
}
