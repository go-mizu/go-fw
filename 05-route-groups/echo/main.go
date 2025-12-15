package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "root")
	})

	api := e.Group("/api/v1")
	api.GET("/users", func(c echo.Context) error {
		return c.String(http.StatusOK, "api v1 users")
	})
	api.GET("/users/:id", func(c echo.Context) error {
		return c.String(http.StatusOK, "api v1 user: "+c.Param("id"))
	})

	admin := e.Group("/admin", requireToken("letmein"))
	admin.GET("/dashboard", func(c echo.Context) error {
		return c.String(http.StatusOK, "admin dashboard")
	})

	e.Start(":8080")
}

func requireToken(token string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Request().Header.Get("X-Admin-Token") != token {
				return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
			}
			return next(c)
		}
	}
}
