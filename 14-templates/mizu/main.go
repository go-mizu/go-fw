package main

import (
	"html/template"
	"net/http"

	"github.com/go-mizu/mizu"
)

func main() {
	app := mizu.New()

	tpl := template.Must(template.ParseGlob("templates/*.html"))

	app.Get("/", func(c *mizu.Ctx) error {
		c.SetHeader("Content-Type", "text/html; charset=utf-8")
		return tpl.Execute(c.Writer(), map[string]string{
			"Message": "hello from mizu",
		})
	})

	app.Listen(":8080")
}
