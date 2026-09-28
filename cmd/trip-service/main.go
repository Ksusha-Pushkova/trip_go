package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Ksusha-Pushkova/trip_go/internal/config"
	"github.com/Ksusha-Pushkova/trip_go/internal/repository/postgres"
	httptransport "github.com/Ksusha-Pushkova/trip_go/internal/transport/http"
	"github.com/Ksusha-Pushkova/trip_go/internal/usecase"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	logger := newLogger(cfg.LogLevel)

	pool, err := postgres.NewPool(ctx, cfg)
	if err != nil {
		logger.Error("connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	logger.Info("database connected")

	txManager := postgres.NewTxManager(
		pool,
		cfg.DatabaseQueryTimeout,
	)

	tripRepo := postgres.NewTripRepository(
		pool,
		cfg.DatabaseQueryTimeout,
	)

	tripUsecase := usecase.NewTripUsecase(
		tripRepo,
		txManager,
	)

	handlers := httptransport.NewHandlers(
		logger,
		tripUsecase,
		pool,
	)

	router := httptransport.NewRouter(handlers)
	server := httptransport.NewServer(
		cfg.HTTPAddr,
		router,
		logger,
	)

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

	handler := slog.NewJSONHandler(
		os.Stdout,
		&slog.HandlerOptions{Level: lvl},
	)

	return slog.New(handler)
}
