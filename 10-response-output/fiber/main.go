package main

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	app.Get("/text", func(c *fiber.Ctx) error {
		return c.SendString("hello")
	})

	app.Get("/json", func(c *fiber.Ctx) error {
		return c.JSON(map[string]string{"message": "hello"})
	})

	app.Get("/stream", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/plain")
		for i := 0; i < 3; i++ {
			c.WriteString("chunk\n")
			time.Sleep(time.Second)
		}
		return nil
	})

	app.Listen(":8080")
}
