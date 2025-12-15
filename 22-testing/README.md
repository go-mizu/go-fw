# Testing handlers, routers, and middleware

Testing exposes the real contract of a framework far more clearly than documentation. It shows what is considered public behavior, what is internal, and what assumptions the framework makes about execution order, state, and lifecycle. In practice, testing answers questions about isolation, determinism, and control that are difficult to infer from examples alone.

Across frameworks, testing typically aims to validate the same things. Handlers should be testable without starting a real server. Routing behavior should be observable without binding to a port. Middleware side effects such as headers, status codes, or context mutations should be assertable. Errors and panics should be capturable in tests. Tests should be fast, isolated, and free from shared global state.

Although these goals are common, frameworks differ significantly in how easily they are achieved and what tradeoffs they impose.

## net/http

```go
package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func handler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func TestHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	res := rec.Result()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
}
```

In the standard library, handlers are ordinary functions. Testing is a direct function call with a synthetic request and a recorder. There is no hidden state, no global router, and no lifecycle beyond the call itself. This makes handler tests trivial to write and extremely fast.

Routing tests are just as simple. A `ServeMux` is constructed, routes are registered, and `ServeHTTP` is called with a request and recorder. Middleware is tested by explicitly wrapping handlers and asserting on the recorded response. Because everything is explicit and compositional, tests are pure and deterministic.

There is no test mode, no special helpers, and no framework state to reset between tests. This simplicity is the baseline against which all other frameworks can be compared.

## Chi

```go
package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestRouter(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Body.String() != "pong" {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}
```

Chi preserves the `net/http` contract, so testing looks almost identical to the standard library. The router implements `http.Handler`, which means all existing `httptest` helpers work without modification. Routing behavior is exercised by calling `ServeHTTP` on the router, and middleware is tested by attaching it to the router and asserting on the response.

Because Chi does not introduce a custom handler signature or context type, there is very little framework specific testing knowledge required. Request context propagation, cancellation, and deadlines behave exactly as they do in raw `net/http`. The main difference from the standard library is that routing logic is more expressive, not that testing is more complex.

Chi’s testing model remains simple, explicit, and close to production behavior.

## Gin

```go
package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGinHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong")
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Body.String() != "pong" {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}
```

Gin introduces its own context type and internal lifecycle management. As a result, handlers cannot be called directly as plain functions. All meaningful tests must go through the Gin engine and router by invoking `ServeHTTP`.

Gin provides a test mode that disables logging and recovery behavior to make assertions predictable. Middleware is tested by registering it on the engine and observing its effects on the response. Because Gin maintains internal state about the request lifecycle, some behaviors only appear when the request flows through the full router pipeline.

Testing Gin is still straightforward, but it is less granular than `net/http` or Chi. The framework encourages testing the system as a whole rather than individual handlers in isolation.

## Echo

```go
package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestEchoHandler(t *testing.T) {
	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := func(c echo.Context) error {
		return c.String(200, "pong")
	}(c); err != nil {
		t.Fatal(err)
	}

	if rec.Body.String() != "pong" {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}
```

Echo offers two testing styles. Handlers can be tested directly by manually constructing a context, which allows logic to be exercised without routing or middleware. This enables fast, isolated tests that focus on handler behavior alone.

Routing tests still go through `ServeHTTP`, similar to other frameworks. Middleware can be tested either by wrapping handlers directly or by attaching it to the Echo instance and asserting on responses. Error handling can be tested by returning errors from handlers and observing how the global error handler responds.

This dual model provides flexibility. Developers can choose between isolated unit tests and full pipeline tests depending on what they want to validate.

## Fiber

```go
package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestFiberHandler(t *testing.T) {
	app := fiber.New()

	app.Get("/ping", func(c *fiber.Ctx) error {
		return c.SendString("pong")
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	body, _ := io.ReadAll(res.Body)
	if string(body) != "pong" {
		t.Fatalf("unexpected body: %s", body)
	}
}
```

Fiber provides a dedicated testing helper that executes requests against the application without binding to a network port. This allows routing, middleware, and handlers to be tested end to end while remaining fast and isolated.

Handlers themselves cannot be invoked directly. All tests flow through the router and middleware stack. Responses are fully buffered, which simplifies assertions but also means the execution model differs slightly from streaming behavior in production.

Because Fiber reuses context objects internally for performance, tests must avoid retaining references across requests. This is an important consideration when writing table driven or parallel tests.

## Mizu

```go
package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestMizuHandler(t *testing.T) {
	app := mizu.New()

	app.Get("/ping", func(c *mizu.Ctx) error {
		return c.Text(200, "pong")
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Body.String() != "pong" {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}
```

Mizu intentionally supports both testing styles. Requests can be sent through `ServeHTTP` just like `net/http`, which exercises routing, middleware, error handling, and panic recovery exactly as in production. This makes integration style tests straightforward and realistic.

At the same time, handlers can be tested in isolation by constructing a context directly when finer control is needed. Middleware behavior, error propagation, and panic handling remain deterministic because the request pipeline is explicit and consistent.

This approach allows tests to stay close to production behavior while still supporting fast, focused unit tests where appropriate.

## What learners should focus on

Testing reveals what a framework optimizes for. Some frameworks emphasize composability and isolation, making unit tests trivial. Others emphasize full pipeline correctness, encouraging integration style tests. The key questions are whether handlers can be invoked directly, whether routing is required for every test, how much hidden state exists, and how errors and panics surface in assertions.

Frameworks that preserve the `net/http` contract tend to offer the most flexibility. Frameworks that introduce custom lifecycles often trade isolation for convenience. Understanding these tradeoffs is essential when choosing a framework and when designing a testing strategy that scales with the system.
