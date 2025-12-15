package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()

	r.Get("/", root)
	r.Get("/users", users)
	r.Get("/users/{id}", userByID)

	http.ListenAndServe(":8080", r)
}

func root(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "root")
}

func users(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "users")
}

func userByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	fmt.Fprintln(w, "user:", id)
}
