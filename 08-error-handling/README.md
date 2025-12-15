# Error handling and panic recovery

Once a request can stop early, failure handling becomes the next hard requirement. Some failures are part of normal control flow: missing input, invalid state, rejected auth, conflict, rate limit. Other failures come from programmer mistakes or broken assumptions: nil pointer dereference, out-of-bounds slice access, double-close, unexpected type assertion, and deliberate `panic`.

This section separates those two failure modes:

* expected failures, represented as errors
* unexpected failures, represented as panics

The focus stays on mechanics: where a failure is observed, who turns it into an HTTP response, what runs after the failure, and what safety guarantees exist around response writing.

Scenario:

* `GET /error` produces an error response
* `GET /panic` panics
* the process stays alive and a response is produced

## net/http

[`nethttp/main.go`](nethttp/main.go)

```go
package main

import (
	"fmt"
	"net/http"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /error", errorHandler)
	mux.HandleFunc("GET /panic", panicHandler)

	handler := recoverMiddleware(mux)

	http.ListenAndServe(":8080", handler)
}

func errorHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
	fmt.Fprintln(w, "bad request")
}

func panicHandler(w http.ResponseWriter, r *http.Request) {
	panic("something went wrong")
}

func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintln(w, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
```

net/http uses a write-to-response model for expected failures. A handler decides status code and body and writes them directly. That makes error handling entirely local to each handler unless a shared helper is introduced.

Panic recovery happens at the boundary around request execution. A panic unwinds the stack until a deferred function catches it with `recover`. The recovery wrapper then writes a 500 response.

The placement of the recovery wrapper determines what gets protected. Wrapping the mux protects handlers and anything inside the mux. Wrapping the entire server handler protects routing plus all middleware. The outermost wrapper becomes the last safety net.

One technical edge matters for correctness: response commitment. If a handler writes headers or body before panicking, a later recovery wrapper may be unable to change the status code to 500, because the response may already be committed.

A small guard pattern often appears in real servers:

```go
// pseudo: if headers already sent, skip writing a second response
```

Core properties:

| Failure type     | How it surfaces      | Who writes the response |
| ---------------- | -------------------- | ----------------------- |
| expected failure | handler decides      | handler                 |
| panic            | recovered by wrapper | recovery wrapper        |

## Chi

[`chi/main.go`](chi/main.go)

```go
package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()

	r.Use(recoverMiddleware)

	r.Get("/error", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "bad request")
	})

	r.Get("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("something went wrong")
	})

	http.ListenAndServe(":8080", r)
}

func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintln(w, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
```

Chi keeps net/http semantics for errors and panics, then provides structure to attach cross-cutting behavior uniformly. Recovery is expressed as normal net/http middleware and applied at the router level, which makes it harder to forget in larger applications.

Expected failures still follow the same write-to-response pattern. A handler writes a 400 and returns.

Panic recovery behaves like the net/http wrapper model, with a key benefit: middleware placement is explicit and scoped. A router-level `Use` wraps route handlers consistently, including nested routes when groups are used.

Core properties:

| Failure type     | How it surfaces         | Who writes the response |
| ---------------- | ----------------------- | ----------------------- |
| expected failure | handler decides         | handler                 |
| panic            | recovered by middleware | recovery middleware     |

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

	r.Use(gin.Recovery())

	r.GET("/error", func(c *gin.Context) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": "bad request",
		})
	})

	r.GET("/panic", func(c *gin.Context) {
		panic("something went wrong")
	})

	r.Run(":8080")
}
```

Gin separates two flows: expected failures flow through response-writing on the context, while panics flow through recovery middleware.

For expected failures, handlers use abort-style APIs. The abort both writes a response and signals the framework to stop the remaining chain.

For panics, `gin.Recovery()` wraps the handler chain. When a panic occurs, recovery catches it, logs stack information, and writes a 500 response.

The internal mechanism is tightly coupled to the context pipeline:

* middleware and handlers run in an ordered list
* abort changes control flow state inside the context
* recovery wraps execution so panics unwind into the recovery point

A practical detail shows up when combining abort and recovery: writing a response and then panicking later in the chain can create partially written responses. Recovery can only write a clean 500 response when the response has not been committed.

Core properties:

| Failure type     | How it surfaces           | Who writes the response |
| ---------------- | ------------------------- | ----------------------- |
| expected failure | abort + write via context | handler                 |
| panic            | recovered by middleware   | recovery middleware     |

## Echo

[`echo/main.go`](echo/main.go)

```go
package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	e := echo.New()

	e.Use(middleware.Recover())

	e.GET("/error", func(c echo.Context) error {
		return echo.NewHTTPError(http.StatusBadRequest, "bad request")
	})

	e.GET("/panic", func(c echo.Context) error {
		panic("something went wrong")
	})

	e.Start(":8080")
}
```

Echo routes expected failures through the handler return value. A handler returns an `error`, and the central dispatcher converts that error into an HTTP response.

Panic recovery is middleware. The recovery middleware catches the panic and turns it into an error that goes through the same centralized error handler.

This produces a single conversion point: the global error handler. That simplifies consistency. Status codes, error formatting, and logging can live in one place, while handlers can focus on returning meaningful errors.

A common pattern in larger Echo apps:

* middleware returns a typed HTTP error for expected failures
* unexpected panics become a generic 500 error through recovery
* the centralized error handler decides what to reveal to clients

Core properties:

| Failure type     | How it surfaces      | Who writes the response   |
| ---------------- | -------------------- | ------------------------- |
| expected failure | returned error       | centralized error handler |
| panic            | recovered into error | centralized error handler |

## Fiber

[`fiber/main.go`](fiber/main.go)

```go
package main

import (
	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(500).SendString("internal server error")
		},
	})

	app.Get("/error", func(c *fiber.Ctx) error {
		return fiber.NewError(400, "bad request")
	})

	app.Get("/panic", func(c *fiber.Ctx) error {
		panic("something went wrong")
	})

	app.Listen(":8080")
}
```

Fiber centralizes expected failures through the configured `ErrorHandler`. A handler returns an error, Fiber routes it to the global error handler, and the error handler writes the final response.

Panic recovery is handled within the request execution pipeline, converting the panic into an error that reaches the same error handler. This yields a single place for response formatting.

Because Fiber pools contexts, the framework must also guarantee cleanup after failures. The important property for users is consistency: expected errors and panics both flow toward the error handler, and the request completes with a response.

Core properties:

| Failure type     | How it surfaces      | Who writes the response |
| ---------------- | -------------------- | ----------------------- |
| expected failure | returned error       | global error handler    |
| panic            | recovered into error | global error handler    |

## Mizu

[`mizu/main.go`](mizu/main.go)

```go
package main

import (
	"errors"
	"net/http"

	"github.com/go-mizu/mizu"
)

func main() {
	app := mizu.New()

	app.Get("/error", func(c *mizu.Ctx) error {
		return mizu.HTTPError{
			Status: http.StatusBadRequest,
			Err:    errors.New("bad request"),
		}
	})

	app.Get("/panic", func(c *mizu.Ctx) error {
		panic("something went wrong")
	})

	app.Listen(":8080")
}
```

Mizu routes failure through the handler error return. A handler returns an error value. When that error carries HTTP semantics, the framework converts it into a response with the matching status code. When the error carries no HTTP semantics, the framework converts it into a 500 response.

Panic recovery is part of the request execution pipeline. A panic is recovered, logged, and converted into an internal server error response. Expected failures and panics converge into the same conversion logic.

That convergence creates a stable rule: handlers focus on returning errors, and the framework focuses on response conversion and post-failure guarantees, including preventing process termination and keeping request teardown consistent.

Core properties:

| Failure type     | How it surfaces      | Who writes the response |
| ---------------- | -------------------- | ----------------------- |
| expected failure | returned error       | centralized conversion  |
| panic            | recovered into error | centralized conversion  |

## Comparing failure models

| Framework | Expected failure path     | Panic path             | Central conversion point           |
| --------- | ------------------------- | ---------------------- | ---------------------------------- |
| net/http  | handler writes response   | outer recovery wrapper | optional, user-built               |
| Chi       | handler writes response   | recovery middleware    | optional, user-built               |
| Gin       | handler aborts and writes | recovery middleware    | split between handler and recovery |
| Echo      | handler returns error     | recovered into error   | global error handler               |
| Fiber     | handler returns error     | recovered into error   | configured error handler           |
| Mizu      | handler returns error     | recovered into error   | framework conversion pipeline      |

What to watch for when implementing real services:

* response commitment before failure, especially for panics after partial writes
* consistent status code mapping for domain errors
* consistent logging and client-visible messages for unexpected panics
