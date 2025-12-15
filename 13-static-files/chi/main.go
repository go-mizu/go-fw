package main

import (
	"embed"
	"net/http"

	"github.com/go-chi/chi/v5"
)

//go:embed public/*
var assets embed.FS

func main() {
	r := chi.NewRouter()

	r.Handle("/static/*",
		http.StripPrefix("/static/",
			http.FileServer(http.Dir("./public")),
		),
	)

	r.Handle("/embed/*",
		http.StripPrefix("/embed/",
			http.FileServer(http.FS(assets)),
		),
	)

	http.ListenAndServe(":8080", r)
}
