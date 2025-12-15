package main

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	ginprom "github.com/zsais/go-gin-prometheus"
)

func main() {
	r := gin.New()

	p := ginprom.NewPrometheus("gin")
	p.Use(r)

	r.GET("/", func(c *gin.Context) {
		c.String(200, "ok\n")
	})

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	r.Run(":8080")
}
