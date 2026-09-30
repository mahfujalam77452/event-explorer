package models

import (
	"errors"
	"strings"
)

// MaxSuggestions
const MaxSuggestions = 5

// ErrCityNotFounf
var ErrCityNotFound = errors.New("city or country not found in place details")

// Suggestion autocomplete result
// JSON: {"placeId":"...","text":"Toronto, Canada"}
type Suggestion struct {
	PlaceID string `json:"placeId"`
	Text    string `json:"text"`
}

// City response
// JSON: {"placeId":"...","city":"Toronto","countryCode":"CA","label":"Toronto, Canada"}
type City struct {
	PlaceID     string `json:"placeId"`
	Name        string `json:"city"`
	CountryCode string `json:"countryCode"`
	Label       string `json:"label"`
}

//For Google raw Response

type GoogleText struct {
	Text string `json:"text"`
}

type GoogleStructuredFormat struct {
	MainText      GoogleText `json:"mainText"`
	SecondaryText GoogleText `json:"secondaryText"`
}

type GooglePlacePrediction struct {
	PlaceID          string                 `json:"placeId"`
	Text             GoogleText             `json:"text"`
	StructuredFormat GoogleStructuredFormat `json:"structuredFormat"`
}

type GoogleSuggestion struct {
	PlacePrediction *GooglePlacePrediction `json:"placePrediction"`
}

// GoogleAutocompleteResponse: POST /v1/places:autocomplete 
type GoogleAutocompleteResponse struct {
	Suggestions []GoogleSuggestion `json:"suggestions"`
}

type GoogleAddressComponent struct {
	LongText  string   `json:"longText"`
	ShortText string   `json:"shortText"`
	Types     []string `json:"types"`
}

// GooglePlaceDetailsResponse: GET /v1/places/{id} 
type GooglePlaceDetailsResponse struct {
	AddressComponents []GoogleAddressComponent `json:"addressComponents"`
}

// ToSuggestions raw to cleaner response।
// placeId , MaxSuggestions return
func (r GoogleAutocompleteResponse) ToSuggestions() []Suggestion {
	result := make([]Suggestion, 0, MaxSuggestions)
	for _, s := range r.Suggestions {
		if len(result) >= MaxSuggestions {
			break
		}
		p := s.PlacePrediction
		if p == nil || strings.TrimSpace(p.PlaceID) == "" {
			continue
		}
		text := suggestionText(p)
		if text == "" {
			continue
		}
		result = append(result, Suggestion{
			PlaceID: strings.TrimSpace(p.PlaceID),
			Text:    text,
		})
	}
	return result
}

//Converting  suggestionText Google Text "Toronto, ON, Canada" to "Toronto, Canada" 

func suggestionText(p *GooglePlacePrediction) string {
	main := strings.TrimSpace(p.StructuredFormat.MainText.Text)
	secondary := strings.TrimSpace(p.StructuredFormat.SecondaryText.Text)

	if main == "" {
		return strings.TrimSpace(p.Text.Text) 
	}
	if secondary == "" {
		return main
	}

	parts := strings.Split(secondary, ",")
	country := strings.TrimSpace(parts[len(parts)-1])
	if country == "" || strings.EqualFold(country, main) {
		return main
	}
	return main + ", " + country
}

// The types of city may exist
var cityComponentTypes = []string{
	"locality",
	"postal_town",
	"administrative_area_level_2",
	"administrative_area_level_1",
}

func findComponent(components []GoogleAddressComponent, wanted string) *GoogleAddressComponent {
	for i := range components {
		for _, t := range components[i].Types {
			if t == wanted {
				return &components[i]
			}
		}
	}
	return nil
}

// Extract city,country code,label from ToCity addressComponents  

func (r GooglePlaceDetailsResponse) ToCity(placeID string) (*City, error) {
	name := ""
	for _, t := range cityComponentTypes {
		if c := findComponent(r.AddressComponents, t); c != nil && strings.TrimSpace(c.LongText) != "" {
			name = strings.TrimSpace(c.LongText)
			break
		}
	}

	countryCode := ""
	countryName := ""
	if c := findComponent(r.AddressComponents, "country"); c != nil {
		countryCode = strings.ToUpper(strings.TrimSpace(c.ShortText))
		countryName = strings.TrimSpace(c.LongText)
	}

	if name == "" || len(countryCode) != 2 {
		return nil, ErrCityNotFound
	}
	if countryName == "" {
		countryName = countryCode 
	}

	return &City{
		PlaceID:     placeID,
		Name:        name,
		CountryCode: countryCode,
		Label:       name + ", " + countryName,
	}, nil
}