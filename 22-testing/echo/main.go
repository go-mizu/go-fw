package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestEchoHandler(t *testing.T) {
	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := func(c echo.Context) error {
		return c.String(200, "pong")
	}(c); err != nil {
		t.Fatal(err)
	}

	if rec.Body.String() != "pong" {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}
