package main

import (
	"github.com/go-mizu/mizu"
)

func main() {
	app := mizu.New()

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(200, "hello, world!")
	})

	app.Listen(":8080")
}
