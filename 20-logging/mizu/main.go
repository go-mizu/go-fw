package main

import (
	"net/http"

	"github.com/go-mizu/mizu"
)

const requestIDHeader = "X-Request-Id"

func main() {
	app := mizu.New()

	// If your Mizu logger already generates request ids when missing,
	// you only need to install it once.
	//
	// app.Use(mizu.Logger()) // example, depending on your actual API

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "hello\n")
	})

	_ = app.Listen(":8080")
}
