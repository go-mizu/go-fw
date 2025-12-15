package main

import (
	"net/http"

	"github.com/go-mizu/mizu"
)

func main() {
	app := mizu.New()
	app.Get("/ping", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "pong")
	})

	http.ListenAndServe(":8080", app)
}
