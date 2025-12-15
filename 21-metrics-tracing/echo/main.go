package main

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo-contrib/prometheus"
)

func main() {
	e := echo.New()

	p := prometheus.NewPrometheus("echo", nil)
	p.Use(e)

	e.GET("/", func(c echo.Context) error {
		return c.String(200, "ok\n")
	})

	e.Start(":8080")
}
