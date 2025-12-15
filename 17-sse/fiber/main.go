package main

import (
	"bufio"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	app.Get("/events", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/event-stream")
		c.Set("Cache-Control", "no-cache")

		c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
			for i := 0; i < 10; i++ {
				fmt.Fprintf(w, "data: tick %d\n\n", i)
				w.Flush()
				time.Sleep(time.Second)
			}
		})

		return nil
	})

	app.Listen(":8080")
}
