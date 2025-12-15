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
