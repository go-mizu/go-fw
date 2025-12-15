package main

import (
	"net/http"
	"strconv"

	"github.com/go-mizu/mizu"
)

func main() {
	app := mizu.New()

	app.Get("/users/:id", func(c *mizu.Ctx) error {
		idStr := c.Param("id")

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return c.Text(http.StatusBadRequest, "invalid id")
		}

		return c.JSON(http.StatusOK, map[string]any{"id": id})
	})

	app.Get("/files/*path", func(c *mizu.Ctx) error {
		p := c.Param("path")
		return c.Text(http.StatusOK, "file path="+p)
	})

	app.Listen(":8080")
}
