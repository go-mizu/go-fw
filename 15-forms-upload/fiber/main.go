package main

import (
	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	app.Post("/login", func(c *fiber.Ctx) error {
		user := c.FormValue("user")
		pass := c.FormValue("pass")

		return c.SendString("user=" + user + " pass=" + pass)
	})

	app.Post("/upload", func(c *fiber.Ctx) error {
		file, err := c.FormFile("file")
		if err != nil {
			return c.Status(400).SendString("file missing")
		}

		c.SaveFile(file, "./"+file.Filename)
		return c.SendString("uploaded " + file.Filename)
	})

	app.Listen(":8080")
}
