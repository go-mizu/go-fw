package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /text", text)
	mux.HandleFunc("GET /json", jsonResp)
	mux.HandleFunc("GET /stream", stream)

	http.ListenAndServe(":8080", mux)
}

func text(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("hello"))
}

func jsonResp(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "hello",
	})
}

func stream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", 500)
		return
	}

	for i := 0; i < 3; i++ {
		fmt.Fprintf(w, "chunk %d\n", i)
		flusher.Flush()
		time.Sleep(time.Second)
	}
}
