package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()

	r.Use(middlewareA)
	r.Use(middlewareB)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "handler")
	})

	http.ListenAndServe(":8080", r)
}

func middlewareA(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("A before")
		next.ServeHTTP(w, r)
		fmt.Println("A after")
	})
}

func middlewareB(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("B before")
		next.ServeHTTP(w, r)
		fmt.Println("B after")
	})
}
