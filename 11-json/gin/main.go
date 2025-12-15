package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Payload struct {
	Message string `json:"message" binding:"required"`
}

func main() {
	r := gin.New()

	r.POST("/echo", func(c *gin.Context) {
		var p Payload
		if err := c.ShouldBindJSON(&p); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, p)
	})

	r.Run(":8080")
}
