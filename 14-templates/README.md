# Templates and HTML rendering

Templates introduce server side rendering and response composition. Instead of returning raw bytes or structured data, the server now produces HTML that combines static markup with dynamic data. This shifts responsibility from the client to the server and raises new questions about ownership, lifecycle, and failure modes.

At this point in the stack, several technical details start to matter:

* when templates are parsed and cached
* how data is passed into templates
* whether output is streamed directly or buffered first
* how layouts and partials are composed
* what happens when rendering fails after writing has started

All examples render a simple page with a title and a message. The differences lie in how rendering is wired and how much control the framework takes.

## net/http

```go
package main

import (
	"html/template"
	"net/http"
)

var tpl = template.Must(template.New("page").Parse(`
<!doctype html>
<html>
<head><title>{{.Title}}</title></head>
<body>
<h1>{{.Message}}</h1>
</body>
</html>
`))

type Data struct {
	Title   string
	Message string
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tpl.Execute(w, Data{
			Title:   "Home",
			Message: "hello from template",
		}); err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
		}
	})

	http.ListenAndServe(":8080", mux)
}
```

### How template rendering works

Templates are parsed once at program startup. The parsed template is a Go value that can be executed many times concurrently. Parsing errors fail fast at startup rather than at request time.

When a request arrives, the handler calls `Execute` with the response writer and a data value. Template execution writes directly to the writer. This means output is streamed as the template runs.

This streaming behavior has an important consequence. If execution fails after some bytes have already been written, the response status and headers are already committed. At that point, the handler can no longer change the status code or recover cleanly. Many production systems address this by rendering into a buffer first, then copying the result to the response only if rendering succeeds.

The standard library gives full control over parsing, execution, and error handling, but it also makes all tradeoffs explicit.

## Chi

```go
package main

import (
	"html/template"
	"net/http"

	"github.com/go-chi/chi/v5"
)

var tpl = template.Must(template.New("page").Parse(`
<h1>{{.Message}}</h1>
`))

func main() {
	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		tpl.Execute(w, map[string]string{
			"Message": "hello from chi",
		})
	})

	http.ListenAndServe(":8080", r)
}
```

### How template rendering works

Chi does not introduce a rendering abstraction. The handler executes templates exactly as in net/http.

Routing determines which handler runs, but rendering remains an ordinary function call that writes to the response writer. Parsing strategy, buffering, layout composition, and error handling are all application concerns.

This makes Chi a good fit for teams that already have a rendering layer or want to share rendering logic outside HTTP handlers.

## Gin

```go
package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.New()

	r.LoadHTMLGlob("templates/*")

	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "page.html", gin.H{
			"title":   "Home",
			"message": "hello from gin",
		})
	})

	r.Run(":8080")
}
```

### How template rendering works

Gin integrates template rendering into the framework lifecycle.

Templates are parsed during startup when `LoadHTMLGlob` is called and stored on the engine. Layouts and partials are supported through standard Go template definitions.

Calling `c.HTML` sets the status code, headers, and renders the template. Output is buffered internally. Because of buffering, Gin can detect rendering errors before sending headers and still return an appropriate status code.

This approach reduces boilerplate and avoids partial responses, but it also ties rendering configuration to the Gin engine. Rendering is no longer a plain function call independent of the framework.

## Echo

```go
package main

import (
	"html/template"
	"net/http"

	"github.com/labstack/echo/v4"
)

type TemplateRenderer struct {
	t *template.Template
}

func (r *TemplateRenderer) Render(w http.ResponseWriter, name string, data any, c echo.Context) error {
	return r.t.ExecuteTemplate(w, name, data)
}

func main() {
	e := echo.New()

	e.Renderer = &TemplateRenderer{
		t: template.Must(template.ParseGlob("templates/*.html")),
	}

	e.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "page.html", map[string]string{
			"Message": "hello from echo",
		})
	})

	e.Start(":8080")
}
```

### How template rendering works

Echo makes rendering explicit by requiring a renderer interface.

The framework does not assume a specific template engine. Instead, the application provides a renderer that knows how to render templates. This renderer is responsible for parsing, execution, and error behavior.

Handlers call `c.Render`, which delegates to the configured renderer. Errors returned from rendering propagate through Echo’s centralized error handling.

This design makes rendering pluggable and explicit, at the cost of a small amount of setup code.

## Fiber

```go
package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
)

func main() {
	engine := html.New("./templates", ".html")

	app := fiber.New(fiber.Config{
		Views: engine,
	})

	app.Get("/", func(c *fiber.Ctx) error {
		return c.Render("page", fiber.Map{
			"Message": "hello from fiber",
		})
	})

	app.Listen(":8080")
}
```

### How template rendering works

Fiber treats templates as a view engine configured at application creation time.

The engine parses templates and handles rendering. Handlers call `Render`, which produces output and returns an error.

Rendering is buffered. Headers are not sent until rendering completes successfully. This avoids partial responses and makes error handling straightforward.

Template engines live in external packages rather than the core. This keeps the core small but requires choosing and configuring a renderer explicitly.

## Mizu

```go
package main

import (
	"html/template"
	"net/http"

	"github.com/go-mizu/mizu"
)

func main() {
	app := mizu.New()

	tpl := template.Must(template.ParseGlob("templates/*.html"))

	app.Get("/", func(c *mizu.Ctx) error {
		c.SetHeader("Content-Type", "text/html; charset=utf-8")
		return tpl.Execute(c.Writer(), map[string]string{
			"Message": "hello from mizu",
		})
	})

	app.Listen(":8080")
}
```

### How template rendering works

Mizu does not impose a rendering abstraction.

Templates are standard Go templates. Execution is explicit and returns an error. The handler decides how to handle failures and whether to buffer output.

Because Mizu exposes the underlying writer, streaming and incremental rendering are possible. This provides flexibility, but it also requires care when handling errors, since headers may already be sent.

Rendering fits naturally into Mizu’s error-return handler model, but control remains with the application.

## What to focus on

Template rendering highlights how much a framework wants to manage for you.

Important questions to keep in mind:

* whether rendering is streamed or buffered
* whether templates are tied to the router or standalone
* whether errors are handled locally or centrally
* whether rendering logic can be reused outside HTTP handlers

These decisions influence performance, failure behavior, and how easily rendering logic evolves over time.
