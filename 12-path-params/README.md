# Path parameters and typed access

Path parameters change routing from “does this string match” into “does this string match, and can the router hand me the interesting pieces cheaply and safely”. Once a pattern like `/users/:id` exists, the router performs extra work on every matching request: it must split or walk the path, decide which segments are variables, capture values, and expose them through a request-scoped API. That API then becomes the foundation for typed access. Every framework hands parameters back as strings, so typed access is really two parts: capture and conversion.

Capture has performance and correctness edges:

* capture must be request-scoped and safe under concurrency
* capture should avoid allocations when possible, but still be stable for the duration of the handler
* wildcard capture (`*path`) must decide whether the value includes a leading slash, and what happens when it captures nothing

Conversion has API design edges:

* conversion decides the error shape for malformed input
* conversion decides whether a missing parameter is a route mismatch or a handler-level error
* conversion determines how consistent failures are across the service

This section implements:

* `GET /users/:id` where `id` should parse as `int64`
* `GET /files/*path` where `path` is a wildcard segment

Then it discusses:

* route matches but `id` fails to parse
* route does not match
* wildcard capture yields an empty value

## net/http

`nethttp/main.go`

```go
package main

import (
	"fmt"
	"net/http"
	"strconv"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /users/{id}", getUser)
	mux.HandleFunc("GET /files/{path...}", getFile)

	http.ListenAndServe(":8080", mux)
}

func getUser(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "invalid id: %q\n", idStr)
		return
	}

	fmt.Fprintf(w, "user id=%d\n", id)
}

func getFile(w http.ResponseWriter, r *http.Request) {
	p := r.PathValue("path")
	fmt.Fprintf(w, "file path=%q\n", p)
}
```

`ServeMux` captures parameters as part of the pattern matcher. The capture storage is tied to the request so handlers read values through `r.PathValue(name)`. The important property: the mux owns capture, the handler owns conversion.

Conversion patterns usually fall into one of two styles:

* parse inline and write a 400 directly
* parse inline and return a typed error to a central error layer

A compact typed parse helper keeps handlers consistent:

```go
func pathInt64(r *http.Request, key string) (int64, error) {
  s := r.PathValue(key)
  if s == "" {
    return 0, fmt.Errorf("missing %s", key)
  }
  return strconv.ParseInt(s, 10, 64)
}
```

Wildcard `{path...}` captures the remainder of the path, including slashes. When the route matches `/files/`, the captured value can be empty depending on the exact incoming path. That makes `""` a valid capture and handlers should treat it intentionally.

Behavior focus:

| Case         | Result                                  |
| ------------ | --------------------------------------- |
| `/users/123` | match, `PathValue("id") == "123"`       |
| `/users/x`   | match, parse fails, handler decides 400 |
| `/users`     | no match                                |
| `/files/a/b` | match, `PathValue("path") == "a/b"`     |
| `/files/`    | match, `PathValue("path")` may be `""`  |

## Chi

`chi/main.go`

```go
package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()

	r.Get("/users/{id}", getUser)
	r.Get("/files/*", getFile)

	http.ListenAndServe(":8080", r)
}

func getUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "invalid id: %q\n", idStr)
		return
	}

	fmt.Fprintf(w, "user id=%d\n", id)
}

func getFile(w http.ResponseWriter, r *http.Request) {
	p := chi.URLParam(r, "*")
	fmt.Fprintf(w, "file path=%q\n", p)
}
```

Chi captures parameters during route matching and stores them in routing state attached to the request context. Retrieval via `chi.URLParam` keeps handler signatures net/http-shaped, but the lookup is router-defined.

Typed conversion stays handler-owned. For consistency, many Chi codebases wrap conversion in helpers that return `(T, bool)` or `(T, error)` and centralize 400 formatting.

Wildcard capture uses `/*` and is retrieved as `"*"`. That special key becomes part of the service’s conventions. It helps to normalize it immediately:

```go
p := chi.URLParam(r, "*")
if p == "" { /* empty capture path */ }
```

Chi routes do not rewrite the request path, so captured values correspond to the real incoming URL path. That makes logs and redirects consistent without extra work.

## Gin

`gin/main.go`

```go
package main

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.New()

	r.GET("/users/:id", func(c *gin.Context) {
		idStr := c.Param("id")

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "invalid id",
				"id":    idStr,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{"id": id})
	})

	r.GET("/files/*path", func(c *gin.Context) {
		// Gin includes the leading slash in wildcard params by default.
		p := c.Param("path")
		c.String(http.StatusOK, "file path=%q", p)
	})

	r.Run(":8080")
}
```

Gin captures params in the router and exposes them through the context. Retrieval stays string-based, so typed access is always an explicit parse step.

The important internal behavior difference is wildcard normalization. Gin’s `*path` typically includes a leading slash in the captured value. That affects code that wants to join paths safely. A stable pattern trims it once:

```go
p := c.Param("path")
if len(p) > 0 && p[0] == '/' {
  p = p[1:]
}
```

Conversion failures typically stop the request through abort semantics rather than an error return. That makes “typed parameter parsing” part of the request-control policy of the service.

Behavior focus:

| Case           | Typical handler choice             |
| -------------- | ---------------------------------- |
| parse fails    | `AbortWithStatusJSON(400, ...)`    |
| wildcard value | normalize leading slash before use |

## Echo

`echo/main.go`

```go
package main

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()

	e.GET("/users/:id", func(c echo.Context) error {
		idStr := c.Param("id")

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
		}

		return c.JSON(http.StatusOK, map[string]any{"id": id})
	})

	e.GET("/files/*", func(c echo.Context) error {
		// Echo exposes wildcard captures with Param("*")
		p := c.Param("*")
		return c.String(http.StatusOK, "file path="+p)
	})

	e.Start(":8080")
}
```

Echo stores route params on its context, and `Param(name)` retrieves them. Typed parsing remains explicit. The main difference is the error model: parse failure can return an error value, which flows into centralized error handling.

That error return path makes typed access patterns easier to standardize. A helper that returns `(int64, error)` composes naturally:

```go
func paramInt64(c echo.Context, key string) (int64, error) {
  s := c.Param(key)
  id, err := strconv.ParseInt(s, 10, 64)
  if err != nil {
    return 0, echo.NewHTTPError(400, "invalid "+key)
  }
  return id, nil
}
```

Wildcard capture uses `*` and is retrieved as `Param("*")`. Empty capture should be treated intentionally as a valid match for routes like `/files/`.

## Fiber

`fiber/main.go`

```go
package main

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	app.Get("/users/:id", func(c *fiber.Ctx) error {
		idStr := c.Params("id")

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return c.Status(400).SendString("invalid id")
		}

		return c.JSON(map[string]any{"id": id})
	})

	app.Get("/files/*", func(c *fiber.Ctx) error {
		// Fiber wildcard is Params("*")
		p := c.Params("*")
		return c.SendString("file path=" + p)
	})

	app.Listen(":8080")
}
```

Fiber captures params during match and stores them in the request context object, which is pooled. The handler reads params as strings via `Params(name)`.

For typed access, the main practical rule is to treat param values as request-scoped data. Parse immediately into a typed value and store the typed result if it needs to move deeper into the application.

Wildcard capture is retrieved through `Params("*")`. Empty capture can happen for `/files/` and should be treated as either a valid “root of files” request or an input error, depending on service policy.

## Mizu

`mizu/main.go`

```go
package main

import (
	"net/http"
	"strconv"

	"github.com/go-mizu/mizu"
)

func main() {
	app := mizu.New()

	app.Get("/users/:id", func(c *mizu.Ctx) error {
		idStr := c.Param("id")

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return c.Text(http.StatusBadRequest, "invalid id")
		}

		return c.JSON(http.StatusOK, map[string]any{"id": id})
	})

	app.Get("/files/*path", func(c *mizu.Ctx) error {
		p := c.Param("path")
		return c.Text(http.StatusOK, "file path="+p)
	})

	app.Listen(":8080")
}
```

Mizu captures params during routing and stores them on the request-scoped context. Retrieval via `c.Param(name)` keeps handlers framework-shaped while keeping the conversion step explicit and visible.

Typed access is usually structured as a small helper that returns `(T, error)` and plugs into the handler’s error-return model:

```go
func paramInt64(c *mizu.Ctx, key string) (int64, error) {
  s := c.Param(key)
  id, err := strconv.ParseInt(s, 10, 64)
  if err != nil {
    return 0, mizu.HTTPError{Status: 400, Err: err}
  }
  return id, nil
}
```

Wildcard capture uses a named star segment `*path`, which avoids the `"*"` magic key pattern and makes code more self-documenting. Empty capture should be treated explicitly, especially for `/files/` style routes.

## What to pay attention to

Every stack follows the same broad shape: route matches, router captures strings, handler parses to types. The real differences show up in the edges:

* where the captured values live (request context, router context, pooled context)
* how wildcards are represented and whether the captured value includes a leading slash
* how parse failures stop execution and how error responses are formed

A compact comparison:

| Framework | Param API              | Wildcard API                       | Failure style            |
| --------- | ---------------------- | ---------------------------------- | ------------------------ |
| net/http  | `r.PathValue("id")`    | `{path...}`                        | handler writes response  |
| Chi       | `chi.URLParam(r,"id")` | key `"*"`                          | handler writes response  |
| Gin       | `c.Param("id")`        | `*path` often includes leading `/` | abort + write response   |
| Echo      | `c.Param("id")`        | `Param("*")`                       | return error             |
| Fiber     | `c.Params("id")`       | `Params("*")`                      | write response / return  |
| Mizu      | `c.Param("id")`        | `*path` named                      | return error or response |
