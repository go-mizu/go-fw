package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.New()
	r.GET("/", handler)
	r.Run(":8080")
}

func handler(c *gin.Context) {
	c.String(http.StatusOK, "hello, world!")
}
