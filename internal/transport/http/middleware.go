package httptransport

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/Ksusha-Pushkova/trip_go/internal/telemetry"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}

	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}

	return w.ResponseWriter.Write(body)
}

// TracingMiddleware adds OpenTelemetry tracing to HTTP requests.
func TracingMiddleware() func(http.Handler) http.Handler {
	tracer := otel.Tracer("trip-service/http")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span := tracer.Start(
				r.Context(),
				r.Method+" "+r.URL.Path,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					attribute.String("http.request.method", r.Method),
					attribute.String("url.path", r.URL.Path),
				),
			)
			defer span.End()

			recorder := &responseWriter{ResponseWriter: w}
			next.ServeHTTP(recorder, r.WithContext(ctx))

			status := recorder.status
			if status == 0 {
				status = http.StatusOK
			}

			route := chi.RouteContext(r.Context()).RoutePattern()
			if route == "" {
				route = r.URL.Path
			}

			span.SetName(r.Method + " " + route)
			span.SetAttributes(
				attribute.String("http.route", route),
				attribute.Int("http.response.status_code", status),
			)

			if status >= http.StatusInternalServerError {
				span.SetStatus(codes.Error, http.StatusText(status))
			}
		})
	}
}

// LoggingMiddleware logs HTTP request details and execution duration.
func LoggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			recorder := &responseWriter{ResponseWriter: w}

			next.ServeHTTP(recorder, r)

			status := recorder.status
			if status == 0 {
				status = http.StatusOK
			}

			route := chi.RouteContext(r.Context()).RoutePattern()
			if route == "" {
				route = r.URL.Path
			}

			spanContext := trace.SpanContextFromContext(r.Context())

			logger.Info(
				"http request",
				"method", r.Method,
				"route", route,
				"status", status,
				"duration_ms", time.Since(start).Milliseconds(),
				"trace_id", spanContext.TraceID().String(),
				"span_id", spanContext.SpanID().String(),
			)
		})
	}
}

// RecoverMiddleware recovers from panics in HTTP request handlers.
func RecoverMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.Error(
						"panic recovered",
						"panic", recovered,
						"stack", string(debug.Stack()),
						"method", r.Method,
						"path", r.URL.Path,
					)

					writeProblem(
						w,
						r,
						http.StatusInternalServerError,
						"internal_error",
						"Internal Server Error",
						"Internal server error",
					)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// MetricsMiddleware records HTTP request metrics.
func MetricsMiddleware(metrics *telemetry.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			metrics.RequestStarted(r.Context())

			recorder := &responseWriter{ResponseWriter: w}
			next.ServeHTTP(recorder, r)

			status := recorder.status
			if status == 0 {
				status = http.StatusOK
			}

			route := chi.RouteContext(r.Context()).RoutePattern()
			if route == "" {
				route = r.URL.Path
			}

			metrics.RequestFinished(
				r.Context(),
				r.Method,
				route,
				status,
				time.Since(start),
			)
		})
	}
}
