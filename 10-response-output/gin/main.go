package main

import (
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.New()

	r.GET("/text", func(c *gin.Context) {
		c.String(http.StatusOK, "hello")
	})

	r.GET("/json", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "hello"})
	})

	r.GET("/stream", func(c *gin.Context) {
		c.Stream(func(w io.Writer) bool {
			for i := 0; i < 3; i++ {
				w.Write([]byte("chunk\n"))
				time.Sleep(time.Second)
			}
			return false
		})
	})

	r.Run(":8080")
}
