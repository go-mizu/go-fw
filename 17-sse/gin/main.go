package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.New()

	r.GET("/events", func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")

		flusher := c.Writer.(http.Flusher)

		for i := 0; ; i++ {
			select {
			case <-c.Request.Context().Done():
				return
			case <-time.After(time.Second):
				fmt.Fprintf(c.Writer, "data: tick %d\n\n", i)
				flusher.Flush()
			}
		}
	})

	r.Run(":8080")
}
