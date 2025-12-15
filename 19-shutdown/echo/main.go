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
