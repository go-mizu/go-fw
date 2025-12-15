package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chiprom "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	r := chi.NewRouter()

	r.Use(chiprom.RequestID)
	r.Use(chiprom.RealIP)

	// Chi does not ship Prometheus metrics by default.
	// This example assumes a standard chi Prometheus middleware is used.
	r.Use(chiprom.NewWrapResponseWriter)
	r.Use(chiprom.NewCompressor)

	r.Handle("/metrics", promhttp.Handler())

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok\n"))
	})

	http.ListenAndServe(":8080", r)
}
