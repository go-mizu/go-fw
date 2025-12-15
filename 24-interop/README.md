# net/http interoperability

Interoperability answers a practical, long term question: can a framework participate naturally in the Go HTTP ecosystem, or does it require adapters, shims, or architectural boundaries to coexist with existing code.

In Go, interoperability is not an abstract concept. It is largely determined by whether a framework aligns with `net/http` contracts. If a framework is an `http.Handler`, it can be mounted into a standard `http.Server`. If it accepts `http.Handler`, it can host existing libraries. If its middleware model matches `net/http`, middleware can be reused without translation. When these properties hold, third party libraries work unchanged and systems remain composable.

This section evaluates interoperability through concrete checks. A standard `http.Handler` is mounted inside the framework. The framework itself is mounted inside a standard `http.Server`. Existing `net/http` middleware is reused without rewriting. These checks reveal how much friction exists at the integration boundary.

## net/http

```go
package main

import (
	"net/http"
)

func stdHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("from std handler\n"))
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/std", stdHandler)

	http.ListenAndServe(":8080", mux)
}
```

The standard library defines the reference model. Everything revolves around `http.Handler`. Servers accept handlers, routers are handlers, and middleware is just a function that wraps a handler and returns another handler.

Because this is the base abstraction, all `net/http` libraries are interoperable by definition. There is no adaptation cost, no translation layer, and no impedance mismatch. This simplicity is why `net/http` remains the foundation of most Go web systems, even when higher level frameworks are used on top.

## Chi

```go
package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func stdHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("from std handler\n"))
}

func main() {
	r := chi.NewRouter()

	r.Handle("/std", http.HandlerFunc(stdHandler))

	http.ListenAndServe(":8080", r)
}
```

Chi preserves the `net/http` contract completely. The router itself implements `http.Handler`, and routes accept standard handlers directly. This means a Chi router can be mounted anywhere a standard handler is expected, and any standard handler can be mounted inside Chi without modification.

Middleware compatibility follows naturally. Middleware written for `net/http` can wrap a Chi router, and Chi middleware can wrap standard handlers. Third party libraries that expect `http.Handler` work unchanged. From an interoperability perspective, Chi behaves as a structured extension of `net/http`, not a replacement.

This property makes Chi particularly easy to introduce into existing codebases incrementally.

## Gin

```go
package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func stdHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("from std handler\n"))
}

func main() {
	r := gin.New()

	r.GET("/std", gin.WrapH(http.HandlerFunc(stdHandler)))

	http.ListenAndServe(":8080", r)
}
```

Gin implements `http.Handler`, so it can be mounted inside a standard `http.Server`. This allows Gin applications to run within the `net/http` server infrastructure without issue.

However, Gin does not accept standard handlers directly. Handler signatures use `*gin.Context`, so standard handlers must be adapted using `gin.WrapH`. This adapter bridges between Gin’s context model and `http.ResponseWriter` and `*http.Request`.

The result is functional but not seamless. Standard middleware written for `net/http` cannot be reused directly inside Gin without similar adapters. Interoperability exists, but it lives behind explicit wrapping boundaries. Gin remains close to `net/http`, but it does not sit fully on the same abstraction path.

## Echo

```go
package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func stdHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("from std handler\n"))
}

func main() {
	e := echo.New()

	e.GET("/std", echo.WrapHandler(http.HandlerFunc(stdHandler)))

	http.ListenAndServe(":8080", e)
}
```

Echo also implements `http.Handler`, which allows it to be served by a standard `http.Server`. Like Gin, Echo uses its own handler signature and context abstraction, so standard handlers must be adapted before being mounted.

Echo provides explicit adapter functions such as `WrapHandler`, which makes the interoperability boundary clear and intentional. Net/http middleware cannot be reused directly without adaptation, but the required translation is well defined and supported.

Echo’s interoperability model is explicit rather than implicit. Integration is possible, but the framework does not attempt to hide the boundary between its abstractions and the standard library.

## Fiber

```go
package main

import (
	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	app.Get("/std", func(c *fiber.Ctx) error {
		return c.SendString("from fiber\n")
	})

	app.Listen(":8080")
}
```

Fiber is not built on `net/http`. It does not implement `http.Handler`, and it cannot be mounted inside a standard `http.Server`. Its handler and middleware model is entirely separate, built on top of `fasthttp`.

As a result, standard `net/http` handlers and middleware cannot be reused directly. Integration with `net/http` based systems requires adapters, reverse proxies, or running separate servers. This is a deliberate design choice that prioritizes performance and control over compatibility.

Fiber represents a clean break from the Go HTTP ecosystem rather than an extension of it.

## Mizu

```go
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
```

Mizu is designed to remain fully on the `net/http` path. It implements `http.Handler`, accepts standard handlers directly, and can be wrapped by standard middleware without translation. Existing `net/http` libraries work unchanged when embedded in a Mizu application.

At the same time, Mizu provides higher level abstractions such as structured routing, middleware chaining, and context helpers without replacing the underlying contracts. This allows applications to grow in complexity while preserving interoperability with the broader ecosystem.

Interoperability is treated as a first class constraint rather than a compatibility layer.

## What learners should focus on

Interoperability determines how easily a system evolves over time. Frameworks that preserve `net/http` contracts allow code to be reused across services, libraries to be shared without adapters, and components to be composed freely. They also make it possible to remove or replace the framework later without rewriting the entire application.

Frameworks that diverge from `net/http` can offer performance or ergonomics benefits, but they narrow the integration surface. Understanding where a framework sits on this spectrum is essential for making architectural decisions that remain flexible as systems grow.
