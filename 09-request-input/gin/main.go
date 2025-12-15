package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Payload struct {
	Message string `json:"message"`
}

func main() {
	r := gin.New()

	r.GET("/search", func(c *gin.Context) {
		q := c.Query("q")
		ua := c.GetHeader("User-Agent")

		c.String(http.StatusOK, "query=%s ua=%s\n", q, ua)
	})

	r.POST("/echo", func(c *gin.Context) {
		var p Payload
		if err := c.BindJSON(&p); err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		c.JSON(http.StatusOK, p)
	})

	r.Run(":8080")
}
