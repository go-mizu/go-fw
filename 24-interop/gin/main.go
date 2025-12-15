package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func stdHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("from std handler\n"))
}

func main() {
	r := gin.New()

	r.GET("/std", gin.WrapH(http.HandlerFunc(stdHandler)))

	http.ListenAndServe(":8080", r)
}
