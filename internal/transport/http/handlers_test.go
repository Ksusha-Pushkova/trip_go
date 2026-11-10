package httptransport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Ksusha-Pushkova/trip_go/api"
	tripdomain "github.com/Ksusha-Pushkova/trip_go/internal/domain/trip"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestCreateTrip_Success(t *testing.T) {
	ctrl := gomock.NewController(t)

	createUC := NewMockcreateUseCase(ctrl)
	getUC := NewMockgetUseCase(ctrl)
	finishUC := NewMockfinishUseCase(ctrl)

	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	createUC.EXPECT().
		Execute(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, trip tripdomain.Trip) error {
			if trip.Status != tripdomain.StatusActive {
				t.Errorf("expected active status, got %s", trip.Status)
			}
			return nil
		})

	h := NewHandlers(
		logger,
		createUC,
		getUC,
		finishUC,
		nil,
		0,
	)

	userID := tripdomain.Trip{}.ID
	driverID := tripdomain.Trip{}.ID

	userID[0] = 1
	driverID[0] = 2

	body := api.TripData{
		UserId:   userID,
		DriverId: driverID,
		StartPoint: api.Coordinates{
			Latitude:  55.75,
			Longitude: 37.61,
		},
		EndPoint: api.Coordinates{
			Latitude:  55.76,
			Longitude: 37.62,
		},
		Price: 500,
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/trips",
		bytes.NewReader(bodyJSON),
	)
	rec := httptest.NewRecorder()

	h.CreateTrip(rec, req, api.CreateTripParams{})

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	if rec.Header().Get("Location") == "" {
		t.Fatal("expected Location header")
	}
}

func TestCreateTrip_InvalidBody(t *testing.T) {
	ctrl := gomock.NewController(t)

	createUC := NewMockcreateUseCase(ctrl)
	getUC := NewMockgetUseCase(ctrl)
	finishUC := NewMockfinishUseCase(ctrl)

	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	h := NewHandlers(
		logger,
		createUC,
		getUC,
		finishUC,
		nil,
		0,
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/trips",
		bytes.NewBufferString(`{"user_id":`),
	)
	rec := httptest.NewRecorder()

	h.CreateTrip(rec, req, api.CreateTripParams{})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateTrip_DriverBusy(t *testing.T) {
	ctrl := gomock.NewController(t)

	createUC := NewMockcreateUseCase(ctrl)
	getUC := NewMockgetUseCase(ctrl)
	finishUC := NewMockfinishUseCase(ctrl)

	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	createUC.EXPECT().
		Execute(gomock.Any(), gomock.Any()).
		Return(tripdomain.ErrDriverBusy)

	h := NewHandlers(logger, createUC, getUC, finishUC, nil, 0)

	userID := tripdomain.Trip{}.ID
	driverID := tripdomain.Trip{}.ID
	userID[0] = 1
	driverID[0] = 2

	body := api.TripData{
		UserId:   userID,
		DriverId: driverID,
		StartPoint: api.Coordinates{
			Latitude:  55.75,
			Longitude: 37.61,
		},
		EndPoint: api.Coordinates{
			Latitude:  55.76,
			Longitude: 37.62,
		},
		Price: 500,
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", bytes.NewReader(bodyJSON))
	rec := httptest.NewRecorder()

	h.CreateTrip(rec, req, api.CreateTripParams{})

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetTrip_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)

	createUC := NewMockcreateUseCase(ctrl)
	getUC := NewMockgetUseCase(ctrl)
	finishUC := NewMockfinishUseCase(ctrl)

	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	getUC.EXPECT().
		Execute(gomock.Any(), gomock.Any()).
		Return(nil, tripdomain.ErrTripNotFound)

	h := NewHandlers(logger, createUC, getUC, finishUC, nil, 0)

	tripID := api.TripId{}
	tripID[0] = 1

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/trips/test-id",
		nil,
	)
	rec := httptest.NewRecorder()

	h.GetTrip(rec, req, tripID)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
	}

	if contentType := rec.Header().Get("Content-Type"); contentType != "application/problem+json" {
		t.Errorf("expected problem+json, got %s", contentType)
	}
}

func TestGetTrip_Success(t *testing.T) {
	ctrl := gomock.NewController(t)

	createUC := NewMockcreateUseCase(ctrl)
	getUC := NewMockgetUseCase(ctrl)
	finishUC := NewMockfinishUseCase(ctrl)

	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	tripID := uuid.New()
	userID := uuid.New()
	driverID := uuid.New()

	expectedTrip := &tripdomain.Trip{
		ID:       tripID,
		UserID:   userID,
		DriverID: driverID,
		Start: tripdomain.Destination{
			Latitude:  55.75,
			Longitude: 37.61,
		},
		End: tripdomain.Destination{
			Latitude:  55.76,
			Longitude: 37.62,
		},
		Price:     500,
		Status:    tripdomain.StatusActive,
		StartedAt: time.Now().UTC(),
	}

	getUC.EXPECT().
		Execute(gomock.Any(), tripID).
		Return(expectedTrip, nil)

	h := NewHandlers(logger, createUC, getUC, finishUC, nil, 0)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/trips/"+tripID.String(),
		nil,
	)
	rec := httptest.NewRecorder()

	h.GetTrip(rec, req, tripID)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response api.Trip
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Id != tripID {
		t.Errorf("expected trip ID %s, got %s", tripID, response.Id)
	}

	if response.Status != api.TripStatus(tripdomain.StatusActive) {
		t.Errorf("expected active status, got %s", response.Status)
	}

	if response.Price != 500 {
		t.Errorf("expected price 500, got %d", response.Price)
	}
}

func TestFinishTrip_Success(t *testing.T) {
	ctrl := gomock.NewController(t)

	createUC := NewMockcreateUseCase(ctrl)
	getUC := NewMockgetUseCase(ctrl)
	finishUC := NewMockfinishUseCase(ctrl)

	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	tripID := uuid.New()
	expectedTrip := &tripdomain.Trip{
		ID:     tripID,
		Status: tripdomain.StatusCompleted,
		Price:  500,
	}

	finishUC.EXPECT().
		Execute(gomock.Any(), tripID).
		Return(expectedTrip, nil)

	h := NewHandlers(logger, createUC, getUC, finishUC, nil, 0)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/trips/"+tripID.String()+"/finish",
		nil,
	)
	rec := httptest.NewRecorder()

	h.FinishTrip(rec, req, tripID)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response api.Trip
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Status != api.TripStatus(tripdomain.StatusCompleted) {
		t.Errorf("expected completed status, got %s", response.Status)
	}
}

func TestFinishTrip_AlreadyCompleted(t *testing.T) {
	ctrl := gomock.NewController(t)

	createUC := NewMockcreateUseCase(ctrl)
	getUC := NewMockgetUseCase(ctrl)
	finishUC := NewMockfinishUseCase(ctrl)

	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	finishUC.EXPECT().
		Execute(gomock.Any(), gomock.Any()).
		Return(nil, tripdomain.ErrTripCompleted)

	h := NewHandlers(logger, createUC, getUC, finishUC, nil, 0)

	tripID := uuid.New()

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/trips/"+tripID.String()+"/finish",
		nil,
	)
	rec := httptest.NewRecorder()

	h.FinishTrip(rec, req, tripID)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d: %s", rec.Code, rec.Body.String())
	}

	if contentType := rec.Header().Get("Content-Type"); contentType != "application/problem+json" {
		t.Errorf("expected problem+json, got %s", contentType)
	}
}

func TestGetTrip_InternalError(t *testing.T) {
	ctrl := gomock.NewController(t)

	createUC := NewMockcreateUseCase(ctrl)
	getUC := NewMockgetUseCase(ctrl)
	finishUC := NewMockfinishUseCase(ctrl)

	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	getUC.EXPECT().
		Execute(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("database password leaked"))

	h := NewHandlers(logger, createUC, getUC, finishUC, nil, 0)

	tripID := uuid.New()

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/trips/"+tripID.String(),
		nil,
	)
	rec := httptest.NewRecorder()

	h.GetTrip(rec, req, tripID)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d: %s", rec.Code, rec.Body.String())
	}

	if strings.Contains(rec.Body.String(), "database password leaked") {
		t.Error("internal error details must not be exposed")
	}
}

func TestReady_Success(t *testing.T) {
	ctrl := gomock.NewController(t)

	readiness := NewMockreadinessChecker(ctrl)
	readiness.EXPECT().
		Ping(gomock.Any()).
		Return(nil)

	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	h := NewHandlers(
		logger,
		NewMockcreateUseCase(ctrl),
		NewMockgetUseCase(ctrl),
		NewMockfinishUseCase(ctrl),
		readiness,
		time.Second,
	)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()

	h.Ready(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func TestReady_DatabaseUnavailable(t *testing.T) {
	ctrl := gomock.NewController(t)

	readiness := NewMockreadinessChecker(ctrl)
	readiness.EXPECT().
		Ping(gomock.Any()).
		Return(errors.New("database unavailable"))

	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	h := NewHandlers(
		logger,
		NewMockcreateUseCase(ctrl),
		NewMockgetUseCase(ctrl),
		NewMockfinishUseCase(ctrl),
		readiness,
		time.Second,
	)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()

	h.Ready(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rec.Code)
	}
}
