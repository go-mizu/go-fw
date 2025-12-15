package main

import (
	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	app.Get("/", root)
	app.Get("/users", users)
	app.Get("/users/:id", userByID)

	app.Listen(":8080")
}

func root(c *fiber.Ctx) error {
	return c.SendString("root")
}

func users(c *fiber.Ctx) error {
	return c.SendString("users")
}

func userByID(c *fiber.Ctx) error {
	return c.SendString("user: " + c.Params("id"))
}
