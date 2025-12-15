package main

import (
	"fmt"
	"net/http"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", root)
	mux.HandleFunc("GET /users", users)
	mux.HandleFunc("GET /users/", userSubtree)

	http.ListenAndServe(":8080", mux)
}

func root(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "root")
}

func users(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "users")
}

func userSubtree(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "users subtree:", r.URL.Path)
}
