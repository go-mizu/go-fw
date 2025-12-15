package bench

import (
	"io"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func BenchmarkFiber(b *testing.B) {
	app := fiber.New()
	app.Get("/ping", func(c *fiber.Ctx) error {
		return c.SendString("pong")
	})

	req, _ := http.NewRequest(http.MethodGet, "/ping", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, _ := app.Test(req)
		io.ReadAll(resp.Body)
		resp.Body.Close()
	}
}
