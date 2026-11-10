package httptransport

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestLoggingMiddleware(t *testing.T) {
	var logOutput bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&logOutput, nil))

	router := chi.NewRouter()
	router.Use(LoggingMiddleware(logger))

	router.Get("/trips/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("accepted"))
	})

	request := httptest.NewRequest(http.MethodGet, "/trips/123", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d",
			http.StatusAccepted, recorder.Code)
	}

	var logEntry map[string]any
	if err := json.Unmarshal(logOutput.Bytes(), &logEntry); err != nil {
		t.Fatalf("decode log output: %v", err)
	}

	expectedFields := map[string]any{
		"msg":    "http request",
		"method": http.MethodGet,
		"route":  "/trips/{id}",
		"status": float64(http.StatusAccepted),
	}

	for field, expected := range expectedFields {
		if logEntry[field] != expected {
			t.Errorf("expected %s=%v, got %v",
				field, expected, logEntry[field])
		}
	}

	if _, ok := logEntry["duration_ms"]; !ok {
		t.Error("expected duration_ms field in log")
	}
}

func TestRecoverMiddleware(t *testing.T) {
	var logOutput bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&logOutput, nil))

	router := chi.NewRouter()
	router.Use(RecoverMiddleware(logger))

	router.Get("/panic", func(_ http.ResponseWriter, _ *http.Request) {
		panic("test panic")
	})

	request := httptest.NewRequest(http.MethodGet, "/panic", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d",
			http.StatusInternalServerError, recorder.Code)
	}

	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/problem+json" {
		t.Errorf("expected application/problem+json, got %s", contentType)
	}

	var problem map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &problem); err != nil {
		t.Fatalf("decode problem response: %v", err)
	}

	if problem["code"] != "internal_error" {
		t.Errorf("expected internal_error, got %v", problem["code"])
	}

	if !strings.Contains(logOutput.String(), "test panic") {
		t.Error("expected panic details in log")
	}

	if !strings.Contains(logOutput.String(), "stack") {
		t.Error("expected stack trace in log")
	}
}
