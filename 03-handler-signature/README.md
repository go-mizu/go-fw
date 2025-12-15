# Handler signatures and request lifetime

This section explains what a handler actually represents in each framework and how a single HTTP request moves through your program from arrival to completion.

At first glance, handlers across frameworks appear similar. They all respond to requests and write responses. The important differences appear when you look at *where data lives*, *who owns the response*, *how errors are expressed*, and *how long objects remain valid*. These details shape how middleware works, how code composes, how testing feels, and how easy it is to reason about failures.

A handler signature is a contract. It defines which objects are created by the server, which are owned by the framework, which are safe to keep, and which must never escape the request. Once this contract is understood, most higher-level framework behavior becomes predictable.

Each subsection below shows the full runnable program and then walks through the request lifetime using concrete mechanics, small snippets, and focused comparisons.

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
	mux.HandleFunc("GET /", handler)
	http.ListenAndServe(":8080", mux)
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "hello, world!")
}
```

In net/http, the handler is a direct function call from the server into user code. After the TCP connection is accepted and the request is parsed, the server allocates two objects and passes them to the handler: a `*http.Request` and a `ResponseWriter`. No additional abstraction is involved.

The request object represents parsed input. Its fields are populated before the handler runs. The body is a stream tied to the underlying connection. Reading from `r.Body` affects connection reuse. Leaving unread data can prevent keep-alive. Draining or closing the body early changes server behavior.

The response writer represents a stateful output stream. It tracks whether headers have been committed and writes bytes back to the client. The first call to `Write` implicitly commits headers if `WriteHeader` was not called explicitly.

```go
w.WriteHeader(200)
w.Write([]byte("hello"))
```

or simply:

```go
fmt.Fprintln(w, "hello")
```

There is no return value. The handler communicates intent only through side effects. Errors are expressed by writing an error response, returning without writing, or panicking.

The lifetime model is strict and simple. The server calls the handler and expects all request-scoped work to finish before the function returns. Any goroutine that outlives the handler must copy everything it needs.

Key properties:

| Aspect             | net/http            |
| ------------------ | ------------------- |
| Handler type       | func(w, r)          |
| Response ownership | direct              |
| Error signaling    | side effects, panic |
| Object reuse       | none                |

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
	r.Get("/", handler)
	http.ListenAndServe(":8080", r)
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "hello, world!")
}
```

Chi keeps the same handler contract as net/http and moves framework logic into routing and middleware composition. The server still calls `ServeHTTP` with a `ResponseWriter` and a `*http.Request`.

Before invoking the handler, Chi matches the request path and method against its routing tree. Route parameters and metadata are attached to the request context.

```go
id := chi.URLParam(r, "id")
```

From the handler’s perspective, nothing else changes. Response writing follows the same commit rules. Errors propagate in the same way.

The lifetime of objects remains identical to net/http. Chi does not pool or reuse handler-level objects. Middleware wraps handlers using standard function composition.

Key properties:

| Aspect             | Chi                 |
| ------------------ | ------------------- |
| Handler type       | func(w, r)          |
| Response ownership | direct              |
| Error signaling    | side effects, panic |
| Request extension  | via context         |
| Object reuse       | none                |

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
	r.GET("/", handler)
	r.Run(":8080")
}

func handler(c *gin.Context) {
	c.String(http.StatusOK, "hello, world!")
}
```

Gin replaces the handler contract with a framework-managed context. When a request arrives, Gin allocates or reuses a `gin.Context` from a pool and binds it to the current request and response writer.

The context contains both input and output state:

```go
c.Request        // *http.Request
c.Writer         // wrapped ResponseWriter
c.Params         // route params
```

Handlers do not write to the response directly. Instead, they call helper methods that mutate the context and forward writes through a wrapped writer.

```go
c.JSON(200, obj)
c.String(200, "ok")
```

Control flow is managed by Gin. Middleware and handlers execute in a linear chain controlled by an internal index. Early exits occur by mutating the context rather than by returning values.

Context pooling defines the lifetime rule. After the request completes, the context is reset and returned to the pool. Any reference to the context after the handler returns is unsafe.

Key properties:

| Aspect             | Gin                  |
| ------------------ | -------------------- |
| Handler type       | func(*Context)       |
| Response ownership | via context          |
| Error signaling    | context state, panic |
| Object reuse       | pooled context       |

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
	e.GET("/", handler)
	e.Start(":8080")
}

func handler(c echo.Context) error {
	return c.String(http.StatusOK, "hello, world!")
}
```

Echo introduces a return value to express control flow. The handler returns an error that represents success or failure.

The context wraps request and response objects and exposes helper methods. Writing output still happens through helpers, but the outcome is explicit.

```go
return c.JSON(200, data)
```

or:

```go
return echo.NewHTTPError(400, "bad request")
```

After the handler returns, Echo inspects the error. A non-nil error triggers centralized error handling, which decides how to write the response.

The context is request-scoped and not reused in unsafe ways. Execution paths are easier to follow because success and failure are expressed as values rather than inferred.

Key properties:

| Aspect             | Echo                |
| ------------------ | ------------------- |
| Handler type       | func(Context) error |
| Response ownership | via context         |
| Error signaling    | return value        |
| Object reuse       | request-scoped      |

## Fiber

[`fiber/main.go`](fiber/main.go)

```go
package main

import (
	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()
	app.Get("/", handler)
	app.Listen(":8080")
}

func handler(c *fiber.Ctx) error {
	return c.SendString("hello, world!")
}
```

Fiber uses a similar signature to Echo but operates on top of fasthttp rather than net/http. Requests and responses are represented by fasthttp structures, and the context holds pointers into those structures.

When a request arrives, Fiber retrieves a context from a pool, routes the request, invokes the handler, then resets the context.

```go
body := c.Body()   // byte slice reused later
```

Buffers and slices returned from the context are reused aggressively. Keeping references beyond the handler can lead to corrupted data.

Error returns feed Fiber’s error handling pipeline. Response writing occurs by mutating the fasthttp response and letting the server flush it after the handler returns.

Key properties:

| Aspect             | Fiber                      |
| ------------------ | -------------------------- |
| Handler type       | func(*Ctx) error           |
| Response ownership | via context                |
| Error signaling    | return value               |
| Object reuse       | pooled context and buffers |

## Mizu

[`mizu/main.go`](mizu/main.go)

```go
package main

import (
	"github.com/go-mizu/mizu"
)

func main() {
	app := mizu.New()
	app.Get("/", handler)
	app.Listen(":8080")
}

func handler(c *mizu.Ctx) error {
	return c.Text(200, "hello, world!")
}
```

Mizu combines explicit error returns with net/http compatibility. The context wraps `http.ResponseWriter` and `*http.Request` and carries routing and middleware state.

Handlers write responses through helper methods and return an error to signal failure. Centralized error handling converts errors into responses consistently.

```go
return c.JSON(200, data)
```

or:

```go
return mizu.ErrBadRequest
```

The request lifetime stays aligned with net/http. Parsing, cancellation, and response commit semantics come from the standard library. The context is request-scoped and designed to remain safe as long as handlers respect request boundaries.

Key properties:

| Aspect             | Mizu             |
| ------------------ | ---------------- |
| Handler type       | func(*Ctx) error |
| Response ownership | via context      |
| Error signaling    | return value     |
| Object reuse       | controlled       |

## Summary

| Framework | Handler type        | Error channel       | Response ownership | Reuse model    |
| --------- | ------------------- | ------------------- | ------------------ | -------------- |
| net/http  | func(w, r)          | side effects, panic | direct             | none           |
| Chi       | func(w, r)          | side effects, panic | direct             | none           |
| Gin       | func(*Context)      | context state       | indirect           | pooled         |
| Echo      | func(Context) error | return value        | indirect           | request-scoped |
| Fiber     | func(*Ctx) error    | return value        | indirect           | pooled         |
| Mizu      | func(*Ctx) error    | return value        | indirect           | controlled     |

Once the handler contract and lifetime rules are clear, the behavior of middleware, testing patterns, cancellation, and shutdown becomes much easier to reason about across frameworks.
