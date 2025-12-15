package main

import (
	"fmt"

	"github.com/go-mizu/mizu"
)

func main() {
	app := mizu.New()

	app.Use(middlewareA)
	app.Use(middlewareB)

	app.Get("/", func(c *mizu.Ctx) error {
		fmt.Println("handler")
		return c.Text(200, "handler")
	})

	app.Listen(":8080")
}

func middlewareA(next mizu.Handler) mizu.Handler {
	return func(c *mizu.Ctx) error {
		fmt.Println("A before")
		if err := next(c); err != nil {
			return err
		}
		fmt.Println("A after")
		return nil
	}
}

func middlewareB(next mizu.Handler) mizu.Handler {
	return func(c *mizu.Ctx) error {
		fmt.Println("B before")
		if err := next(c); err != nil {
			return err
		}
		fmt.Println("B after")
		return nil
	}
}
