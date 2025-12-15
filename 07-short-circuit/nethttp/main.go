package main

import (
	"net/http"
)

func main() {
	handler := deny(
		http.HandlerFunc(finalHandler),
	)

	http.ListenAndServe(":8080", handler)
}

func finalHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("handler reached"))
}

func deny(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("denied"))
	})
}
