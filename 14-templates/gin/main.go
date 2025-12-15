package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.New()

	r.LoadHTMLGlob("templates/*")

	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "page.html", gin.H{
			"title":   "Home",
			"message": "hello from gin",
		})
	})

	r.Run(":8080")
}
