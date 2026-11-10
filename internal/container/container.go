// Package container initializes and manages application dependencies.
package container

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Ksusha-Pushkova/trip_go/internal/config"
	"github.com/Ksusha-Pushkova/trip_go/internal/repository/postgres"
	"github.com/Ksusha-Pushkova/trip_go/internal/telemetry"
	httptransport "github.com/Ksusha-Pushkova/trip_go/internal/transport/http"
	createusecase "github.com/Ksusha-Pushkova/trip_go/internal/usecase/trip/create"
	finishusecase "github.com/Ksusha-Pushkova/trip_go/internal/usecase/trip/finish"
	getusecase "github.com/Ksusha-Pushkova/trip_go/internal/usecase/trip/get"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Container manages application dependencies and lifecycle.
type Container struct {
	pool   *pgxpool.Pool
	server *httptransport.Server
}

// New creates and initializes the application container.
func New(
	ctx context.Context,
	cfg config.Config,
	appLogger *slog.Logger,
) (*Container, error) {
	pool, err := postgres.NewPool(ctx, cfg)
	if err != nil {
		return nil, err
	}

	metrics, err := telemetry.NewMetrics()
	if err != nil {
		pool.Close()
		return nil, err
	}

	txManager := postgres.NewTxManager(
		pool,
		cfg.Database.QueryTimeout,
	)

	tripRepo := postgres.NewTripRepository(
		pool,
		cfg.Database.QueryTimeout,
	)

	activeTrips, err := tripRepo.CountActive(ctx)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("count active trips: %w", err)
	}

	metrics.SetActiveTrips(activeTrips)

	createUC := createusecase.New(tripRepo, txManager, metrics)
	getUC := getusecase.New(tripRepo)
	finishUC := finishusecase.New(tripRepo, txManager, metrics)

	handlers := httptransport.NewHandlers(
		appLogger,
		createUC,
		getUC,
		finishUC,
		pool,
		cfg.Database.QueryTimeout,
	)

	router := httptransport.NewRouter(handlers, appLogger, metrics)

	server := httptransport.NewServer(
		cfg.HTTP,
		router,
		appLogger,
	)

	return &Container{
		pool:   pool,
		server: server,
	}, nil
}

// Run starts the application services.
func (c *Container) Run(
	ctx context.Context,
	shutdownTimeout time.Duration,
) error {
	return c.server.Run(ctx, shutdownTimeout)
}

// Close releases application resources.
func (c *Container) Close() {
	if c.pool != nil {
		c.pool.Close()
	}
}
