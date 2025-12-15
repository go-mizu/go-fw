package main

import (
	"fmt"
	"net/http"
	"strconv"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /users/{id}", getUser)
	mux.HandleFunc("GET /files/{path...}", getFile)

	http.ListenAndServe(":8080", mux)
}

func getUser(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "invalid id: %q\n", idStr)
		return
	}

	fmt.Fprintf(w, "user id=%d\n", id)
}

func getFile(w http.ResponseWriter, r *http.Request) {
	p := r.PathValue("path")
	fmt.Fprintf(w, "file path=%q\n", p)
}
