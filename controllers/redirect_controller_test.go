package controllers_test

import (
	"strings"
	"testing"
)

func TestRedirect_Success(t *testing.T) {
	resetFake(t)
	rec := do("GET", "/redirect/ev1")

	if rec.Code != 302 {
		t.Fatalf("want 302, got %d", rec.Code)
	}

	if got := rec.Header().Get("Location"); got != "https://www.ticketmaster.com/event/ev1" {
		t.Fatalf("unexpected Location %q", got)
	}

	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("redirect must not be cached, got %q", got)
	}
}

// A visitor-supplied destination must never be accepted.
func TestRedirect_IgnoresVisitorSuppliedDestination(t *testing.T) {
	resetFake(t)

	rec := do(
		"GET",
		"/redirect/ev1?url=https://evil.com&redirect=//evil.com&next=https://evil.com&to=https://evil.com",
	)

	if rec.Code != 302 {
		t.Fatalf("want 302, got %d", rec.Code)
	}

	if got := rec.Header().Get("Location"); got != "https://www.ticketmaster.com/event/ev1" {
		t.Fatalf("a visitor-supplied destination changed the redirect: %q", got)
	}
}

func TestRedirect_RefusesUnapprovedHostAndMissingUrl(t *testing.T) {
	for _, id := range []string{"evil1", "nourl1"} {
		t.Run(id, func(t *testing.T) {
			resetFake(t)
			rec := do("GET", "/redirect/"+id)

			if rec.Code != 502 {
				t.Fatalf("want 502, got %d", rec.Code)
			}

			if loc := rec.Header().Get("Location"); loc != "" {
				t.Fatalf("must not redirect, got Location %q", loc)
			}

			if !strings.Contains(rec.Body.String(), "Tickets for this event are not available") {
				t.Fatal("want a clear message")
			}

			if strings.Contains(rec.Body.String(), "evil.com") {
				t.Fatal("the rejected url must not be shown to the visitor")
			}
		})
	}
}

func TestRedirect_ErrorPages(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		setup   func()
		status  int
		message string
	}{
		{
			"unknown event",
			"/redirect/missing",
			nil,
			404,
			"We could not find this event",
		},
		{
			"malformed id",
			"/redirect/bad!id",
			nil,
			404,
			"We could not find this event",
		},
		{
			"upstream down",
			"/redirect/ev1",
			func() {
				fake.tmDown.Store(true)
			},
			502,
			"unavailable right now",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetFake(t)

			if tc.setup != nil {
				tc.setup()
			}

			rec := do("GET", tc.target)

			if rec.Code != tc.status ||
				!strings.Contains(rec.Body.String(), tc.message) {
				t.Fatalf(
					"want %d with %q, got %d:\n%s",
					tc.status,
					tc.message,
					rec.Code,
					rec.Body.String(),
				)
			}

			if rec.Header().Get("Location") != "" {
				t.Fatal("an error must never redirect")
			}
		})
	}
}