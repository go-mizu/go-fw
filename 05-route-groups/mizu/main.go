package main

import (
	"net/http"

	"github.com/go-mizu/mizu"
)

func main() {
	app := mizu.New()

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "root")
	})

	api := app.Group("/api/v1")
	api.Get("/users", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "api v1 users")
	})
	api.Get("/users/:id", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "api v1 user: "+c.Param("id"))
	})

	admin := app.Group("/admin")
	admin.Use(requireToken("letmein"))
	admin.Get("/dashboard", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "admin dashboard")
	})

	app.Listen(":8080")
}

func requireToken(token string) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			if c.Request().Header.Get("X-Admin-Token") != token {
				return c.Text(http.StatusUnauthorized, "unauthorized")
			}
			return next(c)
		}
	}
}
