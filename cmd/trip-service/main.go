package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Ksusha-Pushkova/trip_go/internal/config"
	"github.com/Ksusha-Pushkova/trip_go/internal/logger"
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

	appLogger := logger.New(cfg.LogLevel)

	pool, err := postgres.NewPool(ctx, cfg)
	if err != nil {
		appLogger.Error(
			"connect to database",
			"error", err,
		)
		os.Exit(1)
	}
	defer pool.Close()

	appLogger.Info("database connected")

	txManager := postgres.NewTxManager(
		pool,
		cfg.Database.QueryTimeout,
	)

	tripRepo := postgres.NewTripRepository(
		pool,
		cfg.Database.QueryTimeout,
	)

	tripUsecase := usecase.NewTripUsecase(
		tripRepo,
		txManager,
	)

	handlers := httptransport.NewHandlers(
		appLogger,
		tripUsecase,
		pool,
		cfg.Database.QueryTimeout,
	)

	router := httptransport.NewRouter(handlers)

	server := httptransport.NewServer(
		cfg.HTTP,
		router,
		appLogger,
	)

	if err := server.Run(
		ctx,
		cfg.ShutdownTimeout,
	); err != nil {
		appLogger.Error(
			"server run",
			"error", err,
		)
		os.Exit(1)
	}

	appLogger.Info("application stopped")
}
