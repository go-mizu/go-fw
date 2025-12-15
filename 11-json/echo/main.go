package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type Payload struct {
	Message string `json:"message"`
}

func main() {
	e := echo.New()

	e.POST("/echo", func(c echo.Context) error {
		var p Payload
		if err := c.Bind(&p); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid json")
		}

		return c.JSON(http.StatusOK, p)
	})

	e.Start(":8080")
}
