package main

import (
	"embed"

	"github.com/gin-gonic/gin"
)

//go:embed public/*
var assets embed.FS

func main() {
	r := gin.New()

	// serve from disk
	r.Static("/static", "./public")

	// serve embedded files
	r.StaticFS("/embed", gin.FS(assets))

	r.Run(":8080")
}
