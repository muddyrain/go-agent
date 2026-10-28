package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ping", pingHandler)
	mux.HandleFunc("POST /echo", echoHandler)
	mux.HandleFunc("GET /hello", helloHandler)
	mux.HandleFunc("POST /greet", greetHandler)

	address := "127.0.0.1:8090"
	log.Printf("starting HTTP server at http://%s", address)

	handler := loggingMiddleware(mux)

	if err := http.ListenAndServe(address, handler); err != nil {
		log.Fatalf("HTTP server stopped: %v", err)
	}
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	if _, err := io.WriteString(w, "pong\n"); err != nil {
		log.Printf("write ping response: %v", err)
	}
}

func echoHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 4*1024)

	body, err := io.ReadAll(r.Body)
	log.Printf("echo request body=%q", body)
	if err != nil {
		var sizeErr *http.MaxBytesError
		if errors.As(err, &sizeErr) {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}

		http.Error(w, "cannot read request body", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	if _, err := w.Write(body); err != nil {
		log.Printf("write echo response: %v", err)
	}
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	query := r.URL.Query()
	name := query.Get("name")
	if name == "" {
		http.Error(w, "name parameter is required", http.StatusBadRequest)
		return
	}

	if _, err := fmt.Fprintf(w, "Hello, %s\n", name); err != nil {
		log.Printf("write hello response: %v", err)
	}
}
