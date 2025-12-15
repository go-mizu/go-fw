package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()

	r.Get("/events", func(w http.ResponseWriter, r *http.Request) {
		flusher := w.(http.Flusher)

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")

		for {
			select {
			case <-r.Context().Done():
				return
			case <-time.After(time.Second):
				fmt.Fprint(w, "data: tick\n\n")
				flusher.Flush()
			}
		}
	})

	http.ListenAndServe(":8080", r)
}
