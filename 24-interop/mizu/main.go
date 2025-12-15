package main

import (
	"net/http"

	"github.com/go-mizu/mizu"
)

func stdHandler(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("from std handler\n"))
}

func main() {
	app := mizu.New()

	app.Handle("GET", "/std", func(c *mizu.Ctx) error {
		stdHandler(c.Writer(), c.Request())
		return nil
	})

	_ = http.ListenAndServe(":8080", app)
}
