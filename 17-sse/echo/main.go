package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()

	e.GET("/events", func(c echo.Context) error {
		res := c.Response()
		req := c.Request()

		res.Header().Set("Content-Type", "text/event-stream")
		res.Header().Set("Cache-Control", "no-cache")

		for i := 0; ; i++ {
			select {
			case <-req.Context().Done():
				return nil
			case <-time.After(time.Second):
				fmt.Fprintf(res, "data: tick %d\n\n", i)
				res.Flush()
			}
		}
	})

	e.Start(":8080")
}
