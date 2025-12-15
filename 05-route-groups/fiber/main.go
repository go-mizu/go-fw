package main

import (
	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("root")
	})

	api := app.Group("/api/v1")
	api.Get("/users", func(c *fiber.Ctx) error {
		return c.SendString("api v1 users")
	})
	api.Get("/users/:id", func(c *fiber.Ctx) error {
		return c.SendString("api v1 user: " + c.Params("id"))
	})

	admin := app.Group("/admin", requireToken("letmein"))
	admin.Get("/dashboard", func(c *fiber.Ctx) error {
		return c.SendString("admin dashboard")
	})

	app.Listen(":8080")
}

func requireToken(token string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Get("X-Admin-Token") != token {
			return c.Status(401).SendString("unauthorized")
		}
		return c.Next()
	}
}
