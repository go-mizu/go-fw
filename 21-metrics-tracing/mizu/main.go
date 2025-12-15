package main

import (
	"github.com/go-mizu/mizu"
)

func main() {
	app := mizu.New()

	app.Use(mizu.Metrics())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(200, "ok\n")
	})

	app.Listen(":8080")
}
