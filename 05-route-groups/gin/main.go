package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.New()

	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "root")
	})

	api := r.Group("/api/v1")
	{
		api.GET("/users", func(c *gin.Context) {
			c.String(http.StatusOK, "api v1 users")
		})
		api.GET("/users/:id", func(c *gin.Context) {
			c.String(http.StatusOK, "api v1 user: %s", c.Param("id"))
		})
	}

	admin := r.Group("/admin", requireToken("letmein"))
	{
		admin.GET("/dashboard", func(c *gin.Context) {
			c.String(http.StatusOK, "admin dashboard")
		})
	}

	r.Run(":8080")
}

func requireToken(token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("X-Admin-Token") != token {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}
