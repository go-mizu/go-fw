package main

import (
	"errors"
	"net/http"

	"github.com/go-mizu/mizu"
)

func main() {
	app := mizu.New()

	app.Get("/error", func(c *mizu.Ctx) error {
		return mizu.HTTPError{
			Status: http.StatusBadRequest,
			Err:    errors.New("bad request"),
		}
	})

	app.Get("/panic", func(c *mizu.Ctx) error {
		panic("something went wrong")
	})

	app.Listen(":8080")
}
