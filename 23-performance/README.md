# Performance model and benchmarks

Performance is not defined by a single number. Throughput and latency are outcomes of deeper properties such as how long the request path is inside the framework, where allocations occur, how much state is created or reset per request, and whether data is streamed or buffered. Benchmarks are only meaningful when you understand which parts of the system they exercise and which parts they ignore.

This section explains performance in terms of execution model rather than rankings. The focus is on the request path inside each framework, the sources of allocation, the cost of abstraction layers, and the limits of microbenchmarks. All examples use a minimal `GET /ping` handler measured with `go test -bench`, with no logging and no middleware unless explicitly shown. The intent is to isolate framework overhead, not application behavior.

## net/http

```go
package bench

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func handler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("pong"))
}

func BenchmarkNetHTTP(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ping", handler)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec.Body.Reset()
		mux.ServeHTTP(rec, req)
	}
}
```

The standard library establishes the baseline execution model. A request is parsed, routed through `ServeMux`, passed directly to a handler function, and written to a response writer. There is very little indirection and almost no hidden state. Each request allocates what is required for request parsing and response writing, and nothing more unless user code introduces it.

There is no context pooling, no handler wrapping beyond what is strictly necessary, and no framework level bookkeeping. This makes `net/http` predictable and easy to reason about. Its performance characteristics are stable and transparent, which is why it is often used as a reference point when evaluating other frameworks.

## Chi

```go
package bench

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func BenchmarkChi(b *testing.B) {
	r := chi.NewRouter()
	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec.Body.Reset()
		r.ServeHTTP(rec, req)
	}
}
```

Chi adds a routing and middleware layer on top of `net/http` while preserving the same core request and response types. Each request traverses a route tree, resolves parameters, and walks a middleware chain before reaching the handler. This adds a small amount of overhead compared to raw `ServeMux`.

Because Chi does not pool request contexts or response writers, allocations are slightly higher than the standard library but still modest. The cost comes primarily from route matching and middleware traversal rather than from object management. Chi’s performance model favors clarity and composability over aggressive optimization, which keeps behavior predictable even as complexity grows.

## Gin

```go
package bench

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func BenchmarkGin(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong")
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec.Body.Reset()
		r.ServeHTTP(rec, req)
	}
}
```

Gin introduces a custom context type that is pooled and reused across requests. On each request, a context object is pulled from a pool, its internal state is reset, middleware is executed, and response metadata such as status and size is tracked explicitly.

This pooling strategy reduces allocations compared to frameworks that create fresh context objects for every request. The tradeoff is additional bookkeeping to reset state correctly and manage lifecycle transitions. For trivial handlers with no middleware, this overhead can make Gin slightly slower than raw `net/http`. As middleware stacks grow, the pooled model often becomes more efficient because it amortizes allocation costs.

Gin’s performance profile is shaped more by its lifecycle management than by routing complexity.

## Echo

```go
package bench

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func BenchmarkEcho(b *testing.B) {
	e := echo.New()
	e.GET("/ping", func(c echo.Context) error {
		return c.String(200, "pong")
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec.Body.Reset()
		e.ServeHTTP(rec, req)
	}
}
```

Echo uses a model similar to Gin, with a custom context abstraction and internal state tracking. Context objects are reused, handlers are invoked through interface calls, and response state is captured by the framework rather than inferred from the response writer.

The cost per request includes context reuse, handler dispatch, and error handling hooks. In practice, Echo’s performance is usually close to Gin’s. Differences tend to appear when error handling, logging, or recovery behavior is enabled, rather than in the routing or handler invocation itself.

## Fiber

```go
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
```

Fiber is built on `fasthttp`, which uses a fundamentally different execution model than `net/http`. It avoids `context.Context`, aggressively reuses objects, and buffers responses by default. This minimizes allocations and system calls for simple request paths.

As a result, Fiber often shows very high throughput and low allocation counts in microbenchmarks. However, these numbers reflect a different set of tradeoffs. Compatibility with standard libraries is reduced, streaming semantics differ, and many tools designed for `net/http` cannot be used directly.

Benchmarks that compare Fiber directly to `net/http` are not measuring the same thing. They compare two different network stacks with different guarantees and expectations.

## Mizu

```go
package bench

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu"
)

func BenchmarkMizu(b *testing.B) {
	app := mizu.New()
	app.Get("/ping", func(c *mizu.Ctx) error {
		return c.Text(200, "pong")
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec.Body.Reset()
		app.ServeHTTP(rec, req)
	}
}
```

Mizu stays aligned with `net/http` while introducing a thin context layer and structured middleware pipeline. Each request creates a context, traverses middleware, executes the handler, and runs error handling hooks. There is no aggressive pooling by default, and the focus is on correctness, transparency, and predictable behavior.

The additional cost compared to raw `net/http` comes from explicit lifecycle management rather than hidden buffering or object reuse. In practice, performance is usually close to Chi and standard library based routers, especially once middleware is present.

## How to read benchmark numbers

Benchmarks only measure what they execute. A minimal handler benchmark measures routing and dispatch overhead, not database access, serialization, or network latency. Numbers are only comparable when frameworks are configured with similar middleware, similar response behavior, and similar network stacks.

Comparing `fasthttp` based frameworks to `net/http` based ones mixes different abstractions. Looking only at nanoseconds per operation hides allocation behavior, which often matters more under load. The most reliable benchmarks are those that include real handlers and realistic workloads.

In most production systems, framework overhead is dwarfed by I/O, database access, and external services. The difference between frameworks becomes relevant only when the rest of the system is already well optimized.

## What learners should focus on

Performance reflects priorities. Some frameworks optimize for raw throughput by reusing aggressively and limiting abstractions. Others prioritize compatibility, clarity, and predictable behavior. Buffering simplifies APIs but affects streaming. Pooling reduces allocations but increases lifecycle complexity.

The most useful framework is not the one with the best microbenchmark result, but the one whose performance model you understand well enough to reason about behavior under real load.
