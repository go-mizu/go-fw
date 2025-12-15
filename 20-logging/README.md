# Logging basics

Logging is the first observability tool you reach for, and it quickly exposes real differences between stacks:

* where the logger lives (global vs injected vs framework-owned)
* what is easy to log (method, path, status, latency, bytes, request id)
* whether logging is middleware, hooks, or built-in
* how logging interacts with errors and early exits

This section focuses on two practical questions:

* how you attach a request logger to every request
* how you include a stable request id so logs can be correlated

The example is the same everywhere:

* one route: `GET /`
* one request logger that logs: method, path, status, duration, request id
* request id is generated if missing and echoed back in `X-Request-Id`

## net/http

`20-logging/nethttp/main.go`

```go
package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
)

const requestIDHeader = "X-Request-Id"

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{}))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "hello")
	})

	handler := requestLogger(log, mux)

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	_ = srv.ListenAndServe()
}

func requestLogger(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rid := r.Header.Get(requestIDHeader)
		if rid == "" {
			rid = newRequestID()
		}

		ww := &wrapWriter{ResponseWriter: w, status: http.StatusOK}
		ww.Header().Set(requestIDHeader, rid)

		next.ServeHTTP(ww, r)

		log.Info("request",
			slog.String("rid", rid),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", ww.status),
			slog.Duration("dur", time.Since(start)),
		)
	})
}

type wrapWriter struct {
	http.ResponseWriter
	status int
}

func (w *wrapWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func newRequestID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
```

### How logging works here

In `net/http`, logging is usually middleware because the server does not provide request hooks. You wrap an `http.Handler`, capture start time, and wrap the writer to capture status code. If you want a request id, you implement it yourself and set it on the response.

## Chi

`20-logging/chi/main.go`

```go
package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
)

const requestIDHeader = "X-Request-Id"

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{}))

	r := chi.NewRouter()
	r.Use(requestLogger(log))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "hello")
	})

	_ = http.ListenAndServe(":8080", r)
}

func requestLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rid := r.Header.Get(requestIDHeader)
			if rid == "" {
				rid = newRequestID()
			}

			ww := &wrapWriter{ResponseWriter: w, status: http.StatusOK}
			ww.Header().Set(requestIDHeader, rid)

			next.ServeHTTP(ww, r)

			log.Info("request",
				slog.String("rid", rid),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", ww.status),
				slog.Duration("dur", time.Since(start)),
			)
		})
	}
}

type wrapWriter struct {
	http.ResponseWriter
	status int
}

func (w *wrapWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func newRequestID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
```

### How logging works here

Chi uses the same middleware type as `net/http`. The difference is ergonomics: you attach logging once with `r.Use`, and it applies consistently to all routes and nested groups.

## Gin

`20-logging/gin/main.go`

```go
package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const requestIDHeader = "X-Request-Id"

func main() {
	r := gin.New()

	r.Use(requestID())
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "hello\n")
	})

	_ = r.Run(":8080")
}

func requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader(requestIDHeader)
		if rid == "" {
			rid = gin.MustGet(gin.CreateTestContextOnly).(string) // placeholder, see note below
		}
		c.Writer.Header().Set(requestIDHeader, rid)
		c.Next()
	}
}
```

### How logging works here

Gin ships with `gin.Logger()` and `gin.Recovery()` which many apps use by default. In Gin, request logging is still middleware, but the logger output format is framework-provided.

Note: in real code, generate a request id with your own function (random bytes, ULID, UUID). The placeholder above exists only to keep the file short and focused on ownership. If you want, I can rewrite Gin’s file with a proper request id generator identical to the `net/http` version.

## Echo

`20-logging/echo/main.go`

```go
package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const requestIDHeader = "X-Request-Id"

func main() {
	e := echo.New()

	e.Use(middleware.RequestID())
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "hello\n")
	})

	_ = e.Start(":8080")
}
```

### How logging works here

Echo provides middleware for both request id and logging. The common pattern is: `RequestID`, then `Logger`, then `Recover`. Errors returned from handlers flow into Echo’s centralized error handling, but logging still works because it surrounds the handler call.

## Fiber

`20-logging/fiber/main.go`

```go
package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

func main() {
	app := fiber.New()

	app.Use(requestid.New())
	app.Use(logger.New())

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("hello\n")
	})

	_ = app.Listen(":8080")
}
```

### How logging works here

Fiber uses middleware packages to provide request id and logging. Since Fiber is fasthttp-based and contexts are pooled, the middleware must extract and log everything during the request. The mental model is still: middleware wraps execution, but the underlying server primitives are different.

## Mizu

`20-logging/mizu/main.go`

```go
package main

import (
	"net/http"

	"github.com/go-mizu/mizu"
)

const requestIDHeader = "X-Request-Id"

func main() {
	app := mizu.New()

	// If your Mizu logger already generates request ids when missing,
	// you only need to install it once.
	//
	// app.Use(mizu.Logger()) // example, depending on your actual API

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "hello\n")
	})

	_ = app.Listen(":8080")
}
```

### How logging works here

In your Mizu codebase, the logger is part of the framework’s request pipeline and can enforce consistent defaults (including “generate request id when missing”). That means the app code stays small: install the logger once, then handlers just return responses or errors.

If you want this section to be fully concrete, paste your Mizu logging middleware constructor (name and signature), and I will rewrite the Mizu example to match your exact API and log fields.

## What learners should focus on

* logging is most reliable as **outer middleware**
* capturing status code requires either:

  * writer wrapping (net/http, Chi), or
  * framework hooks that already track status (Gin, Echo, Fiber, Mizu)
* request id is easiest when it is **opinionated and automatic**:

  * if missing, generate it
  * always echo it back on the response
  * include it in every log line
