# WebSockets and bidirectional connections

WebSockets replace the short lived request response pattern with a long lived connection. After a successful upgrade, HTTP semantics stop applying. There are no headers to change, no status codes to return, and no new requests arriving. Instead, a single TCP connection stays open and both sides can send data at any time.

Understanding WebSockets requires separating two phases clearly.

The first phase is ordinary HTTP. A client sends a request with upgrade headers. The server decides whether to accept it. Middleware, routing, authentication, and rate limits all apply here.

The second phase starts after the upgrade succeeds. At that point the HTTP framework steps aside. You now own a bidirectional stream and are responsible for reads, writes, errors, and shutdown.

This section examines how each framework crosses that boundary and what it means for control flow and lifecycle.

The example is identical everywhere.

* a client connects to `/ws`
* the server echoes every received message back to the client

## net/http

```go
package main

import (
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			mt, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			conn.WriteMessage(mt, msg)
		}
	})

	http.ListenAndServe(":8080", mux)
}
```

### How the connection is established

The standard library does not implement WebSockets. A third party package such as `gorilla/websocket` performs the upgrade and manages the protocol.

The handler begins as a normal HTTP handler. Calling `Upgrade` switches the connection from HTTP to WebSocket. From that moment on, several rules apply.

The response writer must not be used again. Headers and status codes are irrelevant. The request object no longer participates in execution.

The WebSocket connection exposes blocking read and write calls. A loop reads frames, processes them, and writes responses.

When a read or write fails, the connection is usually finished. Cleanup happens by closing the connection and returning from the handler.

Ownership of the connection is explicit and absolute. The handler controls lifetime, concurrency, and shutdown.

## Chi

```go
package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func main() {
	r := chi.NewRouter()

	r.Get("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			mt, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			conn.WriteMessage(mt, msg)
		}
	})

	http.ListenAndServe(":8080", r)
}
```

### How the connection is established

Chi does not change the WebSocket model at all.

Routing and middleware apply before the upgrade. After `Upgrade` succeeds, Chi disappears from the execution path.

This highlights a general rule for long lived connections. Once the handshake is complete, most HTTP frameworks are no longer involved. They only decide whether the connection is allowed to exist.

## Gin

```go
package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func main() {
	r := gin.New()

	r.GET("/ws", func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			mt, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			conn.WriteMessage(mt, msg)
		}
	})

	r.Run(":8080")
}
```

### How the connection is established

Gin exposes the raw `http.ResponseWriter` and `*http.Request` through its context. This makes the upgrade step straightforward.

Middleware and handlers run before the upgrade. Authentication, logging, and limits apply there.

After the upgrade, Gin context helpers, abort logic, and error handling no longer matter. The WebSocket loop runs independently of the framework.

One important detail is that aborting a context after the upgrade has no effect. The connection already exists.

## Echo

```go
package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func main() {
	e := echo.New()

	e.GET("/ws", func(c echo.Context) error {
		conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}
		defer conn.Close()

		for {
			mt, msg, err := conn.ReadMessage()
			if err != nil {
				return nil
			}
			conn.WriteMessage(mt, msg)
		}
	})

	e.Start(":8080")
}
```

### How the connection is established

Echo follows the same boundary as net/http.

The handler participates in HTTP execution until the upgrade completes. After that, the returned error value is irrelevant to the WebSocket loop.

Returning `nil` or an error only affects the handshake phase. Once the connection is upgraded, control flow is entirely manual.

Echo does not buffer or manage messages. The WebSocket library owns framing and protocol behavior.

## Fiber

```go
package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

func main() {
	app := fiber.New()

	app.Get("/ws", websocket.New(func(c *websocket.Conn) {
		for {
			mt, msg, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			c.WriteMessage(mt, msg)
		}
	}))

	app.Listen(":8080")
}
```

### How the connection is established

Fiber uses a WebSocket implementation designed for fasthttp.

The HTTP upgrade is handled by Fiber before your handler runs. The handler receives a WebSocket connection directly.

This removes the HTTP phase from the handler entirely. There is no request object and no response writer.

The result is a simpler mental model once inside the handler, but fewer hooks for inspecting or modifying the handshake.

Execution inside the loop matches other frameworks. Reads and writes block, and errors end the connection.

## Mizu

```go
package main

import (
	"net/http"

	"github.com/go-mizu/mizu"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func main() {
	app := mizu.New()

	app.Get("/ws", func(c *mizu.Ctx) error {
		conn, err := upgrader.Upgrade(c.Writer(), c.Request(), nil)
		if err != nil {
			return err
		}
		defer conn.Close()

		for {
			mt, msg, err := conn.ReadMessage()
			if err != nil {
				return nil
			}
			conn.WriteMessage(mt, msg)
		}
	})

	app.Listen(":8080")
}
```

### How the connection is established

Mizu follows the same model as net/http and Echo.

The handler begins in HTTP mode. Middleware and routing apply normally. Calling `Upgrade` transfers ownership of the connection to the WebSocket layer.

After the upgrade, the request context no longer influences execution. Errors returned before the upgrade follow normal error handling. Errors after the upgrade only affect the connection.

This keeps the boundary explicit and avoids hidden behavior.

## What to focus on

WebSockets behave the same at their core, regardless of framework.

Key ideas to internalize:

* the framework matters only until the upgrade
* after the upgrade, you own the connection
* reads and writes block
* one connection usually maps to one goroutine

Meaningful differences appear in:

* whether raw HTTP objects are exposed
* whether WebSocket support is built in or external
* how middleware participates in the handshake

Once this boundary is clear, WebSocket code becomes predictable and portable across frameworks.
