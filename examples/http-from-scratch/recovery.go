package main

import (
	"errors"
	"log"
	"net/http"
	"runtime/debug"
)

// 仅用于当前普通、非流式 HTTP 示例。
type recoveryWriter struct {
	http.ResponseWriter
	started bool
}

func (rw *recoveryWriter) WriteHeader(code int) {
	if rw.started {
		return
	}

	// 普通 1xx 是临时响应；101 协议切换也视为已经开始。
	if code >= 200 || code == http.StatusSwitchingProtocols {
		rw.started = true
	}

	rw.ResponseWriter.WriteHeader(code)
}

func (rw *recoveryWriter) Write(data []byte) (int, error) {
	if !rw.started {
		rw.WriteHeader(http.StatusOK)
	}

	return rw.ResponseWriter.Write(data)
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &recoveryWriter{ResponseWriter: w}
		defer func() {
			value := recover()
			if value == nil {
				return
			}
			if err, ok := value.(error); ok && errors.Is(err, http.ErrAbortHandler) {
				panic(value)
			}
			log.Printf(
				"http panic method=%s path=%q value=%v\n%s",
				r.Method,
				r.URL.Path,
				value,
				debug.Stack(),
			)

			if rw.started {
				panic(http.ErrAbortHandler)
			}

			rw.Header().Del("Content-Length")
			rw.Header().Del("Content-Encoding")
			http.Error(rw, "internal server error", http.StatusInternalServerError)
		}()
		next.ServeHTTP(rw, r)
	})
}
