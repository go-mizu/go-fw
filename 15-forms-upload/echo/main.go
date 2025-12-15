package main

import (
	"io"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()

	e.POST("/login", func(c echo.Context) error {
		user := c.FormValue("user")
		pass := c.FormValue("pass")

		return c.String(http.StatusOK, "user="+user+" pass="+pass)
	})

	e.POST("/upload", func(c echo.Context) error {
		file, err := c.FormFile("file")
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}

		src, err := file.Open()
		if err != nil {
			return err
		}
		defer src.Close()

		dst, err := os.Create(file.Filename)
		if err != nil {
			return err
		}
		defer dst.Close()

		io.Copy(dst, src)

		return c.String(http.StatusOK, "uploaded "+file.Filename)
	})

	e.Start(":8080")
}
