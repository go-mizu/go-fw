package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "root")
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/users", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "api v1 users")
		})
		r.Get("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "api v1 user:", chi.URLParam(r, "id"))
		})
	})

	r.Route("/admin", func(r chi.Router) {
		r.Use(requireToken("letmein"))
		r.Get("/dashboard", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "admin dashboard")
		})
	})

	http.ListenAndServe(":8080", r)
}

func requireToken(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Admin-Token") != token {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte("unauthorized"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
