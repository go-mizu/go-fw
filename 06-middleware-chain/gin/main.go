package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.New()

	r.Use(middlewareA)
	r.Use(middlewareB)

	r.GET("/", func(c *gin.Context) {
		fmt.Fprintln(c.Writer, "handler")
	})

	r.Run(":8080")
}

func middlewareA(c *gin.Context) {
	fmt.Println("A before")
	c.Next()
	fmt.Println("A after")
}

func middlewareB(c *gin.Context) {
	fmt.Println("B before")
	c.Next()
	fmt.Println("B after")
}
