package main

import (
	"fmt"
	"net/http"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /error", errorHandler)
	mux.HandleFunc("GET /panic", panicHandler)

	handler := recoverMiddleware(mux)

	http.ListenAndServe(":8080", handler)
}

func errorHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
	fmt.Fprintln(w, "bad request")
}

func panicHandler(w http.ResponseWriter, r *http.Request) {
	panic("something went wrong")
}

func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintln(w, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
