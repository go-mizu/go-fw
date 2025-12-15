package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()

	r.Get("/search", search)
	r.Post("/echo", echo)

	http.ListenAndServe(":8080", r)
}

func search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	h := r.Header.Get("User-Agent")

	fmt.Fprintf(w, "query=%s ua=%s\n", q, h)
}

func echo(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid json"))
		return
	}

	json.NewEncoder(w).Encode(payload)
}
