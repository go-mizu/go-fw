package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()
	e.GET("/", handler)
	e.Start(":8080")
}

func handler(c echo.Context) error {
	return c.String(http.StatusOK, "hello, world!")
}
