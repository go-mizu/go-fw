package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()

	r.Get("/work", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		for i := 0; i < 10; i++ {
			select {
			case <-ctx.Done():
				fmt.Println("canceled")
				return
			case <-time.After(time.Second):
				fmt.Fprintf(w, "step %d\n", i)
			}
		}
	})

	http.ListenAndServe(":8080", r)
}
