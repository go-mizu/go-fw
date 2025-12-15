# Short-circuiting and early exits

In real applications, many requests should never reach the final handler. Authentication failures, missing headers, invalid payloads, rate limits, feature flags, and maintenance windows all need a reliable way to stop execution early. Short-circuiting is the mechanism that enforces this stop.

The important detail is not whether a framework can stop a request. All of them can. The important detail is *how* the stop is enforced, *where* the signal lives, and *what guarantees exist* about what code may still run afterward.

This section uses a minimal setup:

* one route: `GET /`
* one middleware that rejects the request
* the final handler must never run

The examples show the full `main.go`, followed by a technical explanation of how execution is terminated inside the framework.

## net/http

[`nethttp/main.go`](nethttp/main.go)

```go
package main

import (
	"net/http"
)

func main() {
	handler := deny(
		http.HandlerFunc(finalHandler),
	)

	http.ListenAndServe(":8080", handler)
}

func finalHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("handler reached"))
}

func deny(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("denied"))
	})
}
```

Short-circuiting in net/http is a direct consequence of how middleware is composed. Middleware wraps a handler and decides whether to call it. The wrapper is the enforcement mechanism.

When the request arrives, the server calls the outermost handler. That handler writes a response and returns. Because it never calls `next.ServeHTTP`, no other handler is invoked. There is no signal, no flag, and no framework-managed state involved.

Once the middleware returns, the server considers the request complete and flushes the response. The final handler is unreachable by construction.

The guarantee is structural:

* execution proceeds only by calling `next.ServeHTTP`
* if that call never happens, nothing downstream can run

This property makes net/http short-circuiting easy to audit. The stop is visible in code and enforced by normal control flow.

## Chi

[`chi/main.go`](chi/main.go)

```go
package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()

	r.Use(deny)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("handler reached"))
	})

	http.ListenAndServe(":8080", r)
}

func deny(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("denied"))
	})
}
```

Chi inherits net/http’s wrapping model. Middleware is applied by wrapping handlers, and short-circuiting works the same way.

At request time, Chi resolves the route and builds a handler chain by wrapping the route handler with all applicable middleware. When the deny middleware runs, it writes a response and returns without calling `next.ServeHTTP`. The wrapped handler chain never progresses further.

No router state is modified. No abort mechanism exists. The stop is enforced by the absence of a function call.

The guarantee matches net/http:

* middleware controls execution by choosing whether to call next
* downstream handlers cannot run unless explicitly invoked

## Gin

[`gin/main.go`](gin/main.go)

```go
package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.New()

	r.Use(deny)

	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "handler reached")
	})

	r.Run(":8080")
}

func deny(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"error": "denied",
	})
}
```

Gin uses an index-based execution model. Middleware and handlers are stored in a single slice, and the context holds an index pointing to the current position in that slice.

When middleware runs, execution continues only if `c.Next()` is called. In this example, the middleware does not call `Next`. Instead, it calls `AbortWithStatusJSON`.

That call performs three actions:

* writes the response
* sets an internal aborted flag on the context
* prevents the index from advancing further

When control returns to the dispatcher, the aborted flag is checked and the remaining handlers are skipped.

The stop is not implicit. Writing a response alone does not stop execution. The framework enforces the stop only if the abort mechanism is triggered correctly.

The guarantee is conditional:

* execution stops only if `Abort` is called
* forgetting to abort allows execution to continue even after writing a response

This makes short-circuiting more powerful but also easier to misuse in security-sensitive middleware.

## Echo

[`echo/main.go`](echo/main.go)

```go
package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()

	e.Use(deny)

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "handler reached")
	})

	e.Start(":8080")
}

func deny(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		return echo.NewHTTPError(http.StatusUnauthorized, "denied")
	}
}
```

Echo enforces short-circuiting through error returns. Middleware wraps the next handler and returns an error instead of calling it.

When the deny middleware returns an error, execution stops immediately. The dispatcher does not call the next handler. Instead, control transfers to the centralized error handler, which produces the response.

The stop signal is explicit and type-checked. Returning an error ends the chain. There is no separate abort flag and no reliance on side effects.

The guarantee is strong:

* once an error is returned, no further middleware or handlers run
* the error path is visible in the function signature

This model makes early exits easy to trace and test.

## Fiber

[`fiber/main.go`](fiber/main.go)

```go
package main

import (
	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	app.Use(deny)

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("handler reached")
	})

	app.Listen(":8080")
}

func deny(c *fiber.Ctx) error {
	return c.Status(401).SendString("denied")
}
```

Fiber also relies on error returns to stop execution, but the underlying execution model is index-based like Gin.

Middleware and handlers live in a slice. Calling `c.Next()` advances execution. In this example, `c.Next()` is never called. Instead, the middleware returns an error.

The dispatcher sees the error and terminates the request. Remaining handlers are skipped, and the response is sent.

Because Fiber contexts are pooled, the framework resets execution state after the request completes. The stop is enforced by the error return rather than by an abort flag.

The guarantee is clear:

* returning an error stops execution
* downstream handlers cannot run without `c.Next()`

## Mizu

[`mizu/main.go`](mizu/main.go)

```go
package main

import (
	"net/http"

	"github.com/go-mizu/mizu"
)

func main() {
	app := mizu.New()

	app.Use(deny)

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "handler reached")
	})

	app.Listen(":8080")
}

func deny(next mizu.Handler) mizu.Handler {
	return func(c *mizu.Ctx) error {
		return c.Text(http.StatusUnauthorized, "denied")
	}
}
```

Mizu uses wrapping-based middleware with an error-returning handler contract. The middleware decides whether to call `next`. In this example, it returns immediately with a response.

Because handlers return `error`, returning early naturally terminates execution. There is no abort flag and no index to manage. Execution stops by normal call stack unwinding.

The stop is enforced structurally:

* `next` is never called
* the returned error ends the chain
* downstream handlers remain unreachable

This combines the clarity of net/http wrapping with the explicit failure channel of Echo.

## Comparing short-circuit behavior

| Framework | Stop mechanism     | How stop is enforced    |
| --------- | ------------------ | ----------------------- |
| net/http  | do not call `next` | call stack structure    |
| Chi       | do not call `next` | call stack structure    |
| Gin       | `Abort*` methods   | context flag + index    |
| Echo      | return error       | dispatcher checks error |
| Fiber     | return error       | dispatcher checks error |
| Mizu      | return error       | call stack + error      |

The key difference lies in who enforces the stop.

* Wrapping-based models rely on normal control flow
* Index-based models require explicit signaling to the framework

This distinction matters when auditing authentication, authorization, and other critical middleware.
