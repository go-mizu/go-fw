package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.New()

	r.Use(gin.Recovery())

	r.GET("/error", func(c *gin.Context) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": "bad request",
		})
	})

	r.GET("/panic", func(c *gin.Context) {
		panic("something went wrong")
	})

	r.Run(":8080")
}
