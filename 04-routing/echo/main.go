package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()

	e.GET("/", root)
	e.GET("/users", users)
	e.GET("/users/:id", userByID)

	e.Start(":8080")
}

func root(c echo.Context) error {
	return c.String(http.StatusOK, "root")
}

func users(c echo.Context) error {
	return c.String(http.StatusOK, "users")
}

func userByID(c echo.Context) error {
	return c.String(http.StatusOK, "user: "+c.Param("id"))
}
