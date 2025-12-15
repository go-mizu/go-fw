package main

import (
	"html/template"
	"net/http"

	"github.com/labstack/echo/v4"
)

type TemplateRenderer struct {
	t *template.Template
}

func (r *TemplateRenderer) Render(w http.ResponseWriter, name string, data any, c echo.Context) error {
	return r.t.ExecuteTemplate(w, name, data)
}

func main() {
	e := echo.New()

	e.Renderer = &TemplateRenderer{
		t: template.Must(template.ParseGlob("templates/*.html")),
	}

	e.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "page.html", map[string]string{
			"Message": "hello from echo",
		})
	})

	e.Start(":8080")
}
