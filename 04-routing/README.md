# Routing: paths, methods, and precedence

Routing determines how an incoming HTTP request is translated into a specific piece of code. By the time routing runs, the server has already accepted a connection, parsed the request line and headers, and constructed the request object. Routing sits between request parsing and handler execution and decides what code will run next, if any.

This decision sounds simple, but it carries several consequences. The routing model defines how paths are compared, where HTTP methods are checked, how conflicts are resolved when multiple routes could apply, and whether redirects or rewrites happen automatically. It also determines how much work happens before your handler is reached and how predictable the matching rules are as a codebase grows.

At this stage, handler signatures stay fixed. The focus shifts entirely to how frameworks organize routing logic and how that logic behaves under overlap, ambiguity, and scale.

Each section below links to the runnable code, shows the full `main.go`, and then explains how routing behaves internally with concrete mechanics and examples.

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

	mux.HandleFunc("GET /", root)
	mux.HandleFunc("GET /users", users)
	mux.HandleFunc("GET /users/", userSubtree)

	http.ListenAndServe(":8080", mux)
}

func root(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "root")
}

func users(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "users")
}

func userSubtree(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "users subtree:", r.URL.Path)
}
```

Routing in `http.ServeMux` is based on pattern rules rather than an explicit routing tree. Each registered pattern is a string that participates in precedence comparison. When a request arrives, the mux selects the best match according to a small set of ordering rules that determine which handler wins.

Two properties dominate precedence. Exact matches outrank subtree matches, and longer patterns outrank shorter ones. A pattern ending with a slash represents a subtree and matches any path that shares the prefix.

In the example above, three patterns coexist:

```text
GET /users        exact
GET /users/       subtree
GET /             subtree
```

A request to `/users` selects the exact match. A request to `/users/42` selects the subtree `/users/`. A request to `/anything-else` falls back to `/`.

Method matching is part of the pattern string. The mux checks the HTTP method before calling the handler. Redirect behavior for trailing slashes is handled internally by the mux and occurs before handler execution.

There is no parameter extraction step. The handler receives the raw path and must parse it manually if needed. Routing remains simple and predictable, but expressive power stays limited.

Key routing characteristics:

| Aspect         | net/http            |
| -------------- | ------------------- |
| Matching model | pattern rules       |
| Method check   | built into mux      |
| Precedence     | exact, then longest |
| Parameters     | none                |
| Redirects      | implicit            |

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

	r.Get("/", root)
	r.Get("/users", users)
	r.Get("/users/{id}", userByID)

	http.ListenAndServe(":8080", r)
}

func root(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "root")
}

func users(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "users")
}

func userByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	fmt.Fprintln(w, "user:", id)
}
```

Chi organizes routes into a radix-style tree per HTTP method. Each path is split into segments, and each segment becomes a node. Static segments and parameter segments occupy different positions in the tree, which allows Chi to enforce clear precedence rules.

When a request arrives, Chi selects the tree for the request method and walks it segment by segment. Static segments are matched first, followed by parameter segments. When a parameter segment matches, the value is captured and stored in the request context.

```go
id := chi.URLParam(r, "id")
```

Route resolution completes before the handler is called. The router determines the handler and enriches the request context with routing metadata. Handler execution then proceeds using the standard net/http contract.

Redirects are not automatic. Trailing slash behavior must be configured explicitly, which keeps routing behavior predictable and avoids implicit rewrites.

Key routing characteristics:

| Aspect         | Chi             |
| -------------- | --------------- |
| Matching model | radix tree      |
| Method check   | router          |
| Precedence     | static > param  |
| Parameters     | request context |
| Redirects      | explicit        |

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

	r.GET("/", root)
	r.GET("/users", users)
	r.GET("/users/:id", userByID)

	r.Run(":8080")
}

func root(c *gin.Context) {
	c.String(http.StatusOK, "root")
}

func users(c *gin.Context) {
	c.String(http.StatusOK, "users")
}

func userByID(c *gin.Context) {
	id := c.Param("id")
	c.String(http.StatusOK, "user: %s", id)
}
```

Gin uses a tree structure derived from `httprouter`, with separate trees per HTTP method. Routing begins by selecting the tree for the request method, which eliminates method mismatches early.

Path matching proceeds segment by segment. Static segments are matched before parameter segments. When a parameter node matches, the value is written directly into the request context.

Routing and execution are tightly coupled. The router does more than select a handler. It prepares the execution context, binds parameters, and sets up the middleware chain that will run around the handler.

Trailing slash redirects are enabled by default. Requests to `/users/` may be redirected to `/users`, depending on configuration.

Key routing characteristics:

| Aspect         | Gin            |
| -------------- | -------------- |
| Matching model | tree           |
| Method check   | router         |
| Precedence     | static > param |
| Parameters     | context        |
| Redirects      | default        |

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

	e.GET("/", root)
	e.GET("/users", users)
	e.GET("/users/:id", userByID)

	e.Start(":8080")
}

func root(c echo.Context) error {
	return c.String(http.StatusOK, "root")
}

func users(c echo.Context) error {
	return c.String(http.StatusOK, "users")
}

func userByID(c echo.Context) error {
	return c.String(http.StatusOK, "user: "+c.Param("id"))
}
```

Echo builds and maintains its own routing tree. The router resolves a request to a handler along with the middleware chain that should execute for that route.

Routing determines which code should run next, but it does not perform response writing itself. Execution proceeds through a central dispatcher that runs middleware, invokes the handler, and processes the returned error.

Parameters are stored on the context and retrieved by name. Redirect behavior is configurable and typically disabled unless enabled explicitly.

Key routing characteristics:

| Aspect         | Echo           |
| -------------- | -------------- |
| Matching model | tree           |
| Method check   | router         |
| Precedence     | static > param |
| Parameters     | context        |
| Redirects      | configurable   |

## Fiber

[`fiber/main.go`](fiber/main.go)

```go
package main

import (
	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	app.Get("/", root)
	app.Get("/users", users)
	app.Get("/users/:id", userByID)

	app.Listen(":8080")
}

func root(c *fiber.Ctx) error {
	return c.SendString("root")
}

func users(c *fiber.Ctx) error {
	return c.SendString("users")
}

func userByID(c *fiber.Ctx) error {
	return c.SendString("user: " + c.Params("id"))
}
```

Fiber delegates routing to a fasthttp-based router optimized for low allocation and fast matching. Method and path matching happen together at the routing layer.

When a route matches, parameter values are written directly into the request context. Because contexts and buffers are reused, parameter values must be consumed during handler execution or copied out.

Routing completes before handler execution, and the handler runs immediately after matching.

Key routing characteristics:

| Aspect         | Fiber          |
| -------------- | -------------- |
| Matching model | tree           |
| Method check   | router         |
| Precedence     | static > param |
| Parameters     | context        |
| Redirects      | configurable   |

## Mizu

[`mizu/main.go`](mizu/main.go)

```go
package main

import (
	"github.com/go-mizu/mizu"
)

func main() {
	app := mizu.New()

	app.Get("/", root)
	app.Get("/users", users)
	app.Get("/users/:id", userByID)

	app.Listen(":8080")
}

func root(c *mizu.Ctx) error {
	return c.Text(200, "root")
}

func users(c *mizu.Ctx) error {
	return c.Text(200, "users")
}

func userByID(c *mizu.Ctx) error {
	return c.Text(200, "user: "+c.Param("id"))
}
```

Mizu implements routing on top of net/http while using a tree-based matcher. Method and path matching occur before handler execution. Parameters are captured explicitly and stored on the request-scoped context.

Routing resolves a handler and prepares middleware execution. The routing layer decides what should run, while execution remains a separate concern handled by the dispatcher and error pipeline.

Redirect behavior stays explicit. No automatic rewrites occur unless configured.

Key routing characteristics:

| Aspect         | Mizu           |
| -------------- | -------------- |
| Matching model | tree           |
| Method check   | router         |
| Precedence     | static > param |
| Parameters     | context        |
| Redirects      | explicit       |

## Routing differences that matter

| Framework | Matching model | Method handling | Parameters | Redirect behavior |
| --------- | -------------- | --------------- | ---------- | ----------------- |
| net/http  | pattern rules  | mux             | none       | implicit          |
| Chi       | radix tree     | router          | context    | explicit          |
| Gin       | tree           | router          | context    | default           |
| Echo      | tree           | router          | context    | configurable      |
| Fiber     | tree           | router          | context    | configurable      |
| Mizu      | tree           | router          | context    | explicit          |

## Why this matters

Routing choices affect performance under load, correctness with overlapping paths, and how safely large route sets can evolve. Understanding precedence rules and matching behavior prevents subtle bugs and makes refactoring predictable as applications grow.
