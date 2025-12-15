package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const requestIDHeader = "X-Request-Id"

func main() {
	r := gin.New()

	r.Use(requestID())
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "hello\n")
	})

	_ = r.Run(":8080")
}

func requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader(requestIDHeader)
		if rid == "" {
			rid = gin.MustGet(gin.CreateTestContextOnly).(string) // placeholder, see note below
		}
		c.Writer.Header().Set(requestIDHeader, rid)
		c.Next()
	}
}
