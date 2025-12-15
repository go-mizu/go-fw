package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.New()

	r.POST("/login", func(c *gin.Context) {
		user := c.PostForm("user")
		pass := c.PostForm("pass")

		c.String(http.StatusOK, "user=%s pass=%s", user, pass)
	})

	r.POST("/upload", func(c *gin.Context) {
		file, err := c.FormFile("file")
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		c.SaveUploadedFile(file, "./"+file.Filename)
		c.String(http.StatusOK, "uploaded %s", file.Filename)
	})

	r.Run(":8080")
}
