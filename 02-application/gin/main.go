package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := newRouter()
	r.Run(":8080")
}

func newRouter() *gin.Engine {
	r := gin.New()

	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "hello, world!")
	})

	return r
}
