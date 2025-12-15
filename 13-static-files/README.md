# Static files and embedded assets

Serving static content looks straightforward, but the moment a real system is involved, several low level rules come into play. A request for a file is still an HTTP request, which means routing, path normalization, headers, streaming, and error handling all apply. Static serving also becomes the first place where deployment choices matter, especially when assets move from disk into the binary.

This section focuses on what actually happens when a file is requested and how each framework wires that behavior. Two cases are covered throughout:

* serving files from disk
* serving files embedded into the binary

The intent is to understand ownership and data flow, not to memorize helpers.

## net/http

```go
package main

import (
	"embed"
	"net/http"
)

//go:embed public/*
var assets embed.FS

func main() {
	mux := http.NewServeMux()

	// serve from disk
	mux.Handle("/static/",
		http.StripPrefix("/static/",
			http.FileServer(http.Dir("./public")),
		),
	)

	// serve embedded files
	fs := http.FS(assets)
	mux.Handle("/embed/",
		http.StripPrefix("/embed/",
			http.FileServer(fs),
		),
	)

	http.ListenAndServe(":8080", mux)
}
```

### How static serving works

`http.FileServer` is just another `http.Handler`. It takes the request path, cleans it, maps it onto a filesystem rooted at a directory or an `fs.FS`, opens the file, sets headers such as `Content-Type`, `Last-Modified`, and `Content-Length`, then streams the file to the client.

The handler does not know about routing prefixes. That is why `http.StripPrefix` is required. Without stripping `/static` or `/embed`, the file server would attempt to open paths like `./public/static/app.css`, which usually do not exist.

Embedded files work because `http.FS` adapts any `fs.FS` into the interface expected by `FileServer`. From the file server’s point of view, disk and embedded files behave the same. The difference is only in how bytes are retrieved.

Security is handled by path cleaning inside `FileServer`. Directory traversal attempts like `../` are rejected as long as the filesystem root is correctly defined.

## Chi

```go
package main

import (
	"embed"
	"net/http"

	"github.com/go-chi/chi/v5"
)

//go:embed public/*
var assets embed.FS

func main() {
	r := chi.NewRouter()

	r.Handle("/static/*",
		http.StripPrefix("/static/",
			http.FileServer(http.Dir("./public")),
		),
	)

	r.Handle("/embed/*",
		http.StripPrefix("/embed/",
			http.FileServer(http.FS(assets)),
		),
	)

	http.ListenAndServe(":8080", r)
}
```

### How static serving works

Chi does not implement static serving itself. Routing decides which handler runs. Once the request reaches the file server handler, everything is standard library behavior.

The important detail is that Chi does not rewrite request paths internally. Prefix stripping remains an explicit step. That makes static serving predictable and consistent with other handlers.

Because the file server is a normal handler, it participates naturally in middleware chains. Authentication, logging, and compression can wrap static content without special cases.

## Gin

```go
package main

import (
	"embed"

	"github.com/gin-gonic/gin"
)

//go:embed public/*
var assets embed.FS

func main() {
	r := gin.New()

	// serve from disk
	r.Static("/static", "./public")

	// serve embedded files
	r.StaticFS("/embed", gin.FS(assets))

	r.Run(":8080")
}
```

### How static serving works

Gin exposes static serving through helpers. `Static` and `StaticFS` register routes and internally construct file serving handlers.

Prefix handling, content type detection, and common headers are configured for you. This reduces boilerplate, but it also hides the fact that the underlying mechanism is still a file server handler.

Files are streamed rather than fully loaded into memory. The handler decides when headers are sent, and the response behaves like any other Gin response once writing begins.

Because static serving is integrated at the router level, it follows Gin’s execution and abort rules.

## Echo

```go
package main

import (
	"embed"

	"github.com/labstack/echo/v4"
)

//go:embed public/*
var assets embed.FS

func main() {
	e := echo.New()

	e.Static("/static", "public")
	e.StaticFS("/embed", assets)

	e.Start(":8080")
}
```

### How static serving works

Echo provides first class helpers for static files. These helpers configure internal handlers that map request paths to filesystem paths and stream file contents.

Static serving is integrated with Echo’s middleware and error handling pipeline. A missing file typically results in a 404 response handled by the framework rather than a raw handler write.

Because Echo handlers return errors, static serving failures flow through the same centralized error logic as application handlers.

## Fiber

```go
package main

import (
	"embed"

	"github.com/gofiber/fiber/v2"
)

//go:embed public/*
var assets embed.FS

func main() {
	app := fiber.New()

	app.Static("/static", "./public")
	app.Static("/embed", "./public") // Fiber does not support embed.FS directly

	app.Listen(":8080")
}
```

### How static serving works

Fiber uses fasthttp based file serving utilities. Static handlers are optimized for speed and often buffer more aggressively than net/http based servers.

Direct support for `embed.FS` is limited. Many Fiber applications either avoid embedding static assets or copy embedded files to disk during startup.

This limitation follows from Fiber’s non net/http foundation. The tradeoff favors performance and simplicity over compatibility with standard library abstractions.

## Mizu

```go
package main

import (
	"embed"
	"net/http"

	"github.com/go-mizu/mizu"
)

//go:embed public/*
var assets embed.FS

func main() {
	app := mizu.New()

	app.Static("/static", "./public")
	app.StaticFS("/embed", http.FS(assets))

	app.Listen(":8080")
}
```

### How static serving works

Mizu exposes static serving explicitly while keeping net/http semantics intact.

`Static` serves files from disk. `StaticFS` serves files from any `fs.FS`, including embedded assets. Internally, Mizu constructs handlers using standard file serving logic and integrates them into its routing and middleware pipeline.

Because static handlers are ordinary handlers, middleware such as logging, authentication, or rate limiting applies consistently. Embedded and disk based assets behave the same at request time.

## What to focus on

Static serving reveals how much a framework hides or exposes.

Important differences to pay attention to:

* whether path rewriting is explicit or implicit
* whether embedded files are first class
* whether responses are streamed or buffered
* how missing files are translated into HTTP responses

These details influence security posture, memory usage, and how easily assets move between development and deployment.
