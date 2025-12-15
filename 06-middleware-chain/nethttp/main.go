package main

import (
	"fmt"
	"net/http"
)

func main() {
	handler := chain(
		http.HandlerFunc(finalHandler),
		middlewareA,
		middlewareB,
	)

	http.ListenAndServe(":8080", handler)
}

func finalHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "handler")
}

type Middleware func(http.Handler) http.Handler

func middlewareA(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("A before")
		next.ServeHTTP(w, r)
		fmt.Println("A after")
	})
}

func middlewareB(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("B before")
		next.ServeHTTP(w, r)
		fmt.Println("B after")
	})
}

func chain(h http.Handler, m ...Middleware) http.Handler {
	for i := len(m) - 1; i >= 0; i-- {
		h = m[i](h)
	}
	return h
}
