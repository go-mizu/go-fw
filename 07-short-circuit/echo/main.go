package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()

	e.Use(deny)

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "handler reached")
	})

	e.Start(":8080")
}

func deny(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		return echo.NewHTTPError(http.StatusUnauthorized, "denied")
	}
}
