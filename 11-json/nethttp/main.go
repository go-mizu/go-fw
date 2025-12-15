package main

import (
	"encoding/json"
	"net/http"
)

type Payload struct {
	Message string `json:"message"`
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /echo", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var p Payload
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(p)
	})

	http.ListenAndServe(":8080", mux)
}
