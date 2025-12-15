# Writing responses: status, headers, streaming

Response output is stateful. Input can be inspected repeatedly, but output commits the connection to a specific status line, header set, and body bytes. After the first byte of the response is sent, the server cannot change its mind. That single constraint drives most of the differences between frameworks.

Three things happen during a response:

* metadata is assembled: status code and headers
* the first write commits headers and status to the wire
* body bytes are produced, either buffered and sent at once, or streamed in chunks

This section uses three patterns:

* a normal text response
* a JSON response with headers
* a streaming response that writes multiple chunks

The deep dive focuses on when headers become immutable, what “write twice” means in each stack, and what error handling can realistically do after output has started.

## net/http

`nethttp/main.go`

```go
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /text", text)
	mux.HandleFunc("GET /json", jsonResp)
	mux.HandleFunc("GET /stream", stream)

	http.ListenAndServe(":8080", mux)
}

func text(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("hello"))
}

func jsonResp(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "hello",
	})
}

func stream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", 500)
		return
	}

	for i := 0; i < 3; i++ {
		fmt.Fprintf(w, "chunk %d\n", i)
		flusher.Flush()
		time.Sleep(time.Second)
	}
}
```

In net/http, `http.ResponseWriter` is a live stream to the client. The server sends headers the moment the response is committed. Commitment happens on either of these events:

* an explicit `WriteHeader(status)`
* the first `Write([]byte(...))`, which implies `200 OK` if `WriteHeader` was never called

After commitment, status code cannot change and headers become effectively immutable. Setting headers after the first write has no effect because the header block has already been sent.

Two gotchas show up quickly:

* calling `WriteHeader` twice does not “update” the status; only the first call matters
* writing a partial body and then deciding an error happened can only affect logging, because the client already received part of the response

Streaming happens when the response is not fully buffered inside the server. `http.Flusher` exposes a mechanism to push buffered bytes down the connection before the handler returns. A handler that writes chunks and calls `Flush` can create observable partial progress on the client.

Response commitment model:

| Action                               | What happens                     |
| ------------------------------------ | -------------------------------- |
| set header before write              | header will be sent              |
| first `Write` without `WriteHeader`  | status becomes 200, headers sent |
| `WriteHeader(500)` after first write | ignored                          |
| `Flush()`                            | pushes current buffered bytes    |

## Chi

`chi/main.go`

```go
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()

	r.Get("/text", text)
	r.Get("/json", jsonResp)
	r.Get("/stream", stream)

	http.ListenAndServe(":8080", r)
}

func text(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello"))
}

func jsonResp(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "hello",
	})
}

func stream(w http.ResponseWriter, r *http.Request) {
	flusher := w.(http.Flusher)

	for i := 0; i < 3; i++ {
		fmt.Fprintf(w, "chunk %d\n", i)
		flusher.Flush()
		time.Sleep(time.Second)
	}
}
```

Chi preserves net/http response semantics because the handler contract stays `http.ResponseWriter` and `*http.Request`. Every important rule remains the same: first write commits headers and status, and streaming uses `http.Flusher`.

The deep difference is not response output, but where response output can be wrapped. Since everything is a normal net/http handler, any buffering, compression, logging, or response recording layers can be introduced as middleware around the router.

One practical detail for streaming: the direct type assertion `w.(http.Flusher)` will panic if the writer does not support flushing. The net/http version used a safe check. That check matters when running under special writers such as test recorders.

Response model:

* same commit rules as net/http
* same streaming mechanism as net/http

## Gin

`gin/main.go`

```go
package main

import (
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.New()

	r.GET("/text", func(c *gin.Context) {
		c.String(http.StatusOK, "hello")
	})

	r.GET("/json", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "hello"})
	})

	r.GET("/stream", func(c *gin.Context) {
		c.Stream(func(w io.Writer) bool {
			for i := 0; i < 3; i++ {
				w.Write([]byte("chunk\n"))
				time.Sleep(time.Second)
			}
			return false
		})
	})

	r.Run(":8080")
}
```

Gin writes through its context. Helpers such as `String` and `JSON` coordinate three things in one call: status code, headers, and body encoding. Under the hood, Gin still sits on net/http, so headers ultimately commit when bytes are written to the underlying `ResponseWriter`.

The context acts as the policy surface. Multiple response helpers called in the same handler create an ordering problem: once the response has started, later attempts at setting status or headers are ineffective. Gin tracks response state and encourages a single “final response” path.

Streaming uses Gin’s `Stream` helper, which hands an `io.Writer` into a callback. The framework controls when headers are written and how the response is driven. This makes streaming feel structured, but it also means the streaming control flow is framework-owned, not a plain `Flusher` loop.

Commit behavior still follows the same physical rule: once bytes hit the underlying writer, headers are committed. The difference is that Gin tends to centralize response construction through helpers, which reduces accidental partial writes.

Execution expectations:

| Pattern   | Typical approach                           |
| --------- | ------------------------------------------ |
| text      | `c.String(status, ...)`                    |
| JSON      | `c.JSON(status, ...)`                      |
| streaming | `c.Stream(func(w io.Writer) bool { ... })` |

## Echo

`echo/main.go`

```go
package main

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()

	e.GET("/text", func(c echo.Context) error {
		return c.String(http.StatusOK, "hello")
	})

	e.GET("/json", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "hello"})
	})

	e.GET("/stream", func(c echo.Context) error {
		c.Response().Header().Set("Content-Type", "text/plain")
		for i := 0; i < 3; i++ {
			c.Response().Write([]byte("chunk\n"))
			c.Response().Flush()
			time.Sleep(time.Second)
		}
		return nil
	})

	e.Start(":8080")
}
```

Echo offers response helpers while still exposing the underlying response writer through `c.Response()`. That gives two styles:

* helper-driven: `c.String`, `c.JSON`
* writer-driven: `c.Response().Write`, `c.Response().Flush`

This is useful for streaming because streaming often needs control that high-level helpers deliberately abstract away.

Echo’s error return affects response writing after commitment. If bytes have already been written and a handler returns an error, the framework can run the error handler, but it cannot reliably replace the response because the connection is already committed. In practice, once streaming starts, error returns become mostly a logging and cleanup channel.

Commit rules still derive from net/http. The main difference is that Echo makes “response already started” a practical concept because its centralized error pipeline depends on whether it still has the ability to write a fresh response.

## Fiber

`fiber/main.go`

```go
package main

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	app.Get("/text", func(c *fiber.Ctx) error {
		return c.SendString("hello")
	})

	app.Get("/json", func(c *fiber.Ctx) error {
		return c.JSON(map[string]string{"message": "hello"})
	})

	app.Get("/stream", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/plain")
		for i := 0; i < 3; i++ {
			c.WriteString("chunk\n")
			time.Sleep(time.Second)
		}
		return nil
	})

	app.Listen(":8080")
}
```

Fiber uses a fasthttp-based response model. The response is typically constructed in memory associated with the request context and written out when the handler returns. That makes “writing twice” mean “append to the response body buffer” rather than “send multiple independent chunks to a connection”.

In this setup, repeated writes accumulate. Headers can usually be modified until the response is finalized. That changes the failure story: many errors can be handled late because nothing has been sent yet.

Streaming exists in Fiber, but it is not the default model. A plain loop writing strings does not guarantee immediate delivery to the client. The framework decides when to flush bytes to the network based on its own streaming APIs and transport behavior.

This makes typical API responses straightforward and efficient, while requiring explicit choices for real streaming semantics.

High-level behavior:

| Operation           | Typical behavior      |
| ------------------- | --------------------- |
| multiple writes     | append to body buffer |
| header changes late | often still effective |
| streaming loop      | not guaranteed flush  |

## Mizu

`mizu/main.go`

```go
package main

import (
	"net/http"
	"time"

	"github.com/go-mizu/mizu"
)

func main() {
	app := mizu.New()

	app.Get("/text", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "hello")
	})

	app.Get("/json", func(c *mizu.Ctx) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "hello"})
	})

	app.Get("/stream", func(c *mizu.Ctx) error {
		c.SetHeader("Content-Type", "text/plain")
		for i := 0; i < 3; i++ {
			c.Write([]byte("chunk\n"))
			c.Flush()
			time.Sleep(time.Second)
		}
		return nil
	})

	app.Listen(":8080")
}
```

Mizu provides response helpers for the common cases while keeping low-level control available for streaming. `Text` and `JSON` establish status and write output in a single step. That reduces accidental “status set too late” situations and makes handler intent obvious.

For streaming, `Write` and `Flush` expose a net/http-like model: once the first write happens, headers are committed. Calling `Flush` pushes data through the underlying writer as chunks are produced.

The important constraint remains: after streaming starts, later errors cannot reliably change the response. An error returned after output begins becomes a signal for logging or middleware cleanup rather than a mechanism to generate a new HTTP error response.

Response commitment model:

* first write commits headers and status
* streaming requires explicit `Flush`
* error return after commitment cannot replace the response

## What to keep in mind

Response semantics shape correctness:

* once the first byte is sent, status and headers are locked
* buffering makes late decisions possible, but changes streaming behavior
* streaming gives fine control, but makes mid-flight error recovery mostly impossible

A quick comparison:

| Framework | Default model            | Streaming style            |
| --------- | ------------------------ | -------------------------- |
| net/http  | direct to writer         | `http.Flusher`             |
| Chi       | direct to writer         | `http.Flusher`             |
| Gin       | helper-driven on writer  | framework streaming helper |
| Echo      | helper + writer access   | manual write + flush       |
| Fiber     | buffered response build  | explicit streaming APIs    |
| Mizu      | helpers + writer control | write + flush              |
