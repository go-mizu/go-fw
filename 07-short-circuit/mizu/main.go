package main

import (
	"net/http"

	"github.com/go-mizu/mizu"
)

func main() {
	app := mizu.New()

	app.Use(deny)

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "handler reached")
	})

	app.Listen(":8080")
}

func deny(next mizu.Handler) mizu.Handler {
	return func(c *mizu.Ctx) error {
		return c.Text(http.StatusUnauthorized, "denied")
	}
}
