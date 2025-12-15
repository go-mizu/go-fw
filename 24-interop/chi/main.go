package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func stdHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("from std handler\n"))
}

func main() {
	r := chi.NewRouter()

	r.Handle("/std", http.HandlerFunc(stdHandler))

	http.ListenAndServe(":8080", r)
}
