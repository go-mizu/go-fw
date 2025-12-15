# Middleware chaining and execution order

Middleware creates the request pipeline. Routing chooses a handler. Middleware decides what runs before the handler, what runs after it, and who gets to stop the request early. Once more than a few endpoints exist, most behavior lives in middleware: auth, logging, panic recovery, rate limiting, CORS, tracing, metrics, gzip, request IDs, and more. The important part sits underneath the API. Different stacks represent middleware differently, which changes how execution proceeds at runtime and how easy the order is to predict when the project grows.

This section focuses on three concrete questions:

* how middleware is represented in code
* how middleware is executed at runtime
* how a request stops or continues through the chain

The behavior stays intentionally small:

* one route: `GET /`
* two middleware layers
* each middleware prints a message before and after the handler

The observable print order reveals the true execution model. Each example below includes a link to the runnable code, the full `main.go`, and a walkthrough of how the pipeline is executed.

Expected output order when the request reaches the handler:

| Stage | Print      |
| ----- | ---------- |
| 1     | `A before` |
| 2     | `B before` |
| 3     | `handler`  |
| 4     | `B after`  |
| 5     | `A after`  |

## net/http

[`nethttp/main.go`](nethttp/main.go)

```go
package main

import (
	"fmt"
	"net/http"
)

func main() {
	handler := chain(
		http.HandlerFunc(finalHandler),
		middlewareA,
		middlewareB,
	)

	http.ListenAndServe(":8080", handler)
}

func finalHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "handler")
}

type Middleware func(http.Handler) http.Handler

func middlewareA(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("A before")
		next.ServeHTTP(w, r)
		fmt.Println("A after")
	})
}

func middlewareB(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("B before")
		next.ServeHTTP(w, r)
		fmt.Println("B after")
	})
}

func chain(h http.Handler, m ...Middleware) http.Handler {
	for i := len(m) - 1; i >= 0; i-- {
		h = m[i](h)
	}
	return h
}
```

net/http middleware is plain functional composition. Each middleware takes a handler and returns a new handler. The chain is created by wrapping, which produces a nested call stack. The important part is that the order is locked in at build time, before any request arrives.

A small snippet captures the effective nesting:

```go
h := middlewareA(middlewareB(http.HandlerFunc(finalHandler)))
```

When a request arrives, the server calls `h.ServeHTTP`. That enters middleware A. A prints its "before" line, then calls `next.ServeHTTP`, which enters middleware B. B prints its "before" line, then calls its next handler, which reaches the final handler. When the handler returns, execution unwinds back up the call stack, so B prints "after" and then A prints "after".

Early exit is a structural property. A middleware can stop the request by returning without calling `next.ServeHTTP`. Because the pipeline is a real call stack, "after" logic only runs for middleware that already called into the next handler.

Useful mental model:

| Concept         | net/http                    |
| --------------- | --------------------------- |
| Chain built     | wrapper nesting             |
| Continue        | call `next.ServeHTTP`       |
| Stop early      | return without calling next |
| After code runs | only after next returns     |

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

	r.Use(middlewareA)
	r.Use(middlewareB)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "handler")
	})

	http.ListenAndServe(":8080", r)
}

func middlewareA(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("A before")
		next.ServeHTTP(w, r)
		fmt.Println("A after")
	})
}

func middlewareB(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("B before")
		next.ServeHTTP(w, r)
		fmt.Println("B after")
	})
}
```

Chi uses the same middleware type as net/http: `func(http.Handler) http.Handler`. The difference appears in where the chain is assembled. Middleware is registered on the router. At request time, after route matching selects an endpoint handler, Chi wraps that endpoint with the currently applicable middleware stack.

That means Chi delays the final chain composition until it knows which route matched. This enables scoping and inheritance for groups and subrouters. It also means a request to different routes can execute different middleware stacks, even though middleware is registered globally or in nested scopes.

Execution order still follows the wrapper call stack model. Middleware A wraps middleware B wraps the handler. Early exit works the same way as net/http: returning before calling next stops the chain.

A compact picture of what Chi effectively builds at dispatch time:

```go
routeHandler := handlerFor("/")
h := middlewareA(middlewareB(routeHandler))
h.ServeHTTP(w, r)
```

Useful mental model:

| Concept         | Chi                         |
| --------------- | --------------------------- |
| Chain built     | at dispatch time per route  |
| Continue        | call `next.ServeHTTP`       |
| Stop early      | return without calling next |
| After code runs | only after next returns     |

## Gin

[`gin/main.go`](gin/main.go)

```go
package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.New()

	r.Use(middlewareA)
	r.Use(middlewareB)

	r.GET("/", func(c *gin.Context) {
		fmt.Fprintln(c.Writer, "handler")
	})

	r.Run(":8080")
}

func middlewareA(c *gin.Context) {
	fmt.Println("A before")
	c.Next()
	fmt.Println("A after")
}

func middlewareB(c *gin.Context) {
	fmt.Println("B before")
	c.Next()
	fmt.Println("B after")
}
```

Gin uses an index-driven execution model. Middleware and the final handler are stored together as a single ordered slice. The context holds an index into that slice. Calling `c.Next()` advances the index and runs subsequent handlers.

A useful way to think about it:

* the pipeline lives as data: `handlers []HandlerFunc`
* the call stack is simulated: `Next` moves forward and then returns back to the caller

The "before" and "after" pattern works because `c.Next()` blocks until the rest of the chain finishes. Once the handler returns, execution resumes at the line after `c.Next()`.

Early exit occurs when middleware does not call `c.Next()`, or when it aborts the chain. In Gin, aborting usually means setting a flag on the context that prevents further handlers from running.

A compact view of the state machine:

| Step   | Who runs    | What advances |
| ------ | ----------- | ------------- |
| 1      | middlewareA | calls `Next`  |
| 2      | middlewareB | calls `Next`  |
| 3      | handler     | returns       |
| unwind | B resumes   | then returns  |
| unwind | A resumes   | then returns  |

This model makes composition fast at request time because the chain is precomputed, but it introduces an execution state machine inside the context.

## Echo

[`echo/main.go`](echo/main.go)

```go
package main

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()

	e.Use(middlewareA)
	e.Use(middlewareB)

	e.GET("/", func(c echo.Context) error {
		fmt.Fprintln(c.Response(), "handler")
		return nil
	})

	e.Start(":8080")
}

func middlewareA(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		fmt.Println("A before")
		if err := next(c); err != nil {
			return err
		}
		fmt.Println("A after")
		return nil
	}
}

func middlewareB(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		fmt.Println("B before")
		if err := next(c); err != nil {
			return err
		}
		fmt.Println("B after")
		return nil
	}
}
```

Echo uses wrapper composition like net/http, but the handler returns an error, and that error becomes the control channel. Each middleware receives `next` and returns a new handler. The chain forms a real call stack, and "after" logic runs when the call returns successfully.

The difference shows up when something fails. A middleware can stop the chain by returning an error. The dispatcher sees the error and routes it to centralized error handling. That gives a uniform failure path without context flags or abort states.

A minimal pattern for early exit in Echo middleware:

```go
if !ok {
	return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
}
return next(c)
```

This keeps stopping behavior explicit and testable as a plain return value.

## Fiber

[`fiber/main.go`](fiber/main.go)

```go
package main

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	app.Use(middlewareA)
	app.Use(middlewareB)

	app.Get("/", func(c *fiber.Ctx) error {
		fmt.Println("handler")
		return c.SendString("handler")
	})

	app.Listen(":8080")
}

func middlewareA(c *fiber.Ctx) error {
	fmt.Println("A before")
	if err := c.Next(); err != nil {
		return err
	}
	fmt.Println("A after")
	return nil
}

func middlewareB(c *fiber.Ctx) error {
	fmt.Println("B before")
	if err := c.Next(); err != nil {
		return err
	}
	fmt.Println("B after")
	return nil
}
```

Fiber uses an index-driven chain like Gin, driven by `c.Next()`. Middleware and handlers are stored in an ordered slice, and the context advances through the slice.

The main difference from Gin is that error propagation is explicit. `c.Next()` returns an error, and returning a non-nil error stops execution and triggers the framework’s error handling pipeline.

The mechanism remains a state machine inside the context. The context is pooled and reused, so the index and pipeline state must be reset per request. That makes correct lifecycle handling critical for correctness.

## Mizu

[`mizu/main.go`](mizu/main.go)

```go
package main

import (
	"fmt"

	"github.com/go-mizu/mizu"
)

func main() {
	app := mizu.New()

	app.Use(middlewareA)
	app.Use(middlewareB)

	app.Get("/", func(c *mizu.Ctx) error {
		fmt.Println("handler")
		return c.Text(200, "handler")
	})

	app.Listen(":8080")
}

func middlewareA(next mizu.Handler) mizu.Handler {
	return func(c *mizu.Ctx) error {
		fmt.Println("A before")
		if err := next(c); err != nil {
			return err
		}
		fmt.Println("A after")
		return nil
	}
}

func middlewareB(next mizu.Handler) mizu.Handler {
	return func(c *mizu.Ctx) error {
		fmt.Println("B before")
		if err := next(c); err != nil {
			return err
		}
		fmt.Println("B after")
		return nil
	}
}
```

Mizu middleware uses wrapper composition with explicit error returns. Each middleware receives `next` and returns a new handler. Execution follows a normal call stack: enter A, enter B, run handler, unwind back to B, unwind back to A.

Early exit is controlled by returning without calling next, or by returning an error. The error return provides a uniform failure channel that middleware can use to stop execution while still fitting clean composition.

The execution order stays visible in the code structure:

```go
h := middlewareA(middlewareB(finalHandler))
```

## What matters across stacks

Middleware models fall into two families:

| Family         | Frameworks                | Mechanism                        | Continue    | Stop                                |
| -------------- | ------------------------- | -------------------------------- | ----------- | ----------------------------------- |
| wrapping-based | net/http, Chi, Echo, Mizu | call stack via wrapper functions | call `next` | return early or return error        |
| index-based    | Gin, Fiber                | state machine via context index  | call `Next` | skip `Next`, abort, or return error |

Both styles can produce the same observable order. The real difference appears when pipelines grow deep, when early exits become common, and when debugging execution order matters.
