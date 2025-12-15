package main

import (
	"fmt"
	"net/http"
	"time"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /work", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		for i := 0; i < 10; i++ {
			select {
			case <-ctx.Done():
				fmt.Println("canceled:", ctx.Err())
				return
			case <-time.After(time.Second):
				fmt.Fprintf(w, "step %d\n", i)
			}
		}
	})

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	srv.ListenAndServe()
}
