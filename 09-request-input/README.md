# Reading requests: headers, query, body

Request input is the boundary between the network and your code. A server parses bytes off a connection, builds a request object, and hands it to your handler. Everything after that depends on where the framework stores data, how it exposes it, and how it treats parsing errors. Headers and query parameters behave like metadata, while the body behaves like a stream. That difference shapes correctness more than any helper API.

This section uses the same logical requests for every framework:

* `GET /search?q=go`
* `POST /echo` with a JSON body

Focus points:

* where header and query data lives and how it is accessed
* when the body is consumed and whether it can be read twice
* where parse errors go and who turns them into HTTP responses
* what can be safely reused and what must be copied

## net/http

`nethttp/main.go`

```go
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /search", search)
	mux.HandleFunc("POST /echo", echo)

	http.ListenAndServe(":8080", mux)
}

func search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	h := r.Header.Get("User-Agent")

	fmt.Fprintf(w, "query=%s ua=%s\n", q, h)
}

func echo(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid json"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payload)
}
```

Headers and query parameters are already available by the time the handler runs. `r.Header` is a map-like structure holding header values, and `r.URL` contains the parsed URL, including query. Calling `r.URL.Query()` returns a parsed view of the query string. These are safe to read repeatedly because the parsing happens outside your handler and the data lives in memory attached to the request.

The body is different. `r.Body` is an `io.ReadCloser`, which represents a forward-only stream of bytes. JSON decoding consumes bytes from that stream. After `Decode` reads the stream, the data is gone unless it was buffered explicitly. A second decode attempt reads from the remaining bytes and typically returns `EOF` or produces incomplete input errors.

One subtle runtime detail: `json.Decoder` can leave trailing bytes unread if the input contains extra data. Many servers add strictness by decoding once and then verifying there is no trailing garbage. The base net/http approach leaves this decision to user code.

Parsing errors remain local. The decoder returns an error, and the handler decides status, body, logging, and whether to terminate.

Useful mental map:

| Input   | Where it lives              | Read cost      | Re-readable |
| ------- | --------------------------- | -------------- | ----------- |
| headers | `r.Header`                  | cheap          | yes         |
| query   | `r.URL` and `r.URL.Query()` | cheap          | yes         |
| body    | `r.Body` stream             | consumes bytes | no          |

## Chi

`chi/main.go`

```go
package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()

	r.Get("/search", search)
	r.Post("/echo", echo)

	http.ListenAndServe(":8080", r)
}

func search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	h := r.Header.Get("User-Agent")

	fmt.Fprintf(w, "query=%s ua=%s\n", q, h)
}

func echo(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid json"))
		return
	}

	json.NewEncoder(w).Encode(payload)
}
```

Chi keeps request input semantics aligned with net/http. The handler receives `http.ResponseWriter` and `*http.Request`, and input parsing uses the same primitives. Chi does not introduce an alternative request object, so the same body-stream rules apply: decoding consumes the body.

The practical difference shows up when route params are involved. Chi stores params in the request context, but headers, query, and body remain standard. That makes it easier to reuse existing parsing libraries that expect `*http.Request`.

Error ownership stays local. Handler code decides how to map decode errors to responses unless an application-level wrapper is introduced.

Useful mental map:

| Input   | Access pattern            |
| ------- | ------------------------- |
| headers | `r.Header.Get`            |
| query   | `r.URL.Query().Get`       |
| body    | `json.NewDecoder(r.Body)` |

## Gin

`gin/main.go`

```go
package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Payload struct {
	Message string `json:"message"`
}

func main() {
	r := gin.New()

	r.GET("/search", func(c *gin.Context) {
		q := c.Query("q")
		ua := c.GetHeader("User-Agent")

		c.String(http.StatusOK, "query=%s ua=%s\n", q, ua)
	})

	r.POST("/echo", func(c *gin.Context) {
		var p Payload
		if err := c.BindJSON(&p); err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		c.JSON(http.StatusOK, p)
	})

	r.Run(":8080")
}
```

Gin exposes request input primarily through its context wrapper. Query and header helpers read from the underlying request but present a uniform API. That hides some details but keeps the common cases direct.

Body parsing happens through bind helpers like `BindJSON`. Under the hood, these helpers read from the request body stream and decode. The body still behaves as a one-shot stream: after it is consumed, the bytes are gone unless Gin is configured to buffer them in a specific mode.

A practical consequence is that calling `BindJSON` and later trying to read `c.Request.Body` again typically fails. In pipelines where multiple layers want access to the raw body, buffering must be introduced deliberately and early.

Error handling uses a hybrid approach. `BindJSON` returns an error, and user code decides whether to abort the chain. The framework does not automatically stop the request on bind failure. The abort call becomes the enforcement mechanism.

Useful mental map:

| Input       | Where it lives                     | Common accessor   |
| ----------- | ---------------------------------- | ----------------- |
| headers     | request + context helpers          | `c.GetHeader`     |
| query       | request + context helpers          | `c.Query`         |
| body        | request stream, decoded via helper | `c.BindJSON`      |
| parse error | returned error + optional abort    | `AbortWithStatus` |

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

	e.GET("/search", func(c echo.Context) error {
		q := c.QueryParam("q")
		ua := c.Request().Header.Get("User-Agent")

		return c.String(http.StatusOK, "query="+q+" ua="+ua)
	})

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

Echo combines direct access to the underlying request with convenience methods on its context. Query lookup is a direct name-based accessor. Headers can be read from `c.Request().Header` or via helpers depending on style.

Body parsing is performed by `Bind`, which consumes the body stream and decodes into a struct. The stream nature remains. Reading twice requires buffering.

The key difference lies in error flow. Handlers return `error`. Bind errors can be returned directly, which routes them into Echo’s centralized error handling. That makes parse errors behave like first-class failures rather than local branches.

A small pattern helps keep parsing strict and readable:

* bind into struct
* return an HTTP error on failure
* proceed with validated struct

Useful mental map:

| Input   | Accessor                 | Failure flow |
| ------- | ------------------------ | ------------ |
| query   | `c.QueryParam`           | none         |
| headers | `c.Request().Header.Get` | none         |
| body    | `c.Bind(&v)`             | return error |

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

	app.Get("/search", func(c *fiber.Ctx) error {
		q := c.Query("q")
		ua := c.Get("User-Agent")

		return c.SendString("query=" + q + " ua=" + ua)
	})

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

Fiber is built on fasthttp. Headers and query values are accessed through the context, backed by fasthttp request structures. These accessors are fast and avoid allocations in many cases.

Body parsing is handled by `BodyParser`, which consumes the request body and decodes into a struct. The one-shot property remains. Multiple parses require buffering or storing the decoded value.

Because Fiber reuses contexts, request-scoped data must be treated as temporary. Values returned by accessors may reference internal buffers. Reading and immediately using values inside the handler is safe. Storing references beyond the handler requires copying.

Useful mental map:

| Input   | Accessor       | Lifetime note  |
| ------- | -------------- | -------------- |
| headers | `c.Get`        | request-scoped |
| query   | `c.Query`      | request-scoped |
| body    | `c.BodyParser` | consumes body  |

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

	app.Get("/search", func(c *mizu.Ctx) error {
		q := c.Query("q")
		ua := c.Request().Header.Get("User-Agent")

		return c.Text(http.StatusOK, "query="+q+" ua="+ua)
	})

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

Mizu exposes the underlying net/http request while providing helpers for common patterns. Query can be read through helper methods. Headers remain available through the underlying request, which keeps compatibility with libraries that expect `*http.Request`.

Body parsing follows the stream rule. `Bind` consumes the body and decodes into a struct. A second bind reads no data unless buffering is added. Parse errors return as `error`, which flows into the framework’s centralized error handling path, consistent with other failures.

This makes ownership clear:

* handler consumes input via explicit bind
* bind returns error on parse failure
* error return controls the response path

Useful mental map:

| Input   | Accessor                 | Failure flow |
| ------- | ------------------------ | ------------ |
| query   | `c.Query`                | none         |
| headers | `c.Request().Header.Get` | none         |
| body    | `c.Bind(&v)`             | return error |

## What to keep in mind

Across all frameworks, the body behaves like a stream. Reading consumes bytes. Helpers can hide the mechanics, but they do not change that fundamental property.

Useful checklist:

* headers and query are safe to read multiple times
* body must be treated as one-shot unless buffered
* parse errors determine control flow and error shape
* context reuse can affect lifetime of returned strings in some stacks
