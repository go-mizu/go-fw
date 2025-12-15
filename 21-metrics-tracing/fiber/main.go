package main

import (
	"github.com/gofiber/fiber/v2"
	fiberprom "github.com/gofiber/contrib/prometheus"
)

func main() {
	app := fiber.New()

	prom := fiberprom.New("fiber")
	prom.RegisterAt(app, "/metrics")
	app.Use(prom.Middleware)

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("ok\n")
	})

	app.Listen(":8080")
}
