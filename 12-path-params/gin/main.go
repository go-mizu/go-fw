package main

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.New()

	r.GET("/users/:id", func(c *gin.Context) {
		idStr := c.Param("id")

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "invalid id",
				"id":    idStr,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{"id": id})
	})

	r.GET("/files/*path", func(c *gin.Context) {
		// Gin includes the leading slash in wildcard params by default.
		p := c.Param("path")
		c.String(http.StatusOK, "file path=%q", p)
	})

	r.Run(":8080")
}
