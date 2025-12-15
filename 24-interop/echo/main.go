package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func stdHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("from std handler\n"))
}

func main() {
	e := echo.New()

	e.GET("/std", echo.WrapHandler(http.HandlerFunc(stdHandler)))

	http.ListenAndServe(":8080", e)
}
