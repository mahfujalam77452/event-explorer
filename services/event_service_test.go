package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"event-explorer/models"
	"event-explorer/utils"
)

const testAPIKey = "test-tm-key"

//helper for  Ticketmaster 

func tmEvent(id, name string) map[string]interface{} {
	return map[string]interface{}{
		"id":   id,
		"name": name,
		"url":  "https://www.ticketmaster.com/event/" + id,
		"images": []map[string]interface{}{
			{"url": "https://img.example/" + id + ".jpg", "ratio": "16_9", "width": 640, "height": 360},
		},
		"dates": map[string]interface{}{
			"start": map[string]interface{}{"localDate": "2026-10-12", "localTime": "19:30:00"},
		},
		"_embedded": map[string]interface{}{
			"venues": []map[string]interface{}{
				{"name": "Test Arena", "city": map[string]interface{}{"name": "Toronto"}},
			},
		},
	}
}

func manyEvents(prefix string, n int) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, n)
	for i := 1; i <= n; i++ {
		out = append(out, tmEvent(fmt.Sprintf("%s%d", prefix, i), fmt.Sprintf("%s event %d", prefix, i)))
	}
	return out
}
// tmListJSON omits "_embedded" when there are no events, matching Ticketmaster's response format.
func tmListJSON(events ...map[string]interface{}) string {
	body := map[string]interface{}{"page": map[string]interface{}{"totalElements": len(events)}}
	if len(events) > 0 {
		body["_embedded"] = map[string]interface{}{"events": events}
	}
	b, _ := json.Marshal(body)
	return string(b)
}

func writeJSON(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(body))
}

func categoryOf(r *http.Request) string {
	return strings.ToLower(r.URL.Query().Get("classificationName"))
}

func newTestEventService(t *testing.T, timeout time.Duration, handler http.HandlerFunc) (*EventService, *EventCache) {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)

	cache := NewEventCache()
	svc := NewEventService(utils.NewHTTPClient(timeout), ts.URL, testAPIKey, 6, cache)
	return svc, cache
}

// Holding captureLog log so that we can verify  HIT/MISS log 
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	return &buf
}

func eventIDs(events []models.Event) []string {
	ids := make([]string, 0, len(events))
	for _, e := range events {
		ids = append(ids, e.ID)
	}
	return ids
}

//  GetListing: Success 

func TestGetListing_ReturnsExactlySixEventsPerCategory(t *testing.T) {
	var hits atomic.Int32

	svc, _ := newTestEventService(t, 5*time.Second, func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		q := r.URL.Query()
		if r.URL.Path != "/events.json" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if q.Get("apikey") != testAPIKey {
			t.Errorf("api key not sent, got %q", q.Get("apikey"))
		}
		if q.Get("city") != "Toronto" || q.Get("countryCode") != "CA" {
			t.Errorf("wrong city/country: %q %q", q.Get("city"), q.Get("countryCode"))
		}
		if q.Get("size") != "6" {
			t.Errorf("want size=6, got %q", q.Get("size"))
		}
		// Intentionally send 8 requests; the service must limit them to 6.
		writeJSON(w, tmListJSON(manyEvents(categoryOf(r), 8)...))
	})

	sections := svc.GetListing(context.Background(), "Toronto", "CA")

	if len(sections) != 2 {
		t.Fatalf("want 2 sections, got %d", len(sections))
	}
	for i, want := range []string{"Music", "Sports"} {
		s := sections[i]
		if s.Category != want {
			t.Fatalf("section %d: want %s, got %s", i, want, s.Category)
		}
		if s.Error != "" {
			t.Fatalf("%s: unexpected error %q", want, s.Error)
		}
		if len(s.Events) != 6 {
			t.Fatalf("%s: want exactly 6 events, got %d", want, len(s.Events))
		}
		prefix := strings.ToLower(want)
		if s.Events[0].ID != prefix+"1" || s.Events[5].ID != prefix+"6" {
			t.Fatalf("%s: wrong events: %v", want, eventIDs(s.Events))
		}
	}
	if got := hits.Load(); got != 2 {
		t.Fatalf("want exactly one request per category (2), got %d", got)
	}
}

func TestGetListing_RunsCategoriesConcurrently(t *testing.T) {
	// Neither request receives a response unless both requests arrive concurrently.
// Running them sequentially will cause the test to fail; only concurrent execution will pass.
	var bothArrived sync.WaitGroup
	bothArrived.Add(2)

	svc, _ := newTestEventService(t, 10*time.Second, func(w http.ResponseWriter, r *http.Request) {
		bothArrived.Done()
		ready := make(chan struct{})
		go func() { bothArrived.Wait(); close(ready) }()

		select {
		case <-ready:
			writeJSON(w, tmListJSON(manyEvents(categoryOf(r), 2)...))
		case <-time.After(3 * time.Second):
			http.Error(w, "the other request never arrived: not concurrent", http.StatusInternalServerError)
		case <-r.Context().Done():
		}
	})

	sections := svc.GetListing(context.Background(), "Toronto", "CA")

	for _, s := range sections {
		if s.Error != "" || len(s.Events) != 2 {
			t.Fatalf("%s: requests were not made concurrently (error=%q, events=%d)",
				s.Category, s.Error, len(s.Events))
		}
	}
}

// GetListing:failed

func TestGetListing_KeepsSuccessfulSectionWhenOtherFails(t *testing.T) {
	svc, cache := newTestEventService(t, 5*time.Second, func(w http.ResponseWriter, r *http.Request) {
		if categoryOf(r) == "sports" {
			http.Error(w, "upstream exploded", http.StatusInternalServerError)
			return
		}
		writeJSON(w, tmListJSON(manyEvents("music", 3)...))
	})

	sections := svc.GetListing(context.Background(), "Toronto", "CA")

	music, sports := sections[0], sections[1]
	if music.Error != "" || len(music.Events) != 3 {
		t.Fatalf("music should have survived: error=%q events=%d", music.Error, len(music.Events))
	}
	if sports.Error == "" || len(sports.Events) != 0 {
		t.Fatalf("sports should show an error: %+v", sports)
	}
	if !strings.Contains(sports.Error, "Sports") {
		t.Fatalf("error message should name the category, got %q", sports.Error)
	}
	if strings.Contains(sports.Error, "exploded") || strings.Contains(sports.Error, testAPIKey) {
		t.Fatalf("internal details leaked to the user: %q", sports.Error)
	}
	if cache.Len() != 1 {
		t.Fatalf("only the successful category may be cached, cache has %d entries", cache.Len())
	}
}

func TestGetListing_BothCategoriesFail(t *testing.T) {
	svc, cache := newTestEventService(t, 5*time.Second, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusBadGateway)
	})

	sections := svc.GetListing(context.Background(), "Toronto", "CA")

	for _, s := range sections {
		if s.Error == "" || len(s.Events) != 0 {
			t.Fatalf("%s should be an error section: %+v", s.Category, s)
		}
	}
	if cache.Len() != 0 {
		t.Fatalf("failures must never be cached, cache has %d entries", cache.Len())
	}
}

func TestGetListing_TimeoutShowsFriendlyMessage(t *testing.T) {
	svc, _ := newTestEventService(t, 100*time.Millisecond, func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(2 * time.Second):
		case <-r.Context().Done():
		}
	})

	sections := svc.GetListing(context.Background(), "Toronto", "CA")

	for _, s := range sections {
		if !strings.Contains(s.Error, "took too long") {
			t.Fatalf("%s: want a timeout message, got %q", s.Category, s.Error)
		}
	}
}

func TestGetListing_EmptyResultIsNotAnError(t *testing.T) {
	svc, _ := newTestEventService(t, 5*time.Second, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, tmListJSON()) // "_embedded" নেই
	})

	sections := svc.GetListing(context.Background(), "Nowhere", "US")

	for _, s := range sections {
		if s.Error != "" {
			t.Fatalf("%s: empty result must not be an error, got %q", s.Category, s.Error)
		}
		if !s.IsEmpty() {
			t.Fatalf("%s should be empty: %+v", s.Category, s)
		}
	}
}

func TestGetListing_DropsInvalidAndDuplicateEvents(t *testing.T) {
	svc, _ := newTestEventService(t, 5*time.Second, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, tmListJSON(
			tmEvent("good1", "Good one"),
			tmEvent("", "No id"),
			tmEvent("bad id!", "Malformed id"),
			tmEvent("noname", "   "),
			tmEvent("good1", "Duplicate"),
			tmEvent("good2", "Good two"),
		))
	})

	sections := svc.GetListing(context.Background(), "Toronto", "CA")

	for _, s := range sections {
		ids := eventIDs(s.Events)
		if len(ids) != 2 || ids[0] != "good1" || ids[1] != "good2" {
			t.Fatalf("%s: want [good1 good2], got %v", s.Category, ids)
		}
	}
}

// Cache: HIT / MISS / clear

func TestGetListing_SecondCallIsServedFromCache(t *testing.T) {
	logs := captureLog(t)
	var hits atomic.Int32

	svc, cache := newTestEventService(t, 5*time.Second, func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		writeJSON(w, tmListJSON(manyEvents(categoryOf(r), 6)...))
	})
	ctx := context.Background()

	first := svc.GetListing(ctx, "Toronto", "CA")
	if hits.Load() != 2 {
		t.Fatalf("first call: want 2 upstream requests, got %d", hits.Load())
	}
	if cache.Len() != 2 {
		t.Fatalf("both categories should be cached, got %d entries", cache.Len())
	}

	// The key is case-insensitive, so Ticketmaster will not be called again.
	second := svc.GetListing(ctx, "toronto", "ca")
	if hits.Load() != 2 {
		t.Fatalf("second call must come from cache, but upstream got %d requests", hits.Load())
	}

	for i := range first {
		a, b := eventIDs(first[i].Events), eventIDs(second[i].Events)
		if strings.Join(a, ",") != strings.Join(b, ",") {
			t.Fatalf("cached result differs: %v vs %v", a, b)
		}
	}

	out := logs.String()
	if n := strings.Count(out, "[cache] MISS"); n != 2 {
		t.Errorf("want 2 MISS log lines, got %d\n%s", n, out)
	}
	if n := strings.Count(out, "[cache] HIT"); n != 2 {
		t.Errorf("want 2 HIT log lines, got %d\n%s", n, out)
	}

	// different city = new key = again MISS
	svc.GetListing(ctx, "London", "GB")
	if hits.Load() != 4 {
		t.Fatalf("a different city must miss the cache, upstream requests = %d", hits.Load())
	}
}

func TestGetListing_FailedResponseIsNotCached(t *testing.T) {
	var failing atomic.Bool
	failing.Store(true)

	svc, cache := newTestEventService(t, 5*time.Second, func(w http.ResponseWriter, r *http.Request) {
		if failing.Load() {
			http.Error(w, "temporary outage", http.StatusServiceUnavailable)
			return
		}
		writeJSON(w, tmListJSON(manyEvents(categoryOf(r), 6)...))
	})
	ctx := context.Background()

	for _, s := range svc.GetListing(ctx, "Toronto", "CA") {
		if s.Error == "" {
			t.Fatalf("%s should have failed", s.Category)
		}
	}
	if cache.Len() != 0 {
		t.Fatalf("failure was cached (%d entries)", cache.Len())
	}

	failing.Store(false) // server is functional now
	for _, s := range svc.GetListing(ctx, "Toronto", "CA") {
		if s.Error != "" || len(s.Events) != 6 {
			t.Fatalf("%s should recover after the outage: %+v", s.Category, s)
		}
	}
	if cache.Len() != 2 {
		t.Fatalf("successful results should now be cached, got %d entries", cache.Len())
	}
}

func TestClearCache_ForcesFreshRequests(t *testing.T) {
	var hits atomic.Int32
	svc, cache := newTestEventService(t, 5*time.Second, func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		writeJSON(w, tmListJSON(manyEvents(categoryOf(r), 6)...))
	})
	ctx := context.Background()

	svc.GetListing(ctx, "Toronto", "CA")
	svc.GetListing(ctx, "Toronto", "CA") //from cache 
	if hits.Load() != 2 {
		t.Fatalf("want 2 upstream requests before clearing, got %d", hits.Load())
	}

	if removed := svc.ClearCache(); removed != 2 {
		t.Fatalf("ClearCache should report 2 removed entries, got %d", removed)
	}
	if cache.Len() != 0 {
		t.Fatalf("cache should be empty, has %d entries", cache.Len())
	}

	svc.GetListing(ctx, "Toronto", "CA")
	if hits.Load() != 4 {
		t.Fatalf("after clearing, upstream must be called again; requests = %d", hits.Load())
	}
}

//GetEvent

func TestGetEvent_Success(t *testing.T) {
	svc, _ := newTestEventService(t, 5*time.Second, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/events/ev1.json" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if r.URL.Query().Get("apikey") != testAPIKey {
			t.Errorf("api key not sent")
		}
		e := tmEvent("ev1", "Great Show")
		e["description"] = "A great night of music."
		b, _ := json.Marshal(e)
		writeJSON(w, string(b))
	})

	ev, err := svc.GetEvent(context.Background(), "ev1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.ID != "ev1" || ev.Name != "Great Show" {
		t.Fatalf("wrong event: %+v", ev)
	}
	if ev.Description != "A great night of music." {
		t.Fatalf("wrong description: %q", ev.Description)
	}
	if !strings.Contains(ev.Date, "12 Oct 2026") || !strings.Contains(ev.Date, "7:30 PM") {
		t.Fatalf("wrong date: %q", ev.Date)
	}
	if ev.Venue != "Test Arena" {
		t.Fatalf("wrong venue: %q", ev.Venue)
	}
	if ev.TicketURL != "https://www.ticketmaster.com/event/ev1" {
		t.Fatalf("ticket url missing: %q", ev.TicketURL)
	}
}

func TestGetEvent_NotFound(t *testing.T) {
	svc, _ := newTestEventService(t, 5*time.Second, func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	_, err := svc.GetEvent(context.Background(), "doesnotexist")
	if !errors.Is(err, models.ErrEventNotFound) {
		t.Fatalf("want ErrEventNotFound, got %v", err)
	}
}

func TestGetEvent_InvalidIDNeverReachesUpstream(t *testing.T) {
	var hits atomic.Int32
	svc, _ := newTestEventService(t, 5*time.Second, func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
	})

	bad := []string{"", "bad id", "../etc/passwd", "a/b", "id?x=1", "id#frag", strings.Repeat("a", 65)}
	for _, id := range bad {
		if _, err := svc.GetEvent(context.Background(), id); !errors.Is(err, models.ErrInvalidEventID) {
			t.Errorf("id %q: want ErrInvalidEventID, got %v", id, err)
		}
	}
	if hits.Load() != 0 {
		t.Fatalf("invalid ids must not be sent upstream, but %d requests were made", hits.Load())
	}
}

func TestGetEvent_UpstreamFailureIsNotReportedAsNotFound(t *testing.T) {
	svc, _ := newTestEventService(t, 5*time.Second, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})

	_, err := svc.GetEvent(context.Background(), "ev1")
	if err == nil {
		t.Fatal("expected an error")
	}
	if errors.Is(err, models.ErrEventNotFound) {
		t.Fatal("a 500 must not be reported as 'not found'")
	}
	var httpErr *utils.HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusInternalServerError {
		t.Fatalf("want *HTTPError with status 500, got %v", err)
	}
}

func TestGetEvent_UnusableEventCountsAsNotFound(t *testing.T) {
	svc, _ := newTestEventService(t, 5*time.Second, func(w http.ResponseWriter, r *http.Request) {
		b, _ := json.Marshal(tmEvent("ev1", "   ")) // নাম নেই
		writeJSON(w, string(b))
	})

	if _, err := svc.GetEvent(context.Background(), "ev1"); !errors.Is(err, models.ErrEventNotFound) {
		t.Fatalf("want ErrEventNotFound, got %v", err)
	}
}

//ValidateCityQuery

func TestValidateCityQuery(t *testing.T) {
	tests := []struct {
		name          string
		city, country string
		valid         bool
	}{
		{"normal", "Toronto", "CA", true},
		{"lowercase country is fine", "New York", "us", true},
		{"unicode city", "São Paulo", "BR", true},
		{"empty city", "", "CA", false},
		{"blank city", "   ", "CA", false},
		{"empty country", "Toronto", "", false},
		{"three letter country", "Toronto", "CAN", false},
		{"digit in country", "Toronto", "C1", false},
		{"city too long", strings.Repeat("a", 101), "CA", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCityQuery(tc.city, tc.country)
			if tc.valid && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tc.valid {
				if err == nil {
					t.Fatal("expected an error")
				}
				if !errors.Is(err, ErrInvalidInput) {
					t.Fatalf("want ErrInvalidInput, got %v", err)
				}
			}
		})
	}
}