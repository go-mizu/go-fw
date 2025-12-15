# Forms, multipart data, and file uploads

Forms and file uploads exercise parts of HTTP that JSON APIs often bypass. They combine multiple concerns in a single request: structured fields, streamed bodies, temporary storage, size limits, and cleanup rules. They also introduce real risk if handled casually, because a single request can allocate large amounts of memory or disk space.

At this level, frameworks must answer several concrete questions:

* when the request body is parsed
* where parsed form fields live
* where uploaded files are stored
* who is responsible for cleanup
* how size limits are enforced
* what happens if parsing fails halfway through

The examples below implement two endpoints:

* `POST /login` using form fields
* `POST /upload` using `multipart/form-data` with a file field named `file`

The focus is not convenience, but understanding data ownership and lifecycle.

## net/http

```go
package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /login", login)
	mux.HandleFunc("POST /upload", upload)

	http.ListenAndServe(":8080", mux)
}

func login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}

	user := r.FormValue("user")
	pass := r.FormValue("pass")

	fmt.Fprintf(w, "user=%s pass=%s\n", user, pass)
}

func upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "bad multipart", http.StatusBadRequest)
		return
	}
	defer r.MultipartForm.RemoveAll()

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file missing", http.StatusBadRequest)
		return
	}
	defer file.Close()

	out, err := os.Create("./" + header.Filename)
	if err != nil {
		http.Error(w, "cannot save file", http.StatusInternalServerError)
		return
	}
	defer out.Close()

	io.Copy(out, file)

	fmt.Fprintf(w, "uploaded %s\n", header.Filename)
}
```

### How form and upload handling works

In net/http, form parsing is explicit and destructive. Calling `ParseForm` or `ParseMultipartForm` consumes the request body and populates internal data structures on the request.

For multipart requests, the standard library automatically spills large parts to disk once a memory threshold is exceeded. The `maxMemory` argument controls how much data is kept in memory before temporary files are created.

Temporary files are not cleaned up automatically. Calling `RemoveAll` on `r.MultipartForm` is required to avoid leaking disk space.

The handler owns everything: limits, validation, storage location, and cleanup. This provides maximum control, but also means every mistake is yours.

## Chi

```go
package main

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()

	r.Post("/login", login)
	r.Post("/upload", upload)

	http.ListenAndServe(":8080", r)
}

func login(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	fmt.Fprintf(w, "user=%s\n", r.FormValue("user"))
}

func upload(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20)
	defer r.MultipartForm.RemoveAll()

	file, header, _ := r.FormFile("file")
	defer file.Close()

	out, _ := os.Create(header.Filename)
	defer out.Close()

	io.Copy(out, file)

	fmt.Fprintf(w, "uploaded %s\n", header.Filename)
}
```

### How form and upload handling works

Chi does not modify form or multipart semantics. Everything behaves exactly as in net/http.

This means:

* the body is consumed when parsing functions are called
* temporary files may be created automatically
* cleanup remains the handler’s responsibility

Chi adds routing structure, but form handling remains a standard library concern.

## Gin

```go
package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.New()

	r.POST("/login", func(c *gin.Context) {
		user := c.PostForm("user")
		pass := c.PostForm("pass")

		c.String(http.StatusOK, "user=%s pass=%s", user, pass)
	})

	r.POST("/upload", func(c *gin.Context) {
		file, err := c.FormFile("file")
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		c.SaveUploadedFile(file, "./"+file.Filename)
		c.String(http.StatusOK, "uploaded %s", file.Filename)
	})

	r.Run(":8080")
}
```

### How form and upload handling works

Gin parses form and multipart data lazily. Parsing happens when helpers such as `PostForm` or `FormFile` are first called.

Uploaded files are represented by headers and temporary storage managed by the framework. `SaveUploadedFile` copies the uploaded content to a destination path and abstracts away file handling details.

This reduces boilerplate, but it also hides:

* where temporary files are stored
* when cleanup happens
* what size limits are enforced

Large uploads still require configuring limits at the server or reverse proxy level.

## Echo

```go
package main

import (
	"io"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()

	e.POST("/login", func(c echo.Context) error {
		user := c.FormValue("user")
		pass := c.FormValue("pass")

		return c.String(http.StatusOK, "user="+user+" pass="+pass)
	})

	e.POST("/upload", func(c echo.Context) error {
		file, err := c.FormFile("file")
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}

		src, err := file.Open()
		if err != nil {
			return err
		}
		defer src.Close()

		dst, err := os.Create(file.Filename)
		if err != nil {
			return err
		}
		defer dst.Close()

		io.Copy(dst, src)

		return c.String(http.StatusOK, "uploaded "+file.Filename)
	})

	e.Start(":8080")
}
```

### How form and upload handling works

Echo exposes form values through context helpers, but file handling remains explicit.

Multipart parsing and temporary file storage are handled by the underlying net/http layer. Echo focuses on control flow and error propagation rather than storage abstractions.

Errors returned from handlers propagate into centralized error handling, keeping failure paths consistent.

## Fiber

```go
package main

import (
	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	app.Post("/login", func(c *fiber.Ctx) error {
		user := c.FormValue("user")
		pass := c.FormValue("pass")

		return c.SendString("user=" + user + " pass=" + pass)
	})

	app.Post("/upload", func(c *fiber.Ctx) error {
		file, err := c.FormFile("file")
		if err != nil {
			return c.Status(400).SendString("file missing")
		}

		c.SaveFile(file, "./"+file.Filename)
		return c.SendString("uploaded " + file.Filename)
	})

	app.Listen(":8080")
}
```

### How form and upload handling works

Fiber parses multipart data using fasthttp primitives.

Uploaded files may be stored in memory or temporary buffers depending on size and configuration. Helpers such as `SaveFile` abstract copying and cleanup.

Because Fiber contexts are pooled and reused, all file handling must complete within the handler. References to request data must not escape the request scope.

## Mizu

```go
package main

import (
	"io"
	"net/http"
	"os"

	"github.com/go-mizu/mizu"
)

func main() {
	app := mizu.New()

	app.Post("/login", func(c *mizu.Ctx) error {
		user := c.Form("user")
		pass := c.Form("pass")

		return c.Text(http.StatusOK, "user="+user+" pass="+pass)
	})

	app.Post("/upload", func(c *mizu.Ctx) error {
		file, header, err := c.FormFile("file")
		if err != nil {
			return c.Text(http.StatusBadRequest, "file missing")
		}
		defer file.Close()

		out, err := os.Create(header.Filename)
		if err != nil {
			return err
		}
		defer out.Close()

		io.Copy(out, file)

		return c.Text(http.StatusOK, "uploaded "+header.Filename)
	})

	app.Listen(":8080")
}
```

### How form and upload handling works

Mizu exposes form access explicitly through the request context while keeping file handling close to net/http semantics.

Parsing consumes the body, uploaded files may involve temporary storage, and cleanup remains explicit. Errors propagate through the same error handling path as other failures.

This keeps upload behavior predictable and consistent with the rest of the request lifecycle.

## What to focus on

Forms and uploads expose hidden defaults that matter in production.

Key questions to keep in mind:

* where uploaded files are stored
* when temporary files are deleted
* what size limits are enforced
* what happens on partial reads or client disconnects

Framework helpers reduce boilerplate, but they also hide answers to these questions. Understanding the underlying model prevents subtle memory, disk, and security issues later.
