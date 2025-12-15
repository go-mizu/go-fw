package main

import (
	"fmt"
	"time"

	"github.com/go-mizu/mizu"
)

func main() {
	app := mizu.New()

	app.Get("/events", func(c *mizu.Ctx) error {
		c.SetHeader("Content-Type", "text/event-stream")
		c.SetHeader("Cache-Control", "no-cache")

		ctx := c.Request().Context()

		for i := 0; ; i++ {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(time.Second):
				fmt.Fprintf(c.Writer(), "data: tick %d\n\n", i)
				c.Flush()
			}
		}
	})

	app.Listen(":8080")
}
