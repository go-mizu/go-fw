package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func main() {
	e := newApp()
	e.Start(":8080")
}

func newApp() *echo.Echo {
	e := echo.New()

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "hello, world!")
	})

	return e
}
