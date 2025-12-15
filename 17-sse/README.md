# Server Sent Events and streaming APIs

Server Sent Events keep an HTTP connection open and stream data from server to client only. There is no protocol upgrade and no bidirectional channel. The connection starts as normal HTTP and stays that way for its entire lifetime.

This simplicity changes how you think about request handling. There is no final response. A single request turns into a long lived stream of writes.

This section focuses on:

* how the connection remains open
* how data is flushed incrementally
* how client disconnects are detected
* how cancellation and timeouts behave

The example is the same across frameworks:

* the client connects to `/events`
* the server sends one event every second
* the stream ends when the client disconnects

## net/http

```go
package main

import (
	"fmt"
	"net/http"
	"time"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /events", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", 500)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		ctx := r.Context()

		for i := 0; ; i++ {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
				fmt.Fprintf(w, "data: tick %d\n\n", i)
				flusher.Flush()
			}
		}
	})

	http.ListenAndServe(":8080", mux)
}
```

### How SSE works here

This is plain HTTP streaming. The handler writes headers once, then repeatedly writes data chunks followed by `Flush`. Each flush pushes bytes to the client immediately.

The request context is central. When the client disconnects, `r.Context().Done()` is closed and the loop exits.

The handler goroutine lives for the full duration of the connection. There is no framework involvement once the loop starts.

## Chi

```go
package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()

	r.Get("/events", func(w http.ResponseWriter, r *http.Request) {
		flusher := w.(http.Flusher)

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")

		for {
			select {
			case <-r.Context().Done():
				return
			case <-time.After(time.Second):
				fmt.Fprint(w, "data: tick\n\n")
				flusher.Flush()
			}
		}
	})

	http.ListenAndServe(":8080", r)
}
```

### How SSE works here

Chi does not change streaming behavior. Routing happens once, then the handler owns the connection.

After the first write and flush, middleware and router logic are no longer relevant. Cancellation still flows through the request context.

## Gin

```go
package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.New()

	r.GET("/events", func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")

		flusher := c.Writer.(http.Flusher)

		for i := 0; ; i++ {
			select {
			case <-c.Request.Context().Done():
				return
			case <-time.After(time.Second):
				fmt.Fprintf(c.Writer, "data: tick %d\n\n", i)
				flusher.Flush()
			}
		}
	})

	r.Run(":8080")
}
```

### How SSE works here

Gin exposes the underlying response writer, so streaming mirrors net/http.

Middleware runs before the handler starts streaming. Once the first flush happens, aborts and status changes no longer apply.

The request context remains the signal for client disconnects.

## Echo

```go
package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()

	e.GET("/events", func(c echo.Context) error {
		res := c.Response()
		req := c.Request()

		res.Header().Set("Content-Type", "text/event-stream")
		res.Header().Set("Cache-Control", "no-cache")

		for i := 0; ; i++ {
			select {
			case <-req.Context().Done():
				return nil
			case <-time.After(time.Second):
				fmt.Fprintf(res, "data: tick %d\n\n", i)
				res.Flush()
			}
		}
	})

	e.Start(":8080")
}
```

### How SSE works here

Echo gives direct access to the response writer and flush mechanism.

Returning an error after streaming begins has no effect. Headers and body are already sent. The lifetime of the handler matches the lifetime of the connection.

## Fiber

```go
package main

import (
	"bufio"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	app.Get("/events", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/event-stream")
		c.Set("Cache-Control", "no-cache")

		c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
			for i := 0; i < 10; i++ {
				fmt.Fprintf(w, "data: tick %d\n\n", i)
				w.Flush()
				time.Sleep(time.Second)
			}
		})

		return nil
	})

	app.Listen(":8080")
}
```

### How SSE works here

Fiber uses a callback based streaming model.

Instead of writing directly to a response writer, you provide a stream producer function. Fiber controls when the stream starts and ends.

Cancellation and error handling must be handled inside the writer itself. The request context is not exposed in the same way as net/http.

## Mizu

```go
package main

import (
	"fmt"
	"time"

	"github.com/go-mizu/mizu"
)

func main() {
	app := mizu.New()

	app.Get("/events", func(c *mizu.Ctx) error {
		c.SetHeader("Content-Type", "text/event-stream")
		c.SetHeader("Cache-Control", "no-cache")

		ctx := c.Request().Context()

		for i := 0; ; i++ {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(time.Second):
				fmt.Fprintf(c.Writer(), "data: tick %d\n\n", i)
				c.Flush()
			}
		}
	})

	app.Listen(":8080")
}
```

### How SSE works here

Mizu follows the same streaming model as net/http.

Once streaming begins, returned errors are ignored. The handler loop controls the connection lifecycle.

This keeps SSE behavior predictable and compatible with standard HTTP tooling.

## What to take away

SSE stretches the request model in ways that normal APIs do not:

* one request produces many writes
* one goroutine typically serves one client
* cancellation is essential for cleanup
* flushing determines when data becomes visible

The main differences across frameworks are not syntax, but ownership:

* who controls the stream
* how cancellation is exposed
* when the framework steps out of the way

Understanding SSE makes long polling, streaming APIs, and live dashboards much easier to reason about.
