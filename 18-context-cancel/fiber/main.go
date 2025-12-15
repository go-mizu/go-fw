package main

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	app.Get("/work", func(c *fiber.Ctx) error {
		for i := 0; i < 10; i++ {
			time.Sleep(time.Second)
			c.WriteString("step\n")
		}
		return nil
	})

	app.Listen(":8080")
}
