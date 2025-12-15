package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.New()

	r.Use(deny)

	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "handler reached")
	})

	r.Run(":8080")
}

func deny(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"error": "denied",
	})
}
