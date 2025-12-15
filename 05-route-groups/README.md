# Route groups and composition

Route groups help avoid repetition once an API grows beyond a few endpoints. They solve two practical problems: attaching a shared path prefix like `/api/v1`, and attaching shared behavior like authentication for `/admin`. The interesting part sits under the surface. Some stacks implement groups as runtime composition, others treat them as route-registration helpers that precompute final paths and handler chains, and net/http pushes you toward building the mechanism yourself by composing handlers.

This section stays focused on two things only:

* grouping by path prefix, like `/api/v1`
* composing shared behavior, like authentication for `/admin`

## net/http

[`nethttp/main.go`](nethttp/main.go)

```go
package main

import (
	"fmt"
	"net/http"
)

func main() {
	root := http.NewServeMux()

	root.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "root")
	})

	apiV1 := http.NewServeMux()
	apiV1.HandleFunc("GET /users", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "api v1 users")
	})
	apiV1.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "api v1 user:", r.PathValue("id"))
	})

	admin := http.NewServeMux()
	admin.HandleFunc("GET /dashboard", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "admin dashboard")
	})

	root.Handle("/api/v1/", http.StripPrefix("/api/v1", apiV1))
	root.Handle("/admin/", chain(http.StripPrefix("/admin", admin), requireToken("letmein")))

	http.ListenAndServe(":8080", root)
}

type Middleware func(http.Handler) http.Handler

func chain(h http.Handler, m ...Middleware) http.Handler {
	for i := len(m) - 1; i >= 0; i-- {
		h = m[i](h)
	}
	return h
}

func requireToken(token string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Admin-Token") != token {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte("unauthorized"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
```

net/http provides routing and handler composition, but no explicit group abstraction. Grouping emerges by mounting sub-routers under a prefix. The root mux performs the outer match, then delegates to a sub-mux. `http.StripPrefix` performs a rewrite so that the mounted mux can match routes as if it lived at `/`.

The effective behavior can be described as a two-step match:

| Client path        | Root match | Path seen by sub-mux |
| ------------------ | ---------- | -------------------- |
| `/api/v1/users`    | `/api/v1/` | `/users`             |
| `/api/v1/users/42` | `/api/v1/` | `/users/42`          |
| `/admin/dashboard` | `/admin/`  | `/dashboard`         |

Middleware composition happens by wrapping handlers. The admin group shows the standard net/http pattern: the group is the handler tree under `/admin/`, and shared behavior is wrapped around that handler tree. The middleware decides whether to call `next.ServeHTTP`.

```go
root.Handle("/admin/",
	chain(
		http.StripPrefix("/admin", admin),
		requireToken("letmein"),
	),
)
```

The main sharp edge is that the sub-mux sees a rewritten path. Any code that logs `r.URL.Path`, constructs redirects, or performs path-based authorization inside the sub-mux operates on the stripped path. If you need the original path, you must preserve it explicitly.

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
		fmt.Fprintln(w, "root")
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/users", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "api v1 users")
		})
		r.Get("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "api v1 user:", chi.URLParam(r, "id"))
		})
	})

	r.Route("/admin", func(r chi.Router) {
		r.Use(requireToken("letmein"))
		r.Get("/dashboard", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "admin dashboard")
		})
	})

	http.ListenAndServe(":8080", r)
}

func requireToken(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Admin-Token") != token {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte("unauthorized"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
```

Chi’s `Route` provides a scoped router view. It applies a prefix and allows middleware to be attached within that scope. Unlike the net/http approach, no path rewriting is required. The incoming request path remains unchanged, and the router maintains internal state about the current prefix while walking the routing tree.

That distinction matters when building redirects and logs. Inside handlers, `r.URL.Path` remains the client path. Parameter extraction also stays consistent because Chi attaches route state to the request context rather than rewriting the URL.

Middleware inheritance composes predictably. Middleware registered outside the group wraps everything inside. Middleware registered inside the group wraps only the group routes. The resulting execution order follows the router scope structure rather than registration tricks.

A compact mental model:

| Concept      | Chi group behavior              |
| ------------ | ------------------------------- |
| Prefix       | applied logically in the router |
| Middleware   | stacked by scope                |
| Request path | preserved                       |
| Composition  | runtime wrapper chain           |

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
		c.String(http.StatusOK, "root")
	})

	api := r.Group("/api/v1")
	{
		api.GET("/users", func(c *gin.Context) {
			c.String(http.StatusOK, "api v1 users")
		})
		api.GET("/users/:id", func(c *gin.Context) {
			c.String(http.StatusOK, "api v1 user: %s", c.Param("id"))
		})
	}

	admin := r.Group("/admin", requireToken("letmein"))
	{
		admin.GET("/dashboard", func(c *gin.Context) {
			c.String(http.StatusOK, "admin dashboard")
		})
	}

	r.Run(":8080")
}

func requireToken(token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("X-Admin-Token") != token {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}
```

Gin groups are `RouterGroup` objects carrying a base path and a middleware list. Route registration through a group combines these pieces into a final route entry. The important technical detail is that the prefix and middleware chain are mostly resolved at registration time rather than at request time.

You can think of Gin’s group as a route builder:

* prefix is concatenated into the final path matcher
* middleware slices are concatenated into the final handler chain

A useful way to visualize what gets stored:

| Registration call            | Effective route    | Effective handlers         |
| ---------------------------- | ------------------ | -------------------------- |
| `api.GET("/users", H)`       | `/api/v1/users`    | `global... + api... + H`   |
| `admin.GET("/dashboard", H)` | `/admin/dashboard` | `global... + admin... + H` |

At runtime, dispatch selects a route and runs a precomputed handler chain. Control flow is driven by the framework context. Middleware uses `c.Next()` to continue and `Abort...` to stop. That changes how you reason about early-exit compared to net/http wrappers, since the “call next” decision happens through framework control flow instead of direct function calls.

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
		return c.String(http.StatusOK, "root")
	})

	api := e.Group("/api/v1")
	api.GET("/users", func(c echo.Context) error {
		return c.String(http.StatusOK, "api v1 users")
	})
	api.GET("/users/:id", func(c echo.Context) error {
		return c.String(http.StatusOK, "api v1 user: "+c.Param("id"))
	})

	admin := e.Group("/admin", requireToken("letmein"))
	admin.GET("/dashboard", func(c echo.Context) error {
		return c.String(http.StatusOK, "admin dashboard")
	})

	e.Start(":8080")
}

func requireToken(token string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Request().Header.Get("X-Admin-Token") != token {
				return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
			}
			return next(c)
		}
	}
}
```

Echo groups combine a prefix with a middleware slice and produce a scoped router. The request-time dispatcher resolves the route and then executes middleware in hierarchical order. Middleware composition uses function wrapping where both middleware and handler return an error. That makes early exit precise: returning an error stops the chain and hands control to the centralized error handler.

This leads to a clear group behavior:

* prefix selects a route subset
* middleware is scoped to that subset
* failure propagates as an error value

A small contrast with Gin helps here:

| Stack | “Stop execution” looks like              |
| ----- | ---------------------------------------- |
| Gin   | `c.Abort...` inside context-driven chain |
| Echo  | `return err` inside wrapper chain        |

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
		return c.SendString("root")
	})

	api := app.Group("/api/v1")
	api.Get("/users", func(c *fiber.Ctx) error {
		return c.SendString("api v1 users")
	})
	api.Get("/users/:id", func(c *fiber.Ctx) error {
		return c.SendString("api v1 user: " + c.Params("id"))
	})

	admin := app.Group("/admin", requireToken("letmein"))
	admin.Get("/dashboard", func(c *fiber.Ctx) error {
		return c.SendString("admin dashboard")
	})

	app.Listen(":8080")
}

func requireToken(token string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Get("X-Admin-Token") != token {
			return c.Status(401).SendString("unauthorized")
		}
		return c.Next()
	}
}
```

Fiber groups work as prefix-based registration helpers and middleware scoping tools. Registration through a group combines prefix and route path into a final matcher, and group middleware is attached to the routes registered under that group.

Middleware flow uses `c.Next()` to continue. The underlying context is pooled and fasthttp-backed, so group middleware must treat all request-derived values as request-bound. Anything stored beyond handler execution must be copied.

A simplified view of what gets produced:

| Registration                       | Effective route | Group behavior          |
| ---------------------------------- | --------------- | ----------------------- |
| `api.Get("/users", ...)`           | `/api/v1/users` | prefix concatenation    |
| `admin := app.Group("/admin", mw)` | `/admin/...`    | group-scoped middleware |

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

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "root")
	})

	api := app.Group("/api/v1")
	api.Get("/users", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "api v1 users")
	})
	api.Get("/users/:id", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "api v1 user: "+c.Param("id"))
	})

	admin := app.Group("/admin")
	admin.Use(requireToken("letmein"))
	admin.Get("/dashboard", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "admin dashboard")
	})

	app.Listen(":8080")
}

func requireToken(token string) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			if c.Request().Header.Get("X-Admin-Token") != token {
				return c.Text(http.StatusUnauthorized, "unauthorized")
			}
			return next(c)
		}
	}
}
```

Mizu groups represent a router scope with an inherited middleware chain. The group carries a prefix and middleware that applies to routes registered on that group. Middleware composition uses a pure wrapping model where a middleware receives `next` and decides whether to call it, returning an error either way.

The technical payoff is a stable mental model: group composition behaves like structured handler wrapping, and the request path remains the real path rather than a rewritten one. That avoids the common StripPrefix confusion where internal handlers observe a different URL path from clients.

A quick comparison table helps anchor the implementation differences:

| Framework | Group implemented as      | Prefix handling              | Middleware shape             |
| --------- | ------------------------- | ---------------------------- | ---------------------------- |
| net/http  | mounted sub-mux           | rewrite via StripPrefix      | `func(next) handler`         |
| Chi       | scoped router view        | internal prefix offset       | `func(next) handler`         |
| Gin       | route builder             | concatenated at registration | context chain, `Next/Abort`  |
| Echo      | scoped group + dispatcher | resolved by router           | wrapper chain, returns error |
| Fiber     | route builder             | concatenated at registration | context chain, `Next`        |
| Mizu      | scoped router view        | internal prefix + tree       | wrapper chain, returns error |

## What learners should focus on

Groups look similar at the surface: a prefix and optional middleware. The deeper difference lies in where the composition happens and what the handler inside the group sees.

* net/http grouping mounts a handler under a prefix and often rewrites the path, which can affect redirects and logs inside the mounted handler
* Chi and Mizu keep the original path and implement groups as scoped router views with predictable middleware inheritance
* Gin and Fiber flatten prefixes and middleware at registration time and drive middleware flow through framework-controlled context progression
* Echo composes groups naturally with error returns, letting group middleware stop execution by returning an error and relying on centralized error handling
