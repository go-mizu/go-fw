package main

import (
	"net/http"
)

func stdHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("from std handler\n"))
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/std", stdHandler)

	http.ListenAndServe(":8080", mux)
}
