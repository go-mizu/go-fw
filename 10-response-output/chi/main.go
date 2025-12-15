package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()

	r.Get("/text", text)
	r.Get("/json", jsonResp)
	r.Get("/stream", stream)

	http.ListenAndServe(":8080", r)
}

func text(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello"))
}

func jsonResp(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "hello",
	})
}

func stream(w http.ResponseWriter, r *http.Request) {
	flusher := w.(http.Flusher)

	for i := 0; i < 3; i++ {
		fmt.Fprintf(w, "chunk %d\n", i)
		flusher.Flush()
		time.Sleep(time.Second)
	}
}
