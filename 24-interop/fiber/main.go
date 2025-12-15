package main

import (
	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	app.Get("/std", func(c *fiber.Ctx) error {
		return c.SendString("from fiber\n")
	})

	app.Listen(":8080")
}
