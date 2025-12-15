package main

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.New()

	r.GET("/work", func(c *gin.Context) {
		ctx := c.Request.Context()

		for i := 0; i < 10; i++ {
			select {
			case <-ctx.Done():
				fmt.Println("canceled")
				return
			case <-time.After(time.Second):
				c.Writer.Write([]byte("step\n"))
			}
		}
	})

	r.Run(":8080")
}
