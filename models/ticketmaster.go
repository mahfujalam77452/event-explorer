package models

import (
	"strings"
	"time"
)


// Raw response for Ticketmaster

type TMImage struct {
	URL      string `json:"url"`
	Ratio    string `json:"ratio"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Fallback bool   `json:"fallback"`
}

// Helper struct for  TMVenue
type CityName struct {
		Name string `json:"name"`
	}

type State struct {
		Name      string `json:"name"`
		StateCode string `json:"stateCode"`
	}
type Country struct {
		Name        string `json:"name"`
		CountryCode string `json:"countryCode"`
	}
type Address struct {
		Line1 string `json:"line1"`
	}



type TMVenue struct {
	Name    string `json:"name"`
	City  CityName  `json:"city"`
	State  State `json:"state"`
	Country  Country `json:"country"`
	Address Address `json:"address"`
}


//helper struct for TMEvent
type Start struct {
			LocalDate string `json:"localDate"`
			LocalTime string `json:"localTime"`
		}

type Dates  struct {
		Start Start `json:"start"`
	}
type Embedded struct {
		Venues []TMVenue `json:"venues"`
	}


type TMEvent struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	URL         string    `json:"url"`
	Info        string    `json:"info"`
	Description string    `json:"description"`
	PleaseNote  string    `json:"pleaseNote"`
	Images      []TMImage `json:"images"`
	Dates    Dates   `json:"dates"`
	Embedded  Embedded `json:"_embedded"`
}

// TMEventsResponse: GET /events.json response

type TMEventsResponse struct {
	Embedded struct {
		Events []TMEvent `json:"events"`
	} `json:"_embedded"`
}

// Translate for Frontend data

// Translate  raw event to view model 
func (t TMEvent) ToEvent() Event {
	ev := Event{
		ID:          strings.TrimSpace(t.ID),
		Name:        strings.TrimSpace(t.Name),
		ImageURL:    pickImage(t.Images),
		Date:        formatEventDate(t.Dates.Start.LocalDate, t.Dates.Start.LocalTime),
		Description: firstNonEmpty(t.Description, t.Info, t.PleaseNote),
		TicketURL:   strings.TrimSpace(t.URL),
	}

	if len(t.Embedded.Venues) > 0 {
		v := t.Embedded.Venues[0]
		ev.Venue = strings.TrimSpace(v.Name)

		state := firstNonEmpty(v.State.StateCode, v.State.Name)
		country := firstNonEmpty(v.Country.Name, v.Country.CountryCode)
		ev.Location = joinNonEmpty(", ",
			v.Name, v.Address.Line1, v.City.Name, state, country)
	}
	return ev
}

//Reform  ToEvents List : truncate illegal and duplicate event .
// return at most 6 events
func (r TMEventsResponse) ToEvents(limit int) []Event {
	events := make([]Event, 0, limit)
	seen := make(map[string]bool)

	for _, raw := range r.Embedded.Events {
		if limit > 0 && len(events) >= limit {
			break
		}
		ev := raw.ToEvent()
		if ev.Validate() != nil || seen[ev.ID] {
			continue
		}
		seen[ev.ID] = true
		events = append(events, ev)
	}
	return events
}

//  helper 

// pickImage selects the smallest image with a 16:9 aspect ratio and a width of 500px or more
// for faster loading. If none is found, it selects the largest image.

func pickImage(images []TMImage) string {
	var best *TMImage
	for i := range images {
		img := &images[i]
		if strings.TrimSpace(img.URL) == "" {
			continue
		}
		if best == nil || betterImage(img, best) {
			best = img
		}
	}
	if best == nil {
		return ""
	}
	return best.URL
}

func betterImage(a, b *TMImage) bool {
	aGood := a.Ratio == "16_9" && a.Width >= 500
	bGood := b.Ratio == "16_9" && b.Width >= 500
	if aGood != bGood {
		return aGood
	}
	if aGood {
		return a.Width < b.Width
	}
	return a.Width > b.Width
}

// formatEventDate "2026-10-12" + "19:30:00" -> "Mon, 12 Oct 2026, 7:30 PM" 

func formatEventDate(localDate, localTime string) string {
	d, err := time.Parse("2006-01-02", strings.TrimSpace(localDate))
	if err != nil {
		return ""
	}
	out := d.Format("Mon, 2 Jan 2006")

	if t, err := time.Parse("15:04:05", strings.TrimSpace(localTime)); err == nil {
		out += ", " + t.Format("3:04 PM")
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

func joinNonEmpty(sep string, values ...string) string {
	parts := make([]string, 0, len(values))
	seen := make(map[string]bool)
	for _, v := range values {
		s := strings.TrimSpace(v)
		if s != "" && !seen[s] {
			seen[s] = true
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, sep)
}