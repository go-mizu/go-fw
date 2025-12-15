package main

import (
	"github.com/gofiber/fiber/v2"
)

func main() {
	app := newApp()
	app.Listen(":8080")
}

func newApp() *fiber.App {
	app := fiber.New()

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("hello, world!")
	})

	return app
}
