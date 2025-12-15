# Hello, world

We implement a minimal HTTP server that listens on port 8080 and responds to `GET /` with plain text `hello, world!`.

Each example shows the full runnable code and then explains what actually happens inside the framework when a request is received. The focus is on ownership, control flow, and how much machinery exists between the socket and your handler.

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

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "hello, world!")
	})

	http.ListenAndServe(":8080", mux)
}
```

When this program starts, `http.ListenAndServe` creates an `http.Server` and binds a TCP listener to port 8080. From this point on, the runtime enters an accept loop where each incoming connection is handled concurrently. The server is responsible for reading raw bytes from the socket, parsing the HTTP request line, headers, and body, and constructing a `*http.Request` value that represents the parsed input.

The `ServeMux` passed to `ListenAndServe` becomes the root `http.Handler`. Once a request has been parsed, the server invokes `ServeHTTP` on the mux. Internally, the mux matches the request method and path against its routing structure and selects the most specific handler. In recent Go versions this is not a naive linear scan, but the important point is that routing happens before any user code runs.

The handler function receives the original `ResponseWriter` and `*http.Request`. There is no intermediate abstraction. Writing to the `ResponseWriter` writes directly to the response stream associated with the connection. The first write implicitly commits the response headers and status code. Once the handler returns, the server finalizes the response and the connection may be reused or closed depending on headers and protocol state.

There is no centralized error handling, no middleware chain, and no shared request context beyond `context.Context` on the request itself. Control flow is explicit and local, and the handler fully owns the response lifecycle.

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

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "hello, world!")
	})

	http.ListenAndServe(":8080", r)
}
```

Chi inserts itself between net/http and your handler only at the routing layer. The HTTP server still performs all parsing, connection management, and request construction exactly as it does with a plain `ServeMux`. The difference begins when the server calls `ServeHTTP` on the Chi router instead of the standard mux.

Inside Chi, the request path is decomposed into segments and matched against a tree structure built from registered routes. This tree allows Chi to efficiently resolve static paths, parameters, and wildcards. When a match is found, any extracted parameters are stored in the request’s context rather than passed as function arguments.

After routing, Chi invokes the handler using the standard `func(http.ResponseWriter, *http.Request)` signature. Response writing is unchanged. The handler still writes directly to the underlying connection through the `ResponseWriter`. Chi does not buffer output or delay writes. It only decides which handler runs and which middleware wraps it.

The result is a router that adds structure and composition without altering the fundamental ownership model of net/http.

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

	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "hello, world!")
	})

	r.Run(":8080")
}
```

Gin changes the execution model by replacing the handler signature and centralizing control in a framework-managed context object. When `r.Run` is called, Gin creates an `http.Server` and installs its engine as the root handler. From net/http’s perspective, Gin is just another `http.Handler`.

When a request arrives, Gin’s `ServeHTTP` method is invoked. At this point Gin allocates or reuses a `gin.Context` from an internal pool. This context wraps the original `ResponseWriter` and `*http.Request` and also carries routing metadata, middleware state, and response status tracking.

Routing occurs inside Gin’s own tree structure, and the resulting handler chain is executed using the shared context. Middleware and handlers all mutate the same context instance. Helper methods such as `c.String` write through Gin’s wrapped `ResponseWriter`, allowing Gin to observe status codes and headers as they are set.

Unlike net/http, the handler no longer owns the response directly. Control flow is inverted. Gin owns the request lifecycle, and user code operates inside it by mutating the context.

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

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "hello, world!")
	})

	e.Start(":8080")
}
```

Echo is similar to Gin in that it introduces a framework context and routing layer, but it differs in one key design choice: handlers return an `error`. This makes error propagation explicit rather than implicit.

When Echo starts, it creates an `http.Server` and registers its internal handler. For each request, Echo creates a context that wraps the `ResponseWriter` and `*http.Request`. The handler is invoked and writes to the response through helper methods on the context.

After the handler returns, Echo inspects the returned error. If it is non-nil, Echo routes execution through a centralized error handler, which decides how to translate the error into an HTTP response. This means error handling is part of the normal control flow rather than an exceptional path.

Response writing still occurs through net/http, but the decision of whether a response represents success or failure is centralized rather than scattered across handlers.

## Fiber

[`fiber/main.go`](fiber/main.go)

```go
package main

import (
	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("hello, world!")
	})

	app.Listen(":8080")
}
```

Fiber departs entirely from net/http and is built on top of fasthttp. This changes the execution model at a much lower level. Instead of net/http parsing requests and managing connections, fasthttp handles raw socket IO and uses its own request and response types.

When a connection arrives, fasthttp parses the request into its own structures. Fiber then wraps these structures in a `*fiber.Ctx`, which is passed to the handler. The handler mutates the context to set status and body and returns an error.

Contexts are aggressively pooled and reused. This makes allocation cheaper but imposes strict lifetime rules. A context must never be stored or referenced outside the handler, because it will be reused for another request.

Because Fiber does not use net/http types, it does not interoperate directly with net/http middleware or handlers. The ecosystem boundary is hard, not conceptual.

## Mizu

[`mizu/main.go`](mizu/main.go)

```go
package main

import (
	"github.com/go-mizu/mizu"
)

func main() {
	app := mizu.New()

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(200, "hello, world!")
	})

	app.Listen(":8080")
}
```

Mizu stays within the net/http execution model while introducing structured error handling and a consistent context API. When `app.Listen` is called, Mizu creates an `http.Server` and registers itself as the root handler. net/http continues to handle connections, parsing, and request construction.

When a request is received, Mizu’s router matches the method and path and allocates or reuses a `*mizu.Ctx`. This context holds the original `ResponseWriter` and `*http.Request`, along with route metadata and middleware state.

Handlers write responses through helper methods on the context, which forward to the underlying `ResponseWriter`. The handler returns an error value. If the error is non-nil, Mizu routes execution through centralized error handling logic that decides how to produce a response.

The key distinction is that Mizu adds structure without leaving the net/http ecosystem. Handlers still operate on real net/http primitives, but control flow is more explicit and uniform.

## Direct technical comparison

| Framework | Handler signature          | Response writing | Error return | Transport |
| --------- | -------------------------- | ---------------- | ------------ | --------- |
| net/http  | `func(w, r)`               | direct           | no           | net/http  |
| Chi       | `func(w, r)`               | direct           | no           | net/http  |
| Gin       | `func(*gin.Context)`       | via context      | no           | net/http  |
| Echo      | `func(echo.Context) error` | via context      | yes          | net/http  |
| Fiber     | `func(*fiber.Ctx) error`   | via context      | yes          | fasthttp  |
| Mizu      | `func(*mizu.Ctx) error`    | via context      | yes          | net/http  |

Understanding these internal differences early makes later topics such as middleware order, cancellation, and graceful shutdown much easier to reason about.
