package httptransport

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Ksusha-Pushkova/trip_go/internal/config"
)

type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
}

func NewServer(
	cfg config.HTTPConfig,
	handler http.Handler,
	logger *slog.Logger,
) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:              cfg.Addr,
			Handler:           handler,
			ReadHeaderTimeout: cfg.ReadHeaderTimeout,
			ReadTimeout:       cfg.ReadTimeout,
			WriteTimeout:      cfg.WriteTimeout,
			IdleTimeout:       cfg.IdleTimeout,
		},
		logger: logger,
	}
}

func (s *Server) Run(
	ctx context.Context,
	shutdownTimeout time.Duration,
) error {
	errCh := make(chan error, 1)

	go func() {
		s.logger.Info(
			"http server starting",
			"addr",
			s.httpServer.Addr,
		)

		if err := s.httpServer.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err

	case <-ctx.Done():
		s.logger.Info("http server shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		shutdownTimeout,
	)
	defer cancel()

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		s.logger.Error(
			"http server shutdown error",
			"error",
			err,
		)

		return err
	}

	s.logger.Info("http server stopped")

	return nil
}
