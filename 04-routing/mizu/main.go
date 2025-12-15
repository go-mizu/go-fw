package main

import (
	"github.com/go-mizu/mizu"
)

func main() {
	app := mizu.New()

	app.Get("/", root)
	app.Get("/users", users)
	app.Get("/users/:id", userByID)

	app.Listen(":8080")
}

func root(c *mizu.Ctx) error {
	return c.Text(200, "root")
}

func users(c *mizu.Ctx) error {
	return c.Text(200, "users")
}

func userByID(c *mizu.Ctx) error {
	return c.Text(200, "user: "+c.Param("id"))
}
