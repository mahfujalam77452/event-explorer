package utils

import (
	"reflect"
	"strings"
	"testing"
)

func TestLoadConfig_MissingKeysAreReported(t *testing.T) {
	t.Setenv("GOOGLE_API_KEY", "")
	t.Setenv("TICKETMASTER_API_KEY", "")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected an error when api keys are missing")
	}
	for _, name := range []string{"GOOGLE_API_KEY", "TICKETMASTER_API_KEY"} {
		if !strings.Contains(err.Error(), name) {
			t.Fatalf("error should mention %s, got: %v", name, err)
		}
	}
}

func TestLoadConfig_ReadsKeysFromEnvironment(t *testing.T) {
	t.Setenv("GOOGLE_API_KEY", "  g-key  ")
	t.Setenv("TICKETMASTER_API_KEY", "tm-key")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.GoogleAPIKey != "g-key" {
		t.Fatalf("key should be trimmed, got %q", cfg.GoogleAPIKey)
	}
	if cfg.TicketmasterAPIKey != "tm-key" {
		t.Fatalf("unexpected ticketmaster key %q", cfg.TicketmasterAPIKey)
	}
	if cfg.EventsPerCategory != 6 {
		t.Fatalf("events per category should be 6, got %d", cfg.EventsPerCategory)
	}
	if cfg.HTTPTimeout <= 0 {
		t.Fatal("timeout must be positive")
	}
}

func TestSplitCSV(t *testing.T) {
	got := splitCSV(" Ticketmaster.com, livenation.COM,,  ,ticketweb.com ")
	want := []string{"ticketmaster.com", "livenation.com", "ticketweb.com"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if len(splitCSV("")) != 0 {
		t.Fatal("empty input should give an empty list")
	}
}