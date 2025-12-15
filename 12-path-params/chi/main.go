package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()

	r.Get("/users/{id}", getUser)
	r.Get("/files/*", getFile)

	http.ListenAndServe(":8080", r)
}

func getUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "invalid id: %q\n", idStr)
		return
	}

	fmt.Fprintf(w, "user id=%d\n", id)
}

func getFile(w http.ResponseWriter, r *http.Request) {
	p := chi.URLParam(r, "*")
	fmt.Fprintf(w, "file path=%q\n", p)
}
