package main

import (
	"embed"
	"net/http"

	"github.com/go-mizu/mizu"
)

//go:embed public/*
var assets embed.FS

func main() {
	app := mizu.New()

	app.Static("/static", "./public")
	app.StaticFS("/embed", http.FS(assets))

	app.Listen(":8080")
}
