# Context, deadlines, and cancellation

Context is how cancellation, deadlines, and request scoped signals move through an HTTP server. It matters most once requests stop being short and predictable. As soon as a handler performs slow work, waits on external systems, or streams data over time, the ability to stop that work cleanly becomes essential.

Context does not stop anything by itself. It is a signal that travels alongside a request. Every piece of code that cares about cancellation must cooperate by observing that signal and exiting when it changes.

This section focuses on:

* where the request context comes from
* how cancellation propagates through handlers and middleware
* how deadlines are enforced
* what actually happens when the client disconnects

The example is the same everywhere:

* `GET /work`
* the handler simulates long running work
* execution stops immediately when the client disconnects or a deadline expires

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

	mux.HandleFunc("GET /work", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		for i := 0; i < 10; i++ {
			select {
			case <-ctx.Done():
				fmt.Println("canceled:", ctx.Err())
				return
			case <-time.After(time.Second):
				fmt.Fprintf(w, "step %d\n", i)
			}
		}
	})

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	srv.ListenAndServe()
}
```

### How context works here

In net/http, the request context is created by the server for every incoming request. That context is canceled when one of several events occurs:

* the client closes the connection
* the server begins shutting down
* a timeout or deadline is reached

Handlers access the context through `r.Context()`. Nothing happens automatically when the context is canceled. No goroutines are stopped. No panics are raised. The only thing that changes is that `<-ctx.Done()` becomes readable.

Long running handlers must actively select on `ctx.Done()` and exit when it fires. If they ignore it, work continues even after the client has gone away.

Deadlines are enforced by canceling the context. The handler observes the same signal whether the cause is a timeout, a disconnect, or a shutdown.

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

	r.Get("/work", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		for i := 0; i < 10; i++ {
			select {
			case <-ctx.Done():
				fmt.Println("canceled")
				return
			case <-time.After(time.Second):
				fmt.Fprintf(w, "step %d\n", i)
			}
		}
	})

	http.ListenAndServe(":8080", r)
}
```

### How context works here

Chi preserves the standard net/http context model.

The router does not create its own cancellation mechanism. It forwards the request context unchanged. Middleware that wants to add deadlines or values does so by wrapping the existing context with `context.WithTimeout` or `context.WithValue`.

From the handler’s point of view, there is no difference between Chi and net/http. Cancellation signals arrive through the same channel and must be handled the same way.

## Gin

```go
package main

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.New()

	r.GET("/work", func(c *gin.Context) {
		ctx := c.Request.Context()

		for i := 0; i < 10; i++ {
			select {
			case <-ctx.Done():
				fmt.Println("canceled")
				return
			case <-time.After(time.Second):
				c.Writer.Write([]byte("step\n"))
			}
		}
	})

	r.Run(":8080")
}
```

### How context works here

Gin uses the standard request context carried by `*http.Request`.

There is no separate Gin cancellation context. The `gin.Context` holds request scoped data and helpers, but cancellation always comes from `c.Request.Context()`.

Gin middleware that enforces timeouts does so by replacing the request context before invoking the next handler. The cancellation signal still propagates through the same mechanism.

Handlers must explicitly observe the context. Writing to the response does not imply cancellation awareness.

## Echo

```go
package main

import (
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()

	e.GET("/work", func(c echo.Context) error {
		ctx := c.Request().Context()

		for i := 0; i < 10; i++ {
			select {
			case <-ctx.Done():
				fmt.Println("canceled")
				return nil
			case <-time.After(time.Second):
				c.Response().Write([]byte("step\n"))
			}
		}
		return nil
	})

	e.Start(":8080")
}
```

### How context works here

Echo also relies on the standard request context.

Returning an error does not cancel execution. Errors are about control flow and response handling. Cancellation is a separate concern and must be checked explicitly through the context.

This separation makes long running handlers predictable. Only the context determines when work should stop.

## Fiber

```go
package main

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	app.Get("/work", func(c *fiber.Ctx) error {
		for i := 0; i < 10; i++ {
			time.Sleep(time.Second)
			c.WriteString("step\n")
		}
		return nil
	})

	app.Listen(":8080")
}
```

### How context works here

Fiber does not expose request cancellation through `context.Context` in the same way as net/http based frameworks.

Because Fiber is built on fasthttp, client disconnects are not surfaced as a cancelable context. Long running handlers must detect cancellation indirectly, usually through write errors or framework specific signals.

This changes how you design slow handlers. You cannot rely on a single shared cancellation primitive. Cleanup logic often lives closer to I/O operations.

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

	app.Get("/work", func(c *mizu.Ctx) error {
		ctx := c.Request().Context()

		for i := 0; i < 10; i++ {
			select {
			case <-ctx.Done():
				fmt.Println("canceled")
				return nil
			case <-time.After(time.Second):
				c.Write([]byte("step\n"))
				c.Flush()
			}
		}
		return nil
	})

	app.Listen(":8080")
}
```

### How context works here

Mizu preserves the net/http context model.

Handlers and middleware observe the same cancellation signals as in the standard library. Deadlines, disconnects, and shutdown all surface through the request context.

Once streaming begins, cancellation still works. The handler must check the context and exit cooperatively.

## What to take away

Context is the backbone of cancellation in Go HTTP servers.

Key points that matter in real systems:

* cancellation is cooperative, never forced
* nothing stops automatically
* streaming and long running handlers must watch the context
* net/http based frameworks behave consistently
* fasthttp based frameworks require different patterns

Once context propagation is clear, graceful shutdown, background work, and streaming APIs become much easier to reason about and much safer to implement.
