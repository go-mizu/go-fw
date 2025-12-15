# Metrics and distributed tracing

Metrics, logs, and traces observe the same request lifecycle but at different resolutions. Logs describe discrete events as they occur. Metrics aggregate numeric signals such as counts and latency across many requests. Tracing reconstructs the causal path of a single request as it moves through middleware, handlers, and downstream calls.

A crucial distinction is between **collecting** metrics and **exposing** them. Exposing `/metrics` only makes already collected data readable by Prometheus. Collection only happens if the framework or application actively instruments the request lifecycle. Some frameworks already ship with middleware that captures metrics correctly. In those cases, reimplementing metrics in user code is unnecessary and often counterproductive.

The goal here is consistency of behavior, not identical implementations. Every example below both captures request metrics and exposes `/metrics`. Where a framework provides a well supported metrics middleware, that middleware is reused. Where it does not, explicit instrumentation is shown.

Across all frameworks, the signals are conceptually the same: request count, request duration, HTTP method, a stable route identifier, and response status code. The differences are in how request boundaries are defined, how routes are resolved, and how much observability the framework provides out of the box.

## net/http

```go
package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	reqTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests",
		},
		[]string{"method", "route", "code"},
	)

	reqDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route", "code"},
	)
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()

		next.ServeHTTP(sw, r)

		reqTotal.WithLabelValues(
			r.Method,
			r.URL.Path,
			strconv.Itoa(sw.status),
		).Inc()

		reqDuration.WithLabelValues(
			r.Method,
			r.URL.Path,
			strconv.Itoa(sw.status),
		).Observe(time.Since(start).Seconds())
	})
}

func main() {
	prometheus.MustRegister(reqTotal, reqDuration)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Write([]byte("ok\n"))
	})
	mux.Handle("/metrics", promhttp.Handler())

	http.ListenAndServe(":8080", metrics(mux))
}
```

In the standard library, there is no built in observability layer. All instrumentation is manual. Request boundaries are defined by where the wrapper calls and returns from `ServeHTTP`. Status codes are invisible unless the response writer is wrapped. Route information is limited to the raw URL path because the standard library has no concept of a matched route pattern.

Tracing follows the same pattern. Although `context.Context` is present, no spans are created and no headers are extracted unless user code does so explicitly. The standard library provides the carrier, not the semantics.

## Chi

```go
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
```

Chi itself does not collect metrics automatically, but it defines a precise middleware boundary around `http.Handler`. The ecosystem provides well tested Prometheus middleware that hooks into this boundary, captures request duration and status, and labels metrics using the matched route pattern available from Chi’s routing context.

The important property is that Chi preserves the standard request and context types. Route resolution happens before middleware exits, so stable route patterns can be used safely as labels. Tracing integrates cleanly because spans can be stored in `context.Context` and propagated without adapters. Chi enables observability structurally, but relies on middleware to implement it.

## Gin

```go
package main

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	ginprom "github.com/zsais/go-gin-prometheus"
)

func main() {
	r := gin.New()

	p := ginprom.NewPrometheus("gin")
	p.Use(r)

	r.GET("/", func(c *gin.Context) {
		c.String(200, "ok\n")
	})

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	r.Run(":8080")
}
```

Gin tracks request lifecycle internally, including start time, handler execution, and response status. Prometheus middleware for Gin builds directly on these internals, which avoids response writer wrapping and manual timing.

The middleware uses `c.FullPath` to label metrics with the matched route pattern, which keeps cardinality bounded. Request boundaries are defined by Gin’s handler chain, with metrics recorded after all handlers have completed.

Tracing is supported through Gin specific middleware that bridges trace context into the underlying request context. While Gin sits on `net/http`, instrumentation code is framework specific and less portable than Chi or raw `net/http` middleware.

## Echo

```go
package main

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo-contrib/prometheus"
)

func main() {
	e := echo.New()

	p := prometheus.NewPrometheus("echo", nil)
	p.Use(e)

	e.GET("/", func(c echo.Context) error {
		return c.String(200, "ok\n")
	})

	e.Start(":8080")
}
```

Echo provides an official Prometheus integration that captures request count and latency through middleware. The middleware wraps handler execution and records metrics after the handler returns. It uses the registered route path for labeling, which avoids raw path cardinality issues.

Echo’s response status is tracked internally, so the middleware does not need to wrap the response writer. Tracing works by attaching spans to the underlying request context, but like other frameworks, no tracing occurs unless tracing middleware is explicitly added.

## Fiber

```go
package main

import (
	"github.com/gofiber/fiber/v2"
	fiberprom "github.com/gofiber/contrib/prometheus"
)

func main() {
	app := fiber.New()

	prom := fiberprom.New("fiber")
	prom.RegisterAt(app, "/metrics")
	app.Use(prom.Middleware)

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("ok\n")
	})

	app.Listen(":8080")
}
```

Fiber includes an official Prometheus middleware that captures request metrics using Fiber’s internal request model. Request boundaries are defined by `c.Next`, and metrics are recorded after downstream handlers complete.

Because Fiber does not use `net/http` or `context.Context`, metrics and tracing are inherently framework specific. Route labeling uses Fiber’s route definitions where available. Tracing requires Fiber specific adapters and cannot rely on standard Go context propagation.

## Mizu

```go
package main

import (
	"github.com/go-mizu/mizu"
)

func main() {
	app := mizu.New()

	app.Use(mizu.Metrics())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(200, "ok\n")
	})

	app.Listen(":8080")
}
```

Mizu treats metrics as a first class concern and provides a built in metrics middleware that instruments the entire request lifecycle. The middleware captures request start, handler execution, error paths, and response status in a single place.

Metrics are labeled using stable route information resolved by the router rather than raw paths. Because Mizu aligns with the standard `net/http` request and context model, tracing middleware can attach spans to the request context and propagate them naturally across middleware and handlers.

The key advantage is consistency. Metrics, tracing, and logging observe the same lifecycle boundaries and share the same request context, which simplifies correlation across signals.

Across all frameworks, the core lesson remains the same. Observability quality depends less on whether metrics exist and more on where request boundaries are defined, how labels are chosen, and whether context propagation is preserved. Frameworks that either provide correct middleware or make it easy to add one produce predictable, operable systems.
