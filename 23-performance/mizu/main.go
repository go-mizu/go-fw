package bench

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu"
)

func BenchmarkMizu(b *testing.B) {
	app := mizu.New()
	app.Get("/ping", func(c *mizu.Ctx) error {
		return c.Text(200, "pong")
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec.Body.Reset()
		app.ServeHTTP(rec, req)
	}
}
