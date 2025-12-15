package main

import (
	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(500).SendString("internal server error")
		},
	})

	app.Get("/error", func(c *fiber.Ctx) error {
		return fiber.NewError(400, "bad request")
	})

	app.Get("/panic", func(c *fiber.Ctx) error {
		panic("something went wrong")
	})

	app.Listen(":8080")
}
