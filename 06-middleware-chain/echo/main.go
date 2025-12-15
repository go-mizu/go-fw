package main

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()

	e.Use(middlewareA)
	e.Use(middlewareB)

	e.GET("/", func(c echo.Context) error {
		fmt.Fprintln(c.Response(), "handler")
		return nil
	})

	e.Start(":8080")
}

func middlewareA(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		fmt.Println("A before")
		if err := next(c); err != nil {
			return err
		}
		fmt.Println("A after")
		return nil
	}
}

func middlewareB(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		fmt.Println("B before")
		if err := next(c); err != nil {
			return err
		}
		fmt.Println("B after")
		return nil
	}
}
