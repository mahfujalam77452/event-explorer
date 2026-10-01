package controllers

import (
	"fmt"
	"net/http"
	"testing"

	"event-explorer/models"
)

// fakeTimeout is a fake timeout error (net.Error interface means)।
type fakeTimeout struct{}

func (fakeTimeout) Error() string   { return "i/o timeout" }
func (fakeTimeout) Timeout() bool   { return true }
func (fakeTimeout) Temporary() bool { return true }

func TestDetailsErrorMapsToStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"invalid event id", models.ErrInvalidEventID, http.StatusNotFound},
		{"event not found", models.ErrEventNotFound, http.StatusNotFound},
		{"wrapped not found", fmt.Errorf("lookup: %w", models.ErrEventNotFound), http.StatusNotFound},
		{"timeout", fmt.Errorf("request failed: %w", fakeTimeout{}), http.StatusGatewayTimeout},
		{"anything else", fmt.Errorf("upstream returned status 500"), http.StatusBadGateway},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, message := detailsError(tc.err)
			if status != tc.wantStatus {
				t.Fatalf("want status %d, got %d", tc.wantStatus, status)
			}
			if message == "" {
				t.Fatal("the user-facing message must not be empty")
			}
		})
	}
}