package main

import (
	"io"
	"net/http"
	"os"

	"github.com/go-mizu/mizu"
)

func main() {
	app := mizu.New()

	app.Post("/login", func(c *mizu.Ctx) error {
		user := c.Form("user")
		pass := c.Form("pass")

		return c.Text(http.StatusOK, "user="+user+" pass="+pass)
	})

	app.Post("/upload", func(c *mizu.Ctx) error {
		file, header, err := c.FormFile("file")
		if err != nil {
			return c.Text(http.StatusBadRequest, "file missing")
		}
		defer file.Close()

		out, err := os.Create(header.Filename)
		if err != nil {
			return err
		}
		defer out.Close()

		io.Copy(out, file)

		return c.Text(http.StatusOK, "uploaded "+header.Filename)
	})

	app.Listen(":8080")
}
