package main

import (
	"embed"

	"github.com/gofiber/fiber/v2"
)

//go:embed public/*
var assets embed.FS

func main() {
	app := fiber.New()

	app.Static("/static", "./public")
	app.Static("/embed", "./public") // Fiber does not support embed.FS directly

	app.Listen(":8080")
}
