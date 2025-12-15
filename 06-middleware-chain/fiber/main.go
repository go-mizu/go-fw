package main

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	app.Use(middlewareA)
	app.Use(middlewareB)

	app.Get("/", func(c *fiber.Ctx) error {
		fmt.Println("handler")
		return c.SendString("handler")
	})

	app.Listen(":8080")
}

func middlewareA(c *fiber.Ctx) error {
	fmt.Println("A before")
	if err := c.Next(); err != nil {
		return err
	}
	fmt.Println("A after")
	return nil
}

func middlewareB(c *fiber.Ctx) error {
	fmt.Println("B before")
	if err := c.Next(); err != nil {
		return err
	}
	fmt.Println("B after")
	return nil
}
