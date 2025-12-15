package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestMizuHandler(t *testing.T) {
	app := mizu.New()

	app.Get("/ping", func(c *mizu.Ctx) error {
		return c.Text(200, "pong")
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Body.String() != "pong" {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}
