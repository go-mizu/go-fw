# JSON input and output

JSON APIs look simple on the surface: read bytes, decode into a struct, write bytes back. The hard parts start when a service grows and needs predictable failure behavior. JSON decoding consumes the request body stream, so the decision to decode in a helper, in middleware, or in the handler changes what later layers can do. The other pressure point is error shape: an invalid JSON body, a missing field, and an unexpected server failure all need to turn into consistent HTTP responses without duplicating logic across every handler.

This section keeps the scenarios small:

* decode JSON into a struct
* handle invalid JSON
* encode a JSON response

The focus stays on ownership and guarantees:

* where decoding happens
* when the body is consumed and whether it can be read again
* how errors propagate and where they become HTTP responses
* what validation means in practice and where it fits

## net/http

`nethttp/main.go`

```go
package main

import (
	"encoding/json"
	"net/http"
)

type Payload struct {
	Message string `json:"message"`
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /echo", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var p Payload
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(p)
	})

	http.ListenAndServe(":8080", mux)
}
```

Decoding is fully owned by the handler. The handler decides which decoder to use, when to close the body, and what happens on failure. The request body is an `io.Reader`, so decoding consumes it. A second decode attempt reads no bytes unless the handler buffered the body beforehand.

The default `json.Decoder` behavior matters for correctness. A typical API eventually wants two extra checks:

* reject unknown fields for schema stability
* reject trailing garbage after the first JSON value

A strict pattern often looks like this:

```go
dec := json.NewDecoder(r.Body)
dec.DisallowUnknownFields()
if err := dec.Decode(&p); err != nil { /* 400 */ }
if dec.More() { /* 400 */ }
```

Encoding is symmetric but still manual. Header choice and status code choice remain the handler’s responsibility. If encoding fails after headers have already been written, the handler cannot reliably change the status code. For JSON APIs, that pushes many teams toward writing through a buffer and only committing after encoding succeeds.

Ownership summary:

| Step          | Owner                     |
| ------------- | ------------------------- |
| decode        | handler code              |
| validation    | handler or custom library |
| error mapping | handler code              |
| encode        | handler code              |

## Chi

`chi/main.go`

```go
package main

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Payload struct {
	Message string `json:"message"`
}

func main() {
	r := chi.NewRouter()

	r.Post("/echo", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var p Payload
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		json.NewEncoder(w).Encode(p)
	})

	http.ListenAndServe(":8080", r)
}
```

Chi keeps the net/http JSON story unchanged. That means the same strengths and constraints: decoding is handler-owned, body consumption happens once, and failure mapping is a local decision unless a shared helper layer is introduced.

The net benefit is composability. Any decoding, strictness, or validation strategy built for `*http.Request` plugs in directly. Many services with Chi end up creating a small internal layer that standardizes “decode + validate + respond” and use it across endpoints.

A practical pattern looks like this:

* middleware sets request size limits and maybe a maximum body reader
* handler calls a shared decode helper
* helper returns typed errors that a shared error handler maps to JSON error responses

Ownership summary:

| Concern                  | Result           |
| ------------------------ | ---------------- |
| JSON semantics           | same as net/http |
| reuse outside framework  | straightforward  |
| centralized error format | app-defined      |

## Gin

`gin/main.go`

```go
package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Payload struct {
	Message string `json:"message" binding:"required"`
}

func main() {
	r := gin.New()

	r.POST("/echo", func(c *gin.Context) {
		var p Payload
		if err := c.ShouldBindJSON(&p); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, p)
	})

	r.Run(":8080")
}
```

Gin concentrates JSON decoding in a context method. `ShouldBindJSON` reads the body and decodes into the struct. The body remains a one-shot stream underneath; the helper does not change that fundamental constraint. After binding, later layers cannot re-read the raw body unless buffering was configured earlier.

The distinctive part is the validation integration. The struct tag `binding:"required"` participates in Gin’s binding and validation pipeline. That makes “missing field” feel like a decode failure at the call site because both surface as an `err`. This reduces boilerplate but couples validation behavior to Gin’s binding system and tag conventions.

Error propagation uses explicit control flow. The handler receives an error, decides to abort, and writes a response. Without aborting, middleware or later handlers can still run. That matters when teams build shared middleware that expects “bind failure stops everything” and forgets the abort.

A common stable pattern in Gin handlers:

```go
if err := c.ShouldBindJSON(&p); err != nil {
  c.AbortWithStatusJSON(400, gin.H{"error": "bad request"})
  return
}
```

Ownership summary:

| Step           | Owner                        |
| -------------- | ---------------------------- |
| decode         | Gin binding                  |
| validation     | Gin binding tags             |
| error mapping  | handler or shared middleware |
| stop execution | handler via abort            |

## Echo

`echo/main.go`

```go
package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type Payload struct {
	Message string `json:"message"`
}

func main() {
	e := echo.New()

	e.POST("/echo", func(c echo.Context) error {
		var p Payload
		if err := c.Bind(&p); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid json")
		}

		return c.JSON(http.StatusOK, p)
	})

	e.Start(":8080")
}
```

Echo’s `Bind` consumes the body and decodes into the struct. The important difference is how failures flow. The handler returns an `error`, so bind failures can be returned directly and handled by the centralized error handler. That gives a natural place to standardize error responses without repeating response-writing logic across every handler.

Echo keeps validation separate from decoding by default. That separation can be useful: decoding answers “is it valid JSON”, while validation answers “is it acceptable input”. Many services implement validation via middleware or explicit calls after binding.

A common pattern:

```go
if err := c.Bind(&p); err != nil { return echo.NewHTTPError(400, "invalid json") }
if p.Message == "" { return echo.NewHTTPError(400, "message required") }
return c.JSON(200, p)
```

Ownership summary:

| Concern        | Owner               |
| -------------- | ------------------- |
| decode         | Echo bind           |
| validation     | app layer           |
| error mapping  | centralized handler |
| stop execution | returning error     |

## Fiber

`fiber/main.go`

```go
package main

import (
	"github.com/gofiber/fiber/v2"
)

type Payload struct {
	Message string `json:"message"`
}

func main() {
	app := fiber.New()

	app.Post("/echo", func(c *fiber.Ctx) error {
		var p Payload
		if err := c.BodyParser(&p); err != nil {
			return c.Status(400).SendString("invalid json")
		}
		return c.JSON(p)
	})

	app.Listen(":8080")
}
```

Fiber parses JSON through `BodyParser`. It consumes the body and fills the struct, returning an error on failure. Validation is typically layered above, similar to Echo, but the runtime model differs because Fiber is fasthttp-based and uses a pooled context.

For JSON response encoding, `c.JSON` handles encoding and content type. The response often behaves like a buffered build rather than immediate streaming, which makes it easier to commit a single consistent JSON response.

When building consistent API error shapes, teams typically standardize around a helper:

```go
func badRequest(c *fiber.Ctx, msg string) error {
  return c.Status(400).JSON(fiber.Map{"error": msg})
}
```

Ownership summary:

| Step            | Owner                    |
| --------------- | ------------------------ |
| decode          | Fiber parser             |
| validation      | app layer                |
| error mapping   | handler or shared helper |
| response encode | Fiber JSON helper        |

## Mizu

`mizu/main.go`

```go
package main

import (
	"net/http"

	"github.com/go-mizu/mizu"
)

type Payload struct {
	Message string `json:"message"`
}

func main() {
	app := mizu.New()

	app.Post("/echo", func(c *mizu.Ctx) error {
		var p Payload
		if err := c.Bind(&p); err != nil {
			return err
		}
		return c.JSON(http.StatusOK, p)
	})

	app.Listen(":8080")
}
```

Mizu makes JSON decoding an explicit operation that returns an error, then lets that error flow through the same error-handling path as other failures. The body is consumed during `Bind`, so the one-shot rule applies. The value is in consistency: decoding failures look like normal handler failures and can be handled centrally.

Encoding is explicit and symmetric via `c.JSON(status, value)`. That encourages a clean “parse input, validate input, return output” structure without mixing manual header setting and ad-hoc encoding in every handler.

A practical pattern is to keep validation separate but error-driven:

```go
if err := c.Bind(&p); err != nil { return err }
if p.Message == "" { return mizu.HTTPError{Status: 400, Err: errors.New("message required")} }
return c.JSON(200, p)
```

Ownership summary:

| Concern       | Owner                      |
| ------------- | -------------------------- |
| decode        | Mizu bind                  |
| validation    | app layer                  |
| error mapping | centralized error handling |
| encode        | Mizu JSON helper           |

## What to keep in mind

The main differences come from ownership and error flow:

* decoding in the handler gives maximum control but repeats logic
* decoding via context helpers reduces boilerplate but binds input parsing to framework types
* a returned-error handler model makes centralized error formatting easier
* validation can be coupled to binding tags or kept as an explicit app-layer step

A compact comparison:

| Framework | Decode API     | Validation          | Error flow               |
| --------- | -------------- | ------------------- | ------------------------ |
| net/http  | manual decoder | manual              | handler writes response  |
| Chi       | manual decoder | manual              | handler writes response  |
| Gin       | bind helper    | tag-driven optional | handler aborts + writes  |
| Echo      | bind helper    | separate layer      | return error             |
| Fiber     | body parser    | separate layer      | return error or response |
| Mizu      | bind helper    | separate layer      | return error             |
