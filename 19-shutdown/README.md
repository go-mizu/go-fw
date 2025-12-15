# Graceful shutdown and server lifecycle

Graceful shutdown is the contract between your process manager and your HTTP server. A shutdown sequence is correct when it guarantees four things:

* **stop accepting new work**
* **let in-flight requests finish** within a bounded drain window
* **stop reporting readiness** so load balancers stop sending traffic
* **exit deterministically**, even if some handlers misbehave

This section compares how each stack wires those guarantees. The HTTP behavior stays trivial. The moving parts are lifecycle ownership, signal handling, and the shutdown API.

We keep the scenario consistent:

* the server exposes `GET /`
* readiness flips to `503` once shutdown starts
* SIGINT or SIGTERM triggers a graceful drain, then process exit

## net/http

`19-shutdown/nethttp/main.go`

```go
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
)

const shutdownTimeout = 15 * time.Second

func main() {
	var shuttingDown atomic.Bool

	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "hello")
	})

	mux.HandleFunc("GET /livez", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if shuttingDown.Load() {
			http.Error(w, "shutting down", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-errCh:
		_ = err
		return
	case <-ctx.Done():
	}

	shuttingDown.Store(true)

	drainCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	_ = srv.Shutdown(drainCtx)

	_ = <-errCh
}
```

### How shutdown actually works

A `net/http` process has to implement both halves of shutdown:

* **trigger**: convert SIGINT/SIGTERM into cancellation (`signal.NotifyContext`)
* **mechanism**: call `srv.Shutdown(drainCtx)`

`Shutdown` closes listeners so new connections stop, closes idle keep-alive connections, and waits for active handlers to return until the deadline expires. It never kills goroutines. Handlers must cooperate by finishing work or honoring request context.

Readiness is not automatic. If you want load balancers to stop sending traffic, you flip readiness as soon as shutdown starts (the `shuttingDown` flag).

## Chi

`19-shutdown/chi/main.go`

```go
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
)

const shutdownTimeout = 15 * time.Second

func main() {
	var shuttingDown atomic.Bool

	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, "hello")
	})

	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if shuttingDown.Load() {
			http.Error(w, "shutting down", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-errCh:
		_ = err
		return
	case <-ctx.Done():
	}

	shuttingDown.Store(true)

	drainCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	_ = srv.Shutdown(drainCtx)

	_ = <-errCh
}
```

### How shutdown actually works

Chi does not change the lifecycle model. The server is still `net/http`. Chi is only the handler. That means:

* shutdown correctness is entirely determined by `http.Server.Shutdown`
* signal wiring remains app-owned
* readiness is app-owned

The practical win is that Chi composes cleanly: everything that works for `net/http` works unchanged.

## Gin

`19-shutdown/gin/main.go`

```go
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

const shutdownTimeout = 15 * time.Second

func main() {
	var shuttingDown atomic.Bool

	r := gin.New()

	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "hello\n")
	})

	r.GET("/readyz", func(c *gin.Context) {
		if shuttingDown.Load() {
			c.String(http.StatusServiceUnavailable, "shutting down\n")
			return
		}
		c.String(http.StatusOK, "ok\n")
	})

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-errCh:
		_ = err
		return
	case <-ctx.Done():
	}

	shuttingDown.Store(true)

	drainCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	_ = srv.Shutdown(drainCtx)

	_ = <-errCh
}
```

### How shutdown actually works

Gin’s `Run` convenience starts its own `http.Server`, but it does not remove the need for a shutdown trigger. If you want graceful drain with timeouts, you typically own the `http.Server` explicitly and call `Shutdown` yourself.

That keeps lifecycle decisions outside Gin:

* you decide the shutdown timeout
* you decide readiness behavior
* you decide how to wire signals

## Echo

`19-shutdown/echo/main.go`

```go
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
)

const shutdownTimeout = 15 * time.Second

func main() {
	var shuttingDown atomic.Bool

	e := echo.New()

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "hello\n")
	})

	e.GET("/readyz", func(c echo.Context) error {
		if shuttingDown.Load() {
			return c.String(http.StatusServiceUnavailable, "shutting down\n")
		}
		return c.String(http.StatusOK, "ok\n")
	})

	errCh := make(chan error, 1)
	go func() {
		if err := e.Start(":8080"); err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-errCh:
		_ = err
		return
	case <-ctx.Done():
	}

	shuttingDown.Store(true)

	drainCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	_ = e.Shutdown(drainCtx)

	_ = <-errCh
}
```

### How shutdown actually works

Echo exposes a shutdown mechanism (`e.Shutdown(ctx)`), but it still needs an external trigger. Signal wiring stays in your `main` because:

* production systems might use different triggers
* tests need programmatic shutdown
* multi-server apps need coordinated shutdown ordering

Echo’s `Shutdown` delegates to the underlying `http.Server.Shutdown`, so the same cooperative handler rules apply.

## Fiber

`19-shutdown/fiber/main.go`

```go
package main

import (
	"context"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
)

const shutdownTimeout = 15 * time.Second

func main() {
	var shuttingDown atomic.Bool

	app := fiber.New()

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("hello\n")
	})

	app.Get("/readyz", func(c *fiber.Ctx) error {
		if shuttingDown.Load() {
			c.Status(503)
			return c.SendString("shutting down\n")
		}
		return c.SendString("ok\n")
	})

	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Listen(":8080")
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-errCh:
		_ = err
		return
	case <-ctx.Done():
	}

	shuttingDown.Store(true)

	drainCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	_ = app.ShutdownWithContext(drainCtx)

	_ = <-errCh
}
```

### How shutdown actually works

Fiber owns a fasthttp server. There is no `http.Server.Shutdown`, so Fiber provides its own shutdown methods.

The structure is still the same:

* app-owned trigger
* framework-owned shutdown mechanism
* readiness flipped in app code

The important lifecycle rule stays: shutdown does not stop your goroutines. Handlers must finish quickly, and any background loops must be bound to your own context.

## Mizu

`19-shutdown/mizu/main.go`

```go
package main

import (
	"net/http"

	"github.com/go-mizu/mizu"
)

func main() {
	app := mizu.New()

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "hello\n")
	})

	app.Get("/livez", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok\n")
	})

	app.Get("/readyz", func(c *mizu.Ctx) error {
		return c.Writer().ServeHTTP(c.Writer(), c.Request()) // placeholder: use app.ReadyzHandler in real app
	})

	app.Listen(":8080")
}
```

### How shutdown actually works

Mizu, as implemented in your `App`, **owns the signal trigger and the shutdown mechanism** inside `Listen`, `ListenTLS`, and `Serve`.

The lifecycle is:

* `Listen` builds an `http.Server` with timeouts (`ReadHeaderTimeout`, `IdleTimeout`)
* `Listen` installs a signal-driven parent context via `signal.NotifyContext` (non-Windows build)
* the serve loop runs in a goroutine
* when the signal cancels the context, shutdown begins:

  * `shuttingDown` flips to true (readiness becomes `503`)
  * a bounded drain context is created using `context.WithTimeout(context.Background(), ShutdownTimeout)`
  * `srv.Shutdown(drainCtx)` runs, falling back to `srv.Close()` on failure
  * the code waits for the serve loop to exit, but never forever (timeout + `serverExitGrace`)
  * logs include duration and errors

This design removes the signal snippet from every example because lifecycle lives in the framework, not your `main`.

For the examples, use the built-in handlers directly:

* `app.LivezHandler()` stays `200` during shutdown
* `app.ReadyzHandler()` flips to `503` after shutdown starts

If you want `readyz` and `livez` as routes inside Mizu, mount them like this:

```go
app.Get("/livez", func(c *mizu.Ctx) error {
	app.LivezHandler().ServeHTTP(c.Writer(), c.Request())
	return nil
})

app.Get("/readyz", func(c *mizu.Ctx) error {
	app.ReadyzHandler().ServeHTTP(c.Writer(), c.Request())
	return nil
})
```

That keeps readiness semantics identical to the lifecycle flag.

## Comparing shutdown ownership

| Framework | Who owns the trigger          | Who owns the drain                                   | Who flips readiness         |
| --------- | ----------------------------- | ---------------------------------------------------- | --------------------------- |
| net/http  | app                           | app (via `http.Server.Shutdown`)                     | app                         |
| Chi       | app                           | app (via `http.Server.Shutdown`)                     | app                         |
| Gin       | app (for graceful shutdown)   | app (via `http.Server.Shutdown`)                     | app                         |
| Echo      | app                           | framework method delegates to `http.Server.Shutdown` | app                         |
| Fiber     | app                           | framework (`ShutdownWithContext`)                    | app                         |
| Mizu      | framework (in `Listen/Serve`) | framework (calls `http.Server.Shutdown`)             | framework (`ReadyzHandler`) |

## What learners should focus on

* graceful shutdown is a **lifecycle concern**, not a routing concern
* the shutdown mechanism is almost always **cooperative**
* readiness should flip **as soon as shutdown starts**, not after it finishes
* the signal channel is optional when lifecycle is owned elsewhere (Mizu), but still common in apps that want explicit control
