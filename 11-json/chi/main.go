package main

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Payload struct {
	Message string `json:"message"`
}

func main() {
	r := chi.NewRouter()

	r.Post("/echo", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var p Payload
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		json.NewEncoder(w).Encode(p)
	})

	http.ListenAndServe(":8080", r)
}
