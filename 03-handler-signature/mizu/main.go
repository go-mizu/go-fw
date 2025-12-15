package main

import (
	"github.com/go-mizu/mizu"
)

func main() {
	app := mizu.New()
	app.Get("/", handler)
	app.Listen(":8080")
}

func handler(c *mizu.Ctx) error {
	return c.Text(200, "hello, world!")
}
