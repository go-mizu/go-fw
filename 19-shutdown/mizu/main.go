package main

import (
	"net/http"

	"github.com/go-mizu/mizu"
)

func main() {
	app := mizu.New()

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "hello\n")
	})

	app.Get("/livez", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok\n")
	})

	app.Get("/readyz", func(c *mizu.Ctx) error {
		return c.Writer().ServeHTTP(c.Writer(), c.Request()) // placeholder: use app.ReadyzHandler in real app
	})

	app.Listen(":8080")
}
