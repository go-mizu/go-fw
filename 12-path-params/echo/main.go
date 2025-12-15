package main

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()

	e.GET("/users/:id", func(c echo.Context) error {
		idStr := c.Param("id")

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
		}

		return c.JSON(http.StatusOK, map[string]any{"id": id})
	})

	e.GET("/files/*", func(c echo.Context) error {
		// Echo exposes wildcard captures with Param("*")
		p := c.Param("*")
		return c.String(http.StatusOK, "file path="+p)
	})

	e.Start(":8080")
}
