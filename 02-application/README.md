# Application wiring

This section grows the program slightly without changing what the HTTP endpoint does.

The request handling still responds with `hello, world!`, but the structure stops being accidental. Wiring becomes explicit: a program decides where objects are created, who holds references to them, and which layer owns each responsibility. That decision sets the shape of everything that follows, including configuration, testing, graceful shutdown, observability, and integration with other services.

Wiring answers questions every service eventually faces:

* who creates the HTTP server
* who owns routing
* where routes are registered
* what object handles incoming requests

These differences are about ownership and control. They influence how easily a service can be embedded, how cleanly it can be tested without sockets, and how confidently timeouts and lifecycle can be enforced.

Each subsection links to runnable code and explains how responsibility flows at runtime.

## net/http

[`nethttp/main.go`](nethttp/main.go)

```go
package main

import (
	"fmt"
	"net/http"
)

func main() {
	mux := newRouter()
	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	srv.ListenAndServe()
}

func newRouter() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "hello, world!")
	})

	return mux
}
```

This layout draws a sharp line between process lifecycle and request behavior.

`main` owns lifecycle decisions. It chooses the address, constructs `http.Server`, and sets the server’s `Handler`. The server becomes the runtime entry point for every request. That `Handler` field is the only dependency the server needs to do its job: accept connections, parse requests, and dispatch.

`newRouter` owns HTTP behavior. It constructs a router, registers routes, and returns an `http.Handler`. Returning an interface is important. It allows the server layer to depend on a contract rather than a concrete implementation. A different router can be substituted without changing how the server is created.

At runtime, the boundary is a single call: the server invokes `ServeHTTP(w, r)` on the handler it holds. Everything beyond that is a handler graph owned by the application. That makes the control flow easy to reason about.

Key ownership boundaries:

| Concern              | Owner             |
| -------------------- | ----------------- |
| sockets, accept loop | `http.Server`     |
| request parsing      | `net/http`        |
| routing decision     | `ServeMux`        |
| response writing     | handler functions |

This approach keeps configuration flexible. Timeouts, TLS, listener type, and shutdown orchestration live next to the server because that layer owns them.

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
	r := newRouter()
	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	srv.ListenAndServe()
}

func newRouter() http.Handler {
	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "hello, world!")
	})

	return r
}
```

The wiring remains the same shape as net/http: `main` creates the server, the router is injected via `Handler`, and the server calls `ServeHTTP`.

The significant detail is that the Chi router satisfies `http.Handler`, so the server stays unaware of Chi. That preserves a stable integration surface. Anything that expects an `http.Handler` can wrap or host this router: reverse proxies, instrumenting middleware, test servers, and other routers.

At runtime, the call graph still begins in net/http and crosses into the application at `ServeHTTP`. Chi performs route matching inside that call and then invokes the registered handler.

Ownership stays clean:

| Concern                | Owner                      |
| ---------------------- | -------------------------- |
| server lifecycle       | `main` via `http.Server`   |
| routing implementation | Chi router                 |
| handler API            | net/http handler signature |

This makes it easy to swap routing implementations without changing how the service binds ports or configures timeouts.

## Gin

[`gin/main.go`](gin/main.go)

```go
package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := newRouter()
	r.Run(":8080")
}

func newRouter() *gin.Engine {
	r := gin.New()

	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "hello, world!")
	})

	return r
}
```

Gin shifts where lifecycle responsibilities live. The application creates a `*gin.Engine`, and `Run` takes over server startup. That changes the ownership boundary: the framework owns the act of constructing and running the server, and user code interacts with that boundary through Gin’s APIs.

This wiring trades explicit server construction for a single call. The router and the server lifecycle are tightly coupled by default.

Two consequences follow from that coupling:

* server configuration is driven through framework knobs or alternative startup paths, rather than by directly constructing `http.Server` in `main`
* the entry point for requests moves behind `Run`, which installs the Gin engine as the server handler and starts listening

At runtime, net/http still performs connection handling and parsing, then calls the Gin engine’s `ServeHTTP`. The difference is where that server is created and configured.

Ownership boundary shifts:

| Concern              | Owner                     |
| -------------------- | ------------------------- |
| server creation      | Gin via `Run`             |
| server configuration | Gin-first unless bypassed |
| routing + middleware | Gin engine                |
| request pipeline     | Gin context chain         |

This model reduces boilerplate and standardizes structure, while moving more control into framework conventions.

## Echo

[`echo/main.go`](echo/main.go)

```go
package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func main() {
	e := newApp()
	e.Start(":8080")
}

func newApp() *echo.Echo {
	e := echo.New()

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "hello, world!")
	})

	return e
}
```

Echo follows a similar shape to Gin: an application object owns routing and middleware, and a method on that object starts the server. The wiring presents the framework object as the primary unit of composition.

The server boundary is created by `Start`. The user does not construct `http.Server` directly in this shape. That pushes lifecycle configuration closer to framework configuration, with the application object becoming the place where request behavior and cross-cutting concerns accumulate.

At runtime, Echo installs its handler into the server and dispatches through its internal router and middleware layers. The handler signature returns `error`, which becomes important later because failures can bubble into centralized handling rather than being written directly in each handler.

Ownership boundary:

| Concern        | Owner                              |
| -------------- | ---------------------------------- |
| server startup | Echo via `Start`                   |
| routing        | Echo router                        |
| error path     | central dispatcher + error handler |

This tends to feel cohesive as applications grow because the framework object becomes the single place where policy and behavior are assembled.

## Fiber

[`fiber/main.go`](fiber/main.go)

```go
package main

import (
	"github.com/gofiber/fiber/v2"
)

func main() {
	app := newApp()
	app.Listen(":8080")
}

func newApp() *fiber.App {
	app := fiber.New()

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("hello, world!")
	})

	return app
}
```

Fiber owns the runtime end-to-end. The service does not present an `http.Handler` boundary and does not use `http.Server` because Fiber uses a fasthttp-based stack.

Wiring is compact because the framework object covers routing, request parsing, and the listen loop. Calling `Listen` enters Fiber’s server loop.

That changes the integration surface:

* middleware and tooling designed for net/http do not plug in directly
* embedding Fiber inside a larger net/http server requires adapters
* the request and response types come from Fiber’s ecosystem

At runtime, request parsing and dispatch happen entirely inside Fiber’s engine. The program hands control to Fiber, and Fiber invokes handlers via its own context type.

Ownership boundary:

| Concern               | Owner          |
| --------------------- | -------------- |
| accept loop + parsing | Fiber/fasthttp |
| routing               | Fiber          |
| handler API           | Fiber context  |

This model tends to prioritize a cohesive internal pipeline over interoperability.

## Mizu

[`mizu/main.go`](mizu/main.go)

```go
package main

import (
	"github.com/go-mizu/mizu"
)

func main() {
	app := newApp()
	app.Listen(":8080")
}

func newApp() *mizu.App {
	app := mizu.New()

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(200, "hello, world!")
	})

	return app
}
```

Mizu presents an application object as the place where routing and middleware are assembled, then exposes a `Listen` method to start serving. Internally, the serving model aligns with net/http, which preserves the standard library’s execution model while offering a framework-style surface.

The wiring separates concerns in practice:

* the app object gathers routes and middleware
* listening creates and runs the server using net/http semantics

That means the entry point remains compatible with net/http style request execution, and the service can interact with net/http concepts like handlers, servers, and middleware wrappers in a predictable way.

Ownership boundary:

| Concern                           | Owner                                          |
| --------------------------------- | ---------------------------------------------- |
| routing + middleware registration | Mizu app object                                |
| connection handling               | net/http server underneath                     |
| error flow                        | handler error return into centralized handling |

This tends to keep integration options open while still giving a structured application object to build on.

## What to learn from this section

Wiring choices determine where control lives.

* net/http and Chi keep a visible `http.Server` boundary in user code, which makes lifecycle configuration straightforward and explicit
* Gin and Echo make the framework object the center of gravity, often starting the server through a framework call
* Fiber takes full ownership by leaving the net/http ecosystem
* Mizu uses an app object while keeping a net/http-aligned execution model underneath

These choices affect:

* configuring timeouts, TLS, and custom listeners
* testing routing and middleware without binding ports
* embedding one service inside another
* attaching ecosystem middleware and instrumentation
