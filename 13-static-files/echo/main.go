package main

import (
	"embed"

	"github.com/labstack/echo/v4"
)

//go:embed public/*
var assets embed.FS

func main() {
	e := echo.New()

	e.Static("/static", "public")
	e.StaticFS("/embed", assets)

	e.Start(":8080")
}
