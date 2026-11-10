package trip

import (
	"errors"
	"testing"
	"time"
)

func TestTrip_Finish(t *testing.T) {
	finishTime := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		initialStatus Status
		wantErr       error
		wantStatus    Status
		wantFinished  bool
	}{
		{
			name:          "active trip is finished successfully",
			initialStatus: StatusActive,
			wantStatus:    StatusCompleted,
			wantFinished:  true,
		},
		{
			name:          "completed trip cannot be finished again",
			initialStatus: StatusCompleted,
			wantErr:       ErrTripCompleted,
			wantStatus:    StatusCompleted,
		},
		{
			name:          "trip with unknown status cannot be finished",
			initialStatus: "unknown",
			wantStatus:    "unknown",
			wantFinished:  false,
			wantErr:       errors.New("unexpected trip status: unknown"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trip := &Trip{
				Status: tt.initialStatus,
			}

			err := trip.Finish(finishTime)

			if tt.wantErr != nil {
				if tt.wantErr == ErrTripCompleted {
					if !errors.Is(err, ErrTripCompleted) {
						t.Fatalf("expected ErrTripCompleted, got %v", err)
					}
				} else if err == nil {
					t.Fatal("expected an error, got nil")
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if trip.Status != tt.wantStatus {
				t.Errorf("expected status %q, got %q", tt.wantStatus, trip.Status)
			}

			if (trip.FinishedAt != nil) != tt.wantFinished {
				t.Errorf("expected FinishedAt presence to be %v", tt.wantFinished)
			}

			if tt.wantFinished {
				if !trip.FinishedAt.Equal(finishTime) {
					t.Errorf("expected FinishedAt %v, got %v", finishTime, trip.FinishedAt)
				}

				if !trip.UpdatedAt.Equal(finishTime) {
					t.Errorf("expected UpdatedAt %v, got %v", finishTime, trip.UpdatedAt)
				}
			}
		})
	}
}
