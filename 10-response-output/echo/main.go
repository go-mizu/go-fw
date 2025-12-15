package main

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()

	e.GET("/text", func(c echo.Context) error {
		return c.String(http.StatusOK, "hello")
	})

	e.GET("/json", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "hello"})
	})

	e.GET("/stream", func(c echo.Context) error {
		c.Response().Header().Set("Content-Type", "text/plain")
		for i := 0; i < 3; i++ {
			c.Response().Write([]byte("chunk\n"))
			c.Response().Flush()
			time.Sleep(time.Second)
		}
		return nil
	})

	e.Start(":8080")
}
