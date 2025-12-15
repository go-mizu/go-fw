package main

import (
	"embed"
	"net/http"
)

//go:embed public/*
var assets embed.FS

func main() {
	mux := http.NewServeMux()

	// serve from disk
	mux.Handle("/static/",
		http.StripPrefix("/static/",
			http.FileServer(http.Dir("./public")),
		),
	)

	// serve embedded files
	fs := http.FS(assets)
	mux.Handle("/embed/",
		http.StripPrefix("/embed/",
			http.FileServer(fs),
		),
	)

	http.ListenAndServe(":8080", mux)
}
