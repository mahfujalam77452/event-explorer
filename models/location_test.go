package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
)

func comp(long, short string, types ...string) GoogleAddressComponent {
	return GoogleAddressComponent{LongText: long, ShortText: short, Types: types}
}

func prediction(id, main, secondary string) GoogleSuggestion {
	return GoogleSuggestion{PlacePrediction: &GooglePlacePrediction{
		PlaceID: id,
		Text:    GoogleText{Text: main + ", " + secondary},
		StructuredFormat: GoogleStructuredFormat{
			MainText:      GoogleText{Text: main},
			SecondaryText: GoogleText{Text: secondary},
		},
	}}
}

func TestToSuggestions_BuildsShortTextWithoutState(t *testing.T) {
	resp := GoogleAutocompleteResponse{Suggestions: []GoogleSuggestion{
		prediction("ChIJtoronto", "Toronto", "ON, Canada"),
	}}

	got := resp.ToSuggestions()
	if len(got) != 1 {
		t.Fatalf("want 1 suggestion, got %d", len(got))
	}
	want := Suggestion{PlaceID: "ChIJtoronto", Text: "Toronto, Canada"}
	if got[0] != want {
		t.Fatalf("want %+v, got %+v", want, got[0])
	}
}

func TestToSuggestions_SkipsUnusableItems(t *testing.T) {
	resp := GoogleAutocompleteResponse{Suggestions: []GoogleSuggestion{
		prediction("ChIJtoronto", "Toronto", "ON, Canada"),
		{PlacePrediction: nil}, // query prediction, place নয়
		{PlacePrediction: &GooglePlacePrediction{PlaceID: "  ", Text: GoogleText{Text: "No id"}}},
		{PlacePrediction: &GooglePlacePrediction{PlaceID: "noText"}}, // কোনো লেখা নেই
	}}

	got := resp.ToSuggestions()
	if len(got) != 1 || got[0].PlaceID != "ChIJtoronto" {
		t.Fatalf("want only the usable suggestion, got %+v", got)
	}
}

func TestToSuggestions_LimitsToFive(t *testing.T) {
	var resp GoogleAutocompleteResponse
	for i := 1; i <= 9; i++ {
		resp.Suggestions = append(resp.Suggestions,
			prediction(fmt.Sprintf("id%d", i), fmt.Sprintf("City %d", i), "Canada"))
	}

	got := resp.ToSuggestions()
	if len(got) != MaxSuggestions || MaxSuggestions != 5 {
		t.Fatalf("want exactly 5 suggestions, got %d", len(got))
	}
	if got[0].PlaceID != "id1" || got[4].PlaceID != "id5" {
		t.Fatalf("order not preserved: %+v", got)
	}
}

func TestToSuggestions_EmptyResponse(t *testing.T) {
	got := (GoogleAutocompleteResponse{}).ToSuggestions()
	if got == nil || len(got) != 0 {
		t.Fatalf("want an empty, non-nil slice (so JSON is [] not null), got %#v", got)
	}
}

func TestSuggestionText(t *testing.T) {
	tests := []struct {
		name, main, secondary, full, want string
	}{
		{"state removed", "Toronto", "ON, Canada", "", "Toronto, Canada"},
		{"two parts", "New York", "NY, USA", "", "New York, USA"},
		{"only country", "Paris", "France", "", "Paris, France"},
		{"no secondary", "Monaco", "", "", "Monaco"},
		{"country equals name", "Singapore", "Singapore", "", "Singapore"},
		{"trailing comma", "Paris", "France,", "", "Paris"},
		{"no structured format, use full text", "", "", "Dhaka, Bangladesh", "Dhaka, Bangladesh"},
		{"spaces trimmed", "  Toronto ", " ON ,  Canada ", "", "Toronto, Canada"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &GooglePlacePrediction{
				Text: GoogleText{Text: tc.full},
				StructuredFormat: GoogleStructuredFormat{
					MainText:      GoogleText{Text: tc.main},
					SecondaryText: GoogleText{Text: tc.secondary},
				},
			}
			if got := suggestionText(p); got != tc.want {
				t.Fatalf("want %q, got %q", tc.want, got)
			}
		})
	}
}

func TestToCity(t *testing.T) {
	tests := []struct {
		name        string
		components  []GoogleAddressComponent
		wantCity    string
		wantCountry string
		wantLabel   string
		wantErr     error
	}{
		{
			name: "locality",
			components: []GoogleAddressComponent{
				comp("Toronto", "Toronto", "locality", "political"),
				comp("Ontario", "ON", "administrative_area_level_1", "political"),
				comp("Canada", "CA", "country", "political"),
			},
			wantCity: "Toronto", wantCountry: "CA", wantLabel: "Toronto, Canada",
		},
		{
			name: "postal town (London)",
			components: []GoogleAddressComponent{
				comp("London", "London", "postal_town"),
				comp("United Kingdom", "GB", "country", "political"),
			},
			wantCity: "London", wantCountry: "GB", wantLabel: "London, United Kingdom",
		},
		{
			name: "no locality, falls back to admin area (Tokyo)",
			components: []GoogleAddressComponent{
				comp("Tokyo", "Tokyo", "administrative_area_level_1", "political"),
				comp("Japan", "JP", "country", "political"),
			},
			wantCity: "Tokyo", wantCountry: "JP", wantLabel: "Tokyo, Japan",
		},
		{
			name: "locality wins over admin area",
			components: []GoogleAddressComponent{
				comp("Ontario", "ON", "administrative_area_level_1"),
				comp("Toronto", "Toronto", "locality"),
				comp("Canada", "CA", "country"),
			},
			wantCity: "Toronto", wantCountry: "CA", wantLabel: "Toronto, Canada",
		},
		{
			name: "country code is upper-cased",
			components: []GoogleAddressComponent{
				comp("Paris", "Paris", "locality"),
				comp("France", "fr", "country"),
			},
			wantCity: "Paris", wantCountry: "FR", wantLabel: "Paris, France",
		},
		{
			name: "missing country long name falls back to the code",
			components: []GoogleAddressComponent{
				comp("Paris", "Paris", "locality"),
				comp("", "FR", "country"),
			},
			wantCity: "Paris", wantCountry: "FR", wantLabel: "Paris, FR",
		},
		{
			name:       "no country",
			components: []GoogleAddressComponent{comp("Toronto", "Toronto", "locality")},
			wantErr:    ErrCityNotFound,
		},
		{
			name: "country code is not two letters",
			components: []GoogleAddressComponent{
				comp("Toronto", "Toronto", "locality"),
				comp("Canada", "CAN", "country"),
			},
			wantErr: ErrCityNotFound,
		},
		{
			name:       "no city-like component",
			components: []GoogleAddressComponent{comp("Canada", "CA", "country")},
			wantErr:    ErrCityNotFound,
		},
		{name: "no components at all", components: nil, wantErr: ErrCityNotFound},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			city, err := GooglePlaceDetailsResponse{AddressComponents: tc.components}.ToCity("mock-place")

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("want error %v, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if city.PlaceID != "mock-place" {
				t.Fatalf("placeId must be echoed back, got %q", city.PlaceID)
			}
			if city.Name != tc.wantCity || city.CountryCode != tc.wantCountry || city.Label != tc.wantLabel {
				t.Fatalf("want %s/%s/%q, got %s/%s/%q",
					tc.wantCity, tc.wantCountry, tc.wantLabel, city.Name, city.CountryCode, city.Label)
			}
		})
	}
}


//is Guide's response shape mached exactly:field name and order
func TestJSONShapeMatchesAssignmentGuide(t *testing.T) {
	s, err := json.Marshal(struct {
		Suggestions []Suggestion `json:"suggestions"`
	}{[]Suggestion{{PlaceID: "mock-toronto", Text: "Toronto, Canada"}}})
	if err != nil {
		t.Fatal(err)
	}
	wantS := `{"suggestions":[{"placeId":"mock-toronto","text":"Toronto, Canada"}]}`
	if string(s) != wantS {
		t.Fatalf("suggestions JSON:\nwant %s\ngot  %s", wantS, s)
	}

	c, err := json.Marshal(City{PlaceID: "mock-toronto", Name: "Toronto", CountryCode: "CA", Label: "Toronto, Canada"})
	if err != nil {
		t.Fatal(err)
	}
	wantC := `{"placeId":"mock-toronto","city":"Toronto","countryCode":"CA","label":"Toronto, Canada"}`
	if string(c) != wantC {
		t.Fatalf("city JSON:\nwant %s\ngot  %s", wantC, c)
	}
}