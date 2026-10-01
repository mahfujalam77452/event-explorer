package controllers_test

import (
	"net/http"
	"strings"
	"testing"
)

const sessionToken = "test-session-1234"

func TestAutocomplete_MatchesGuideShape(t *testing.T) {
	resetFake(t)
	rec := do("GET", "/api/locations/autocomplete?input=toro&sessionToken="+sessionToken)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("google data must not be cached, Cache-Control = %q", got)
	}

	list, ok := decode(t, rec)["suggestions"].([]interface{})
	if !ok || len(list) != 1 {
		t.Fatalf("want exactly 1 suggestion, got %v", list)
	}
	item := list[0].(map[string]interface{})
	if len(item) != 2 || item["placeId"] != "ChIJtoronto" || item["text"] != "Toronto, Canada" {
		t.Fatalf("suggestion must be exactly {placeId, text}, got %v", item)
	}
}

func TestAutocomplete_ShortInputReturnsEmptyList(t *testing.T) {
	resetFake(t)
	rec := do("GET", "/api/locations/autocomplete?input=to&sessionToken="+sessionToken)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
	list, ok := decode(t, rec)["suggestions"].([]interface{})
	if !ok || len(list) != 0 {
		t.Fatalf("want an empty suggestions list, got %v", list)
	}
}

func TestPlaceDetails_MatchesGuideShape(t *testing.T) {
	resetFake(t)
	rec := do("GET", "/api/locations/ChIJtoronto?sessionToken="+sessionToken)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
	body := decode(t, rec)
	if len(body) != 4 {
		t.Fatalf("response must have exactly 4 fields, got %v", body)
	}
	if body["placeId"] != "ChIJtoronto" || body["city"] != "Toronto" ||
		body["countryCode"] != "CA" || body["label"] != "Toronto, Canada" {
		t.Fatalf("unexpected body: %v", body)
	}
}

func TestLocationAPI_ErrorResponses(t *testing.T) {
	valid := "/api/locations/autocomplete?input=toro&sessionToken=" + sessionToken

	tests := []struct {
		name    string
		target  string
		setup   func()
		status  int
		message string
	}{
		{"autocomplete: bad session token", "/api/locations/autocomplete?input=toro&sessionToken=x", nil, 400, "Invalid request."},
		{"autocomplete: input too long", "/api/locations/autocomplete?input=" + strings.Repeat("a", 101) + "&sessionToken=" + sessionToken, nil, 400, "Invalid request."},
		{"autocomplete: google rejects the key", valid, func() { fake.googleDown.Store(true) }, 502, "unavailable right now"},
		{"autocomplete: google too slow", valid, func() { fake.googleSlow.Store(true) }, 504, "took too long"},
		{"place: bad place id", "/api/locations/bad!id?sessionToken=" + sessionToken, nil, 400, "Invalid request."},
		{"place: bad session token", "/api/locations/ChIJtoronto?sessionToken=x", nil, 400, "Invalid request."},
		{"place: no city in place", "/api/locations/ChIJnocity?sessionToken=" + sessionToken, nil, 404, "Could not find a city"},
		{"place: google rejects the key", "/api/locations/ChIJtoronto?sessionToken=" + sessionToken, func() { fake.googleDown.Store(true) }, 502, "unavailable right now"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetFake(t)
			if tc.setup != nil {
				tc.setup()
			}
			rec := do("GET", tc.target)

			if rec.Code != tc.status {
				t.Fatalf("want status %d, got %d: %s", tc.status, rec.Code, rec.Body.String())
			}
			msg, _ := decode(t, rec)["error"].(string)
			if !strings.Contains(msg, tc.message) {
				t.Fatalf("want error containing %q, got %q", tc.message, msg)
			}
			if strings.Contains(rec.Body.String(), "API key") {
				t.Fatalf("upstream error details leaked to the visitor: %s", rec.Body.String())
			}
		})
	}
}