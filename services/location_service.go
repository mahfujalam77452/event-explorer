package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"event-explorer/models"
	"event-explorer/utils"
)

const (
	minInputLength = 3
	maxInputLength = 100
)

// if input, sessionToken or placeId is not valid
var ErrInvalidInput = errors.New("invalid input")

var (
	// session token of google
	sessionTokenPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,36}$`)
	// placeId URL 
	placeIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,256}$`)
)

// LocationService Google Places API (New)
type LocationService struct {
	client  *http.Client
	baseURL string
	apiKey  string
}

func NewLocationService(client *http.Client, baseURL, apiKey string) *LocationService {
	return &LocationService{client: client, baseURL: baseURL, apiKey: apiKey}
}

// Autocomplete suggestion 
//return empty array if inputlenght < 3
func (s *LocationService) Autocomplete(ctx context.Context, input, sessionToken string) ([]models.Suggestion, error) {
	input = strings.TrimSpace(input)
	length := utf8.RuneCountInString(input)

	if length < minInputLength {
		return []models.Suggestion{}, nil
	}
	if length > maxInputLength {
		return nil, fmt.Errorf("%w: input too long", ErrInvalidInput)
	}
	if !sessionTokenPattern.MatchString(sessionToken) {
		return nil, fmt.Errorf("%w: bad session token", ErrInvalidInput)
	}

	payload, err := json.Marshal(map[string]interface{}{
		"input":                input,
		"includedPrimaryTypes": []string{"(cities)"},
		"sessionToken":         sessionToken,
		"languageCode":         "en", // Ticketmaster requere
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.baseURL+"/v1/places:autocomplete", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", s.apiKey)

	var resp models.GoogleAutocompleteResponse
	if err := utils.DoJSON(s.client, req, &resp); err != nil {
		return nil, err
	}
	return resp.ToSuggestions(), nil
}

// PlaceDetails : return   city and country code from placeID
func (s *LocationService) PlaceDetails(ctx context.Context, placeID, sessionToken string) (*models.City, error) {
	if !placeIDPattern.MatchString(placeID) {
		return nil, fmt.Errorf("%w: bad place id", ErrInvalidInput)
	}
	if !sessionTokenPattern.MatchString(sessionToken) {
		return nil, fmt.Errorf("%w: bad session token", ErrInvalidInput)
	}

	endpoint := fmt.Sprintf("%s/v1/places/%s?sessionToken=%s&languageCode=en",
		s.baseURL, url.PathEscape(placeID), url.QueryEscape(sessionToken))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Goog-Api-Key", s.apiKey)
	req.Header.Set("X-Goog-FieldMask", "addressComponents")

	var resp models.GooglePlaceDetailsResponse
	if err := utils.DoJSON(s.client, req, &resp); err != nil {
		return nil, err
	}
    return resp.ToCity(placeID)
}