package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.New()

	r.GET("/", root)
	r.GET("/users", users)
	r.GET("/users/:id", userByID)

	r.Run(":8080")
}

func root(c *gin.Context) {
	c.String(http.StatusOK, "root")
}

func users(c *gin.Context) {
	c.String(http.StatusOK, "users")
}

func userByID(c *gin.Context) {
	id := c.Param("id")
	c.String(http.StatusOK, "user: %s", id)
}
