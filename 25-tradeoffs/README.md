# Tradeoffs

Tradeoffs are easiest to understand when you see **what code you write** and **what code you cannot avoid writing**. This chapter therefore starts with minimal, comparable code snippets and then synthesizes the differences in a single table, followed by a deep dive that explains why those differences matter over time.

The goal is not to repeat comparisons, but to surface **structural tradeoffs** that persist even as frameworks add features or polish APIs.

## net/http

```go
package main

import (
	"net/http"
)

func ping(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("pong"))
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ping", ping)

	http.ListenAndServe(":8080", mux)
}
```

This is the reference model. Handlers are plain functions. Routing is explicit. Middleware is simple wrapping. There is no hidden lifecycle and no framework state. Everything composes through `http.Handler`. The tradeoff is that nothing is provided for you beyond primitives. Structure, consistency, and safety are your responsibility.

## Chi

```go
package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()
	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	http.ListenAndServe(":8080", r)
}
```

Chi preserves the net/http contract while adding structure. Handlers remain plain functions, middleware is still wrapping, and routing becomes expressive. The tradeoff is a small amount of routing and middleware overhead in exchange for clarity and composability. Chi does not try to manage lifecycle or state for you, so behavior remains visible and testable.

## Gin

```go
package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.New()
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	http.ListenAndServe(":8080", r)
}
```

Gin replaces the handler contract. Handlers are no longer plain functions but methods operating on a pooled context object. This enables convenience APIs and reduces allocations, but it also couples application code to Gin’s lifecycle. The tradeoff is ergonomics and performance versus isolation and portability. Testing and middleware reuse must flow through Gin’s engine.

## Echo

```go
package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()
	e.GET("/ping", func(c echo.Context) error {
		return c.String(http.StatusOK, "pong")
	})

	http.ListenAndServe(":8080", e)
}
```

Echo makes a similar tradeoff to Gin, but keeps a slightly looser abstraction. Handlers still depend on a framework context, but can be tested in isolation by constructing that context manually. The tradeoff sits between Gin and Chi: more structure and batteries included than Chi, but less hidden lifecycle than Gin.

## Fiber

```go
package main

import (
	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()
	app.Get("/ping", func(c *fiber.Ctx) error {
		return c.SendString("pong")
	})

	app.Listen(":8080")
}
```

Fiber opts out of the net/http contract entirely. The execution model, context, middleware, and networking stack are all framework specific. This enables aggressive reuse and high throughput, but it also isolates the application from the standard Go HTTP ecosystem. The tradeoff is performance and simplicity versus interoperability and reuse.

## Mizu

```go
package main

import (
	"net/http"

	"github.com/go-mizu/mizu"
)

func main() {
	app := mizu.New()
	app.Get("/ping", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "pong")
	})

	http.ListenAndServe(":8080", app)
}
```

Mizu deliberately stays on the net/http path while adding a structured request model. Handlers use a context abstraction, but the framework itself remains an `http.Handler` and preserves standard request semantics. The tradeoff is accepting a thin abstraction layer in exchange for consistency across routing, middleware, observability, and testing without breaking ecosystem compatibility.

## Tradeoffs at a glance

| Axis                    | net/http          | Chi               | Gin           | Echo         | Fiber       | Mizu                |
| ----------------------- | ----------------- | ----------------- | ------------- | ------------ | ----------- | ------------------- |
| Handler contract        | Standard          | Standard          | Custom        | Custom       | Custom      | net/http-aligned    |
| Context type            | `context.Context` | `context.Context` | Pooled custom | Custom       | Custom      | net/http + thin ctx |
| Middleware reuse        | Native            | Native            | Adapter       | Adapter      | None        | Native              |
| Testing granularity     | Max               | Max               | Router-only   | Mixed        | Router-only | Max + pipeline      |
| Lifecycle visibility    | Explicit          | Explicit          | Managed       | Semi-managed | Managed     | Explicit            |
| Ecosystem compatibility | Full              | Full              | Partial       | Partial      | Low         | Full                |
| Performance bias        | Predictable       | Predictable       | Pooled        | Pooled       | Aggressive  | Predictable         |

## Deep dive: why these tradeoffs matter

The most durable tradeoff is **contract preservation versus replacement**. Once a framework replaces the handler contract, all application code becomes framework specific. This affects not only routing, but testing, middleware sharing, observability integration, and long term refactoring. Preserving the contract keeps options open.

The second critical tradeoff is **lifecycle visibility**. When request boundaries are explicit, instrumentation, error handling, and debugging are straightforward. When lifecycle is managed internally, behavior becomes easier to use but harder to reason about when something goes wrong. This difference becomes significant in production systems where observability and correctness matter more than convenience.

Performance tradeoffs are often misunderstood. Pooling and buffering can improve microbenchmarks, but they also introduce constraints on how code can be written and tested. In most real systems, framework overhead is dominated by I/O and business logic. Predictability is usually more valuable than peak throughput.

Finally, interoperability determines how systems evolve. Frameworks that remain compatible with net/http can be introduced gradually, coexist with other components, and be removed later if needed. Frameworks that replace core contracts should be chosen deliberately, with a clear understanding that the boundary they introduce is permanent.

The correct framework is therefore not the one with the most features, but the one whose tradeoffs align with the lifetime and integration needs of the system you are building.
