package main

import (
	"github.com/gofiber/fiber/v2"
)

type Payload struct {
	Message string `json:"message"`
}

func main() {
	app := fiber.New()

	app.Post("/echo", func(c *fiber.Ctx) error {
		var p Payload
		if err := c.BodyParser(&p); err != nil {
			return c.Status(400).SendString("invalid json")
		}
		return c.JSON(p)
	})

	app.Listen(":8080")
}
