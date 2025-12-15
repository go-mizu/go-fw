package main

import (
	"net/http"
	"time"

	"github.com/go-mizu/mizu"
)

func main() {
	app := mizu.New()

	app.Get("/text", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "hello")
	})

	app.Get("/json", func(c *mizu.Ctx) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "hello"})
	})

	app.Get("/stream", func(c *mizu.Ctx) error {
		c.SetHeader("Content-Type", "text/plain")
		for i := 0; i < 3; i++ {
			c.Write([]byte("chunk\n"))
			c.Flush()
			time.Sleep(time.Second)
		}
		return nil
	})

	app.Listen(":8080")
}
