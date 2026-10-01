package services

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"event-explorer/models"
	"event-explorer/utils"
)

const (
	testGoogleKey   = "test-google-key"
	testSessionTok  = "test-session-1234"
	autocompleteRes = `{"suggestions":[
	  {"placePrediction":{"placeId":"ChIJtoronto","text":{"text":"Toronto, ON, Canada"},
	    "structuredFormat":{"mainText":{"text":"Toronto"},"secondaryText":{"text":"ON, Canada"}}}},
	  {"queryPrediction":{"text":{"text":"toronto weather"}}}
	]}`
	placeRes = `{"addressComponents":[
	  {"longText":"Toronto","shortText":"Toronto","types":["locality","political"]},
	  {"longText":"Ontario","shortText":"ON","types":["administrative_area_level_1","political"]},
	  {"longText":"Canada","shortText":"CA","types":["country","political"]}
	]}`
)

func newTestLocationService(t *testing.T, handler http.HandlerFunc) *LocationService {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	return NewLocationService(utils.NewHTTPClient(5*time.Second), ts.URL, testGoogleKey)
}

func TestAutocomplete_SendsCorrectRequestAndParsesResult(t *testing.T) {
	svc := newTestLocationService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("want POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/places:autocomplete" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if r.Header.Get("X-Goog-Api-Key") != testGoogleKey {
			t.Errorf("api key header missing")
		}

		var body struct {
			Input                string   `json:"input"`
			IncludedPrimaryTypes []string `json:"includedPrimaryTypes"`
			SessionToken         string   `json:"sessionToken"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("bad request body: %v", err)
		}
		if body.Input != "toro" {
			t.Errorf("want input toro, got %q", body.Input)
		}
		if len(body.IncludedPrimaryTypes) != 1 || body.IncludedPrimaryTypes[0] != "(cities)" {
			t.Errorf("want includedPrimaryTypes [(cities)], got %v", body.IncludedPrimaryTypes)
		}
		if body.SessionToken != testSessionTok {
			t.Errorf("session token not forwarded: %q", body.SessionToken)
		}
		writeJSON(w, autocompleteRes)
	})

	got, err := svc.Autocomplete(context.Background(), "  toro ", testSessionTok)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
		if len(got) != 1 {
		t.Fatalf("query predictions must be skipped, want 1 suggestion, got %d", len(got))
	}
	if got[0].PlaceID != "ChIJtoronto" || got[0].Text != "Toronto, Canada" {
		t.Fatalf("unexpected suggestion: %+v", got[0])
	}
}

func TestAutocomplete_ShortInputSkipsGoogle(t *testing.T) {
	var hits atomic.Int32
	svc := newTestLocationService(t, func(w http.ResponseWriter, r *http.Request) { hits.Add(1) })

		for _, in := range []string{"", " ", "t", "  t  ", "to", "  to  "} {
		got, err := svc.Autocomplete(context.Background(), in, testSessionTok)
		if err != nil || len(got) != 0 {
			t.Errorf("input %q: want empty result and no error, got %v / %v", in, got, err)
		}
	}
	if hits.Load() != 0 {
		t.Fatalf("Google must not be called for short input, got %d calls", hits.Load())
	}
}

func TestAutocomplete_RejectsBadInputWithoutCallingGoogle(t *testing.T) {
	var hits atomic.Int32
	svc := newTestLocationService(t, func(w http.ResponseWriter, r *http.Request) { hits.Add(1) })

	tests := []struct{ name, input, token string }{
		{"token too short", "toronto", "x"},
		{"token has bad characters", "toronto", "bad token!"},
		{"empty token", "toronto", ""},
		{"input too long", strings.Repeat("a", 101), testSessionTok},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Autocomplete(context.Background(), tc.input, tc.token)
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("want ErrInvalidInput, got %v", err)
			}
		})
	}
	if hits.Load() != 0 {
		t.Fatalf("invalid requests must not reach Google, got %d calls", hits.Load())
	}
}

func TestAutocomplete_GoogleFailure(t *testing.T) {
	svc := newTestLocationService(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"API key not valid"}`, http.StatusForbidden)
	})

	_, err := svc.Autocomplete(context.Background(), "toronto", testSessionTok)

	var httpErr *utils.HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusForbidden {
		t.Fatalf("want *HTTPError 403, got %v", err)
	}
}

func TestPlaceDetails_SendsCorrectRequestAndParsesCity(t *testing.T) {
	svc := newTestLocationService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("want GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/places/ChIJtoronto" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if r.URL.Query().Get("sessionToken") != testSessionTok {
			t.Errorf("session token not forwarded")
		}
		if r.Header.Get("X-Goog-FieldMask") != "addressComponents" {
			t.Errorf("field mask missing, got %q", r.Header.Get("X-Goog-FieldMask"))
		}
		if r.Header.Get("X-Goog-Api-Key") != testGoogleKey {
			t.Errorf("api key header missing")
		}
		writeJSON(w, placeRes)
	})

	city, err := svc.PlaceDetails(context.Background(), "ChIJtoronto", testSessionTok)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if city.Name != "Toronto" || city.CountryCode != "CA" {
		t.Fatalf("want Toronto/CA, got %+v", city)
	}
		if city.PlaceID != "ChIJtoronto" || city.Name != "Toronto" ||
		city.CountryCode != "CA" || city.Label != "Toronto, Canada" {
		t.Fatalf("want ChIJtoronto/Toronto/CA/\"Toronto, Canada\", got %+v", city)
	}
}

func TestPlaceDetails_RejectsBadInputWithoutCallingGoogle(t *testing.T) {
	var hits atomic.Int32
	svc := newTestLocationService(t, func(w http.ResponseWriter, r *http.Request) { hits.Add(1) })

	tests := []struct{ name, placeID, token string }{
		{"empty place id", "", testSessionTok},
		{"path traversal", "../secret", testSessionTok},
		{"slash in id", "a/b", testSessionTok},
		{"bad token", "ChIJtoronto", "x"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.PlaceDetails(context.Background(), tc.placeID, tc.token)
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("want ErrInvalidInput, got %v", err)
			}
		})
	}
	if hits.Load() != 0 {
		t.Fatalf("invalid requests must not reach Google, got %d calls", hits.Load())
	}
}

func TestPlaceDetails_PlaceWithoutCity(t *testing.T) {
	svc := newTestLocationService(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, `{"addressComponents":[{"longText":"Canada","shortText":"CA","types":["country"]}]}`)
	})

	_, err := svc.PlaceDetails(context.Background(), "ChIJcanada", testSessionTok)
	if !errors.Is(err, models.ErrCityNotFound) {
		t.Fatalf("want ErrCityNotFound, got %v", err)
	}

	
}