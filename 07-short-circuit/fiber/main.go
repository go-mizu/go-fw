package main

import (
	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	app.Use(deny)

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("handler reached")
	})

	app.Listen(":8080")
}

func deny(c *fiber.Ctx) error {
	return c.Status(401).SendString("denied")
}
