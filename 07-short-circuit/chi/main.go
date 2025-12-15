package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()

	r.Use(deny)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("handler reached"))
	})

	http.ListenAndServe(":8080", r)
}

func deny(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("denied"))
	})
}
