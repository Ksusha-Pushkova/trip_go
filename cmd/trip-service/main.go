package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Ksusha-Pushkova/trip_go/internal/config"
	httptransport "github.com/Ksusha-Pushkova/trip_go/internal/transport/http"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}
	logger := newLogger(cfg.LogLevel)
	handlers := httptransport.NewHandlers(logger)
	router := httptransport.NewRouter(handlers)
	server := httptransport.NewServer(cfg.HTTPAddr, router, logger)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := server.Run(ctx, cfg.ShutdownTimeout); err != nil {
		logger.Error("server run", "error", err)
		os.Exit(1)
	}

	logger.Info("application stopped")
}

func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	return slog.New(handler)
}
