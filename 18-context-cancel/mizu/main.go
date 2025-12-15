package main

import (
	"fmt"
	"time"

	"github.com/go-mizu/mizu"
)

func main() {
	app := mizu.New()

	app.Get("/work", func(c *mizu.Ctx) error {
		ctx := c.Request().Context()

		for i := 0; i < 10; i++ {
			select {
			case <-ctx.Done():
				fmt.Println("canceled")
				return nil
			case <-time.After(time.Second):
				c.Write([]byte("step\n"))
				c.Flush()
			}
		}
		return nil
	})

	app.Listen(":8080")
}
