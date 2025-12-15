package main

import (
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()

	e.GET("/work", func(c echo.Context) error {
		ctx := c.Request().Context()

		for i := 0; i < 10; i++ {
			select {
			case <-ctx.Done():
				fmt.Println("canceled")
				return nil
			case <-time.After(time.Second):
				c.Response().Write([]byte("step\n"))
			}
		}
		return nil
	})

	e.Start(":8080")
}
