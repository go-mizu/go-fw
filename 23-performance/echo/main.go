package bench

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func BenchmarkEcho(b *testing.B) {
	e := echo.New()
	e.GET("/ping", func(c echo.Context) error {
		return c.String(200, "pong")
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec.Body.Reset()
		e.ServeHTTP(rec, req)
	}
}
