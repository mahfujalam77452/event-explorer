package controllers_test

import (
	"strings"
	"testing"
)

func TestHomePage(t *testing.T) {
	resetFake(t)
	rec := do("GET", "/")
	body := rec.Body.String()

	if rec.Code != 200 {
		t.Fatalf("want 200, got %d", rec.Code)
	}

	for _, want := range []string{
		"<title>Find events in your city | Event Explorer</title>", // Prepare() + RenderPage
		"Powered by Google",
		`src="/static/js/autocomplete.js"`,
		`id="search-form"`,
		"Event Explorer", // header partial
	} {
		if !strings.Contains(body, want) {
			t.Errorf("home page should contain %q", want)
		}
	}
}

func TestListing_ShowsSixMusicAndSixSports(t *testing.T) {
	resetFake(t)
	rec := do("GET", "/events?city=Toronto&countryCode=CA")
	body := rec.Body.String()

	if rec.Code != 200 {
		t.Fatalf("want 200, got %d", rec.Code)
	}

	if n := strings.Count(body, `class="card"`); n != 12 {
		t.Fatalf("want 12 cards (6 + 6), got %d", n)
	}

	for _, want := range []string{
		"Events in Toronto, CA",
		`<h2 class="section-title">Music</h2>`,
		`<h2 class="section-title">Sports</h2>`,
		"Show music1",
		"Show music6",
		"Show sports1",
		"Show sports6",
		"View Details",
		"/events/music1?city=Toronto",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("listing should contain %q", want)
		}
	}
}

func TestListing_UpperCasesCountryCode(t *testing.T) {
	resetFake(t)

	body := do("GET", "/events?city=Toronto&countryCode=ca").Body.String()

	if !strings.Contains(body, "Events in Toronto, CA") {
		t.Fatal("country code should be normalised to upper case")
	}
}

func TestListing_MissingOrInvalidParams(t *testing.T) {
	resetFake(t)

	for _, target := range []string{
		"/events",
		"/events?city=Toronto",
		"/events?city=Toronto&countryCode=Canada",
		"/events?countryCode=CA",
	} {
		rec := do("GET", target)

		if rec.Code != 400 {
			t.Errorf("%s: want 400, got %d", target, rec.Code)
		}

		if !strings.Contains(rec.Body.String(), "Please choose a city first") {
			t.Errorf("%s: want the 'choose a city' message", target)
		}
	}
}

func TestListing_OneCategoryFailsOtherIsKept(t *testing.T) {
	resetFake(t)
	fake.sportsDown.Store(true)

	rec := do("GET", "/events?city=Toronto&countryCode=CA")
	body := rec.Body.String()

	if rec.Code != 200 {
		t.Fatalf("a partial failure must still render the page, got %d", rec.Code)
	}

	if n := strings.Count(body, `class="card"`); n != 6 {
		t.Fatalf("want the 6 music cards only, got %d", n)
	}

	if !strings.Contains(body, "Show music1") ||
		!strings.Contains(body, "Sports events are unavailable") {
		t.Fatal("music should be shown and sports should show an error message")
	}
}

func TestListing_BothCategoriesFail(t *testing.T) {
	resetFake(t)
	fake.tmDown.Store(true)

	body := do("GET", "/events?city=Toronto&countryCode=CA").Body.String()

	if n := strings.Count(body, "events are unavailable right now"); n != 2 {
		t.Fatalf("want 2 error messages, got %d", n)
	}

	if strings.Count(body, `class="card"`) != 0 {
		t.Fatal("no cards should be shown")
	}
}

func TestListing_EmptyResults(t *testing.T) {
	resetFake(t)
	fake.tmEmpty.Store(true)

	body := do("GET", "/events?city=Toronto&countryCode=CA").Body.String()

	for _, want := range []string{
		"No Music events found in Toronto.",
		"No Sports events found in Toronto.",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("want %q", want)
		}
	}
}

func TestDetails_ShowsEventAndHidesTicketURL(t *testing.T) {
	resetFake(t)

	rec := do("GET", "/events/ev1?city=Toronto&countryCode=CA")
	body := rec.Body.String()

	if rec.Code != 200 {
		t.Fatalf("want 200, got %d", rec.Code)
	}

	for _, want := range []string{
		"Show ev1",
		"Description for ev1",
		"Test Arena",
		"12 Oct 2026",
		"Back to results",
		"/events?city=Toronto",
		`href="/redirect/ev1"`,
		"View Tickets",
		"https://img.example/ev1.jpg",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("details should contain %q", want)
		}
	}

	if strings.Contains(body, "ticketmaster.com") {
		t.Fatal("the real ticket url must never appear in the HTML")
	}
}

func TestDetails_DirectLinkWithoutQueryGoesBackToSearch(t *testing.T) {
	resetFake(t)

	rec := do("GET", "/events/ev1")

	if rec.Code != 200 {
		t.Fatalf("direct links must work, got %d", rec.Code)
	}

	body := rec.Body.String()

	if !strings.Contains(body, "Back to search") ||
		!strings.Contains(body, `href="/"`) {
		t.Fatal("without a city the back link should point home")
	}
}

func TestDetails_IgnoresBadBackLinkParams(t *testing.T) {
	resetFake(t)

	body := do(
		"GET",
		"/events/ev1?city=%3Cscript%3E&countryCode=XX1",
	).Body.String()

	if !strings.Contains(body, "Back to search") ||
		strings.Contains(body, "<script>") {
		t.Fatal("bad query values must not reach the back link")
	}
}

func TestDetails_NoTicketURLShowsNotice(t *testing.T) {
	resetFake(t)

	rec := do("GET", "/events/nourl1")

	if rec.Code != 200 {
		t.Fatalf("want 200, got %d", rec.Code)
	}

	body := rec.Body.String()

	if !strings.Contains(body, "Tickets are not available for this event yet.") ||
		strings.Contains(body, "View Tickets") {
		t.Fatal("an event without a ticket url must not show the View Tickets button")
	}
}

func TestDetails_ErrorPages(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		setup   func()
		status  int
		message string
	}{
		{
			"unknown event",
			"/events/missing",
			nil,
			404,
			"We could not find this event",
		},
		{
			"malformed id",
			"/events/bad!id",
			nil,
			404,
			"We could not find this event",
		},
		{
			"upstream down",
			"/events/ev1",
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

			if rec.Code != tc.status {
				t.Fatalf("want %d, got %d", tc.status, rec.Code)
			}

			body := rec.Body.String()

			if !strings.Contains(body, tc.message) ||
				!strings.Contains(body, "Search a city") {
				t.Fatalf("want the error message and a way back, got:\n%s", body)
			}
		})
	}
}