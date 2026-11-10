// Package main starts the trip service application.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Ksusha-Pushkova/trip_go/internal/config"
	"github.com/Ksusha-Pushkova/trip_go/internal/container"
	"github.com/Ksusha-Pushkova/trip_go/internal/logger"
	"github.com/Ksusha-Pushkova/trip_go/internal/telemetry"
)

func main() {
	if err := run(); err != nil {
		slog.Error("application failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	appLogger := logger.New(cfg.LogLevel)

	telemetryProvider, err := telemetry.New(
		ctx,
		cfg.OTLPEndpoint,
		cfg.ServiceName,
	)
	if err != nil {
		return fmt.Errorf("initialize telemetry: %w", err)
	}

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			cfg.ShutdownTimeout,
		)
		defer cancel()

		if err := telemetryProvider.Shutdown(shutdownCtx); err != nil {
			appLogger.Error("shutdown telemetry", "error", err)
		}
	}()

	app, err := container.New(ctx, cfg, appLogger)
	if err != nil {
		return fmt.Errorf("initialize application: %w", err)
	}
	defer app.Close()

	appLogger.Info("application started")

	if err := app.Run(ctx, cfg.ShutdownTimeout); err != nil {
		return fmt.Errorf("server run: %w", err)
	}

	return nil
}
