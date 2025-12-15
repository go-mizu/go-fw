package main

import (
	"fmt"
	"net/http"
	"time"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /events", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", 500)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		ctx := r.Context()

		for i := 0; ; i++ {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
				fmt.Fprintf(w, "data: tick %d\n\n", i)
				flusher.Flush()
			}
		}
	})

	http.ListenAndServe(":8080", mux)
}
