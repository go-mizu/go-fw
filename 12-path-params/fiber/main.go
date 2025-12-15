package main

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	app.Get("/users/:id", func(c *fiber.Ctx) error {
		idStr := c.Params("id")

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return c.Status(400).SendString("invalid id")
		}

		return c.JSON(map[string]any{"id": id})
	})

	app.Get("/files/*", func(c *fiber.Ctx) error {
		// Fiber wildcard is Params("*")
		p := c.Params("*")
		return c.SendString("file path=" + p)
	})

	app.Listen(":8080")
}
