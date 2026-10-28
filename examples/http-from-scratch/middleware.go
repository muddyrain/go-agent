package main

import (
	"log"
	"net/http"
	"time"
)

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		defer func() {
			log.Printf(
				"http request method=%s path=%q duration=%s",
				r.Method,
				r.URL.Path,
				time.Since(start),
			)
		}()
		next.ServeHTTP(w, r)
	})
}
