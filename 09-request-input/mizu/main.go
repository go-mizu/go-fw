package main

import (
	"net/http"

	"github.com/go-mizu/mizu"
)

type Payload struct {
	Message string `json:"message"`
}

func main() {
	app := mizu.New()

	app.Get("/search", func(c *mizu.Ctx) error {
		q := c.Query("q")
		ua := c.Request().Header.Get("User-Agent")

		return c.Text(http.StatusOK, "query="+q+" ua="+ua)
	})

	app.Post("/echo", func(c *mizu.Ctx) error {
		var p Payload
		if err := c.Bind(&p); err != nil {
			return err
		}
		return c.JSON(http.StatusOK, p)
	})

	app.Listen(":8080")
}
