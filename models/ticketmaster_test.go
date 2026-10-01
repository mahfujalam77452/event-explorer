package models

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func rawEvents(n int) TMEventsResponse {
	var r TMEventsResponse
	for i := 1; i <= n; i++ {
		r.Embedded.Events = append(r.Embedded.Events, TMEvent{
			ID:   fmt.Sprintf("ev%d", i),
			Name: fmt.Sprintf("Event %d", i),
		})
	}
	return r
}

func wantDate(day int) string {
	return time.Date(2026, 10, day, 0, 0, 0, 0, time.UTC).Format("Mon, 2 Jan 2006")
}

func TestToEvents_LimitsToSix(t *testing.T) {
	events := rawEvents(10).ToEvents(6)
	if len(events) != 6 {
		t.Fatalf("want exactly 6 events, got %d", len(events))
	}
	if events[0].ID != "ev1" || events[5].ID != "ev6" {
		t.Fatalf("order not preserved: first=%s last=%s", events[0].ID, events[5].ID)
	}
}

func TestToEvents_FewerThanLimitReturnsAll(t *testing.T) {
	if got := len(rawEvents(3).ToEvents(6)); got != 3 {
		t.Fatalf("want 3 events, got %d", got)
	}
}

func TestToEvents_SkipsInvalidAndDuplicateEvents(t *testing.T) {
	var r TMEventsResponse
	r.Embedded.Events = []TMEvent{
		{ID: "good1", Name: "Good one"},
		{ID: "", Name: "No id"},
		{ID: "bad id!", Name: "Malformed id"},
		{ID: "noname", Name: "   "},
		{ID: "good1", Name: "Duplicate of good one"},
		{ID: "good2", Name: "Good two"},
	}

	events := r.ToEvents(6)
	if len(events) != 2 {
		t.Fatalf("want 2 valid events, got %d: %+v", len(events), events)
	}
	if events[0].ID != "good1" || events[1].ID != "good2" {
		t.Fatalf("unexpected events: %+v", events)
	}
	if events[0].Name != "Good one" {
		t.Fatalf("the first duplicate should win, got %q", events[0].Name)
	}
}

func TestToEvents_EmptyResponse(t *testing.T) {
	// কোনো event না থাকলে Ticketmaster "_embedded" ফিল্ডই পাঠায় না
	var r TMEventsResponse
	if err := json.Unmarshal([]byte(`{"page":{"totalElements":0}}`), &r); err != nil {
		t.Fatal(err)
	}
	if got := len(r.ToEvents(6)); got != 0 {
		t.Fatalf("want 0 events, got %d", got)
	}
}

const sampleEventJSON = `{
  "id": "G5vYZ9abc123",
  "name": "  Big Concert  ",
  "url": "https://www.ticketmaster.ca/event/123",
  "info": "Doors at 6",
  "images": [
    {"url": "https://img/small.jpg",     "ratio": "4_3",  "width": 200,  "height": 150},
    {"url": "https://img/wide-640.jpg",  "ratio": "16_9", "width": 640,  "height": 360},
    {"url": "https://img/wide-1024.jpg", "ratio": "16_9", "width": 1024, "height": 576},
    {"url": "https://img/huge.jpg",      "ratio": "3_2",  "width": 2048, "height": 1365}
  ],
  "dates": {"start": {"localDate": "2026-10-12", "localTime": "19:30:00"}},
  "_embedded": {"venues": [{
    "name": "Scotiabank Arena",
    "city": {"name": "Toronto"},
    "state": {"name": "Ontario", "stateCode": "ON"},
    "country": {"name": "Canada", "countryCode": "CA"},
    "address": {"line1": "40 Bay St"}
  }]}
}`

func TestToEvent_MapsAllFields(t *testing.T) {
	var raw TMEvent
	if err := json.Unmarshal([]byte(sampleEventJSON), &raw); err != nil {
		t.Fatal(err)
	}
	ev := raw.ToEvent()

	checks := []struct{ field, got, want string }{
		{"ID", ev.ID, "G5vYZ9abc123"},
		{"Name", ev.Name, "Big Concert"},
		{"ImageURL", ev.ImageURL, "https://img/wide-640.jpg"},
		{"Date", ev.Date, wantDate(12) + ", 7:30 PM"},
		{"Venue", ev.Venue, "Scotiabank Arena"},
		{"Location", ev.Location, "Scotiabank Arena, 40 Bay St, Toronto, ON, Canada"},
		{"Description", ev.Description, "Doors at 6"},
		{"TicketURL", ev.TicketURL, "https://www.ticketmaster.ca/event/123"},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s: want %q, got %q", c.field, c.want, c.got)
		}
	}
}

func TestToEvent_MissingOptionalFields(t *testing.T) {
	ev := TMEvent{ID: "abc", Name: "Bare event"}.ToEvent()
	if ev.ImageURL != "" || ev.Date != "" || ev.Venue != "" || ev.Location != "" || ev.Description != "" {
		t.Fatalf("optional fields should be empty: %+v", ev)
	}
	if err := ev.Validate(); err != nil {
		t.Fatalf("an event with only id and name is still valid: %v", err)
	}
}

func TestPickImage(t *testing.T) {
	if got := pickImage(nil); got != "" {
		t.Fatalf("no images: want empty, got %q", got)
	}

	blank := []TMImage{{URL: "  ", Ratio: "16_9", Width: 800}}
	if got := pickImage(blank); got != "" {
		t.Fatalf("blank urls must be ignored, got %q", got)
	}

	// if no good image then choosing the big size
	noGood := []TMImage{
		{URL: "a", Ratio: "4_3", Width: 200},
		{URL: "b", Ratio: "3_2", Width: 800},
	}
	if got := pickImage(noGood); got != "b" {
		t.Fatalf("want the largest fallback image, got %q", got)
	}
}

func TestDescriptionFallbackOrder(t *testing.T) {
	ev := TMEvent{ID: "a", Name: "n", Description: "", Info: "  ", PleaseNote: "Note"}.ToEvent()
	if ev.Description != "Note" {
		t.Fatalf("want pleaseNote fallback, got %q", ev.Description)
	}
	ev = TMEvent{ID: "a", Name: "n", Description: "Main", Info: "Info"}.ToEvent()
	if ev.Description != "Main" {
		t.Fatalf("description should win, got %q", ev.Description)
	}
}

func TestFormatEventDate(t *testing.T) {
	tests := []struct{ name, date, clock, want string }{
		{"date and time", "2026-10-12", "19:30:00", wantDate(12) + ", 7:30 PM"},
		{"date only", "2026-10-12", "", wantDate(12)},
		{"bad time is ignored", "2026-10-12", "soon", wantDate(12)},
		{"missing date", "", "19:30:00", ""},
		{"garbage date", "garbage", "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatEventDate(tc.date, tc.clock); got != tc.want {
				t.Fatalf("want %q, got %q", tc.want, got)
			}
		})
	}
}