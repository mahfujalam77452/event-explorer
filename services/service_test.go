package services

import (
	"testing"
	"time"

	"event-explorer/utils"
)

func TestInit_CreatesBothServicesWithOneSharedCache(t *testing.T) {
	oldLocation, oldEvents := Location, Events
	t.Cleanup(func() { Location, Events = oldLocation, oldEvents })

	Init(&utils.Config{
		GoogleAPIKey:        "g",
		TicketmasterAPIKey:  "t",
		GoogleBaseURL:       "http://127.0.0.1:1",
		TicketmasterBaseURL: "http://127.0.0.1:1",
		HTTPTimeout:         time.Second,
		EventsPerCategory:   6,
	})

	if Location == nil || Events == nil {
		t.Fatal("Init must create both services")
	}
	if Events.cache == nil {
		t.Fatal("the event service needs a cache")
	}
	if Events.perCategory != 6 {
		t.Fatalf("want 6 events per category, got %d", Events.perCategory)
	}
}