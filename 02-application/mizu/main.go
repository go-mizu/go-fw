package main

import (
	"github.com/go-mizu/mizu"
)

func main() {
	app := newApp()
	app.Listen(":8080")
}

func newApp() *mizu.App {
	app := mizu.New()

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(200, "hello, world!")
	})

	return app
}
