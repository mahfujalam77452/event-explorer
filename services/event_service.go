package services

import (
	"context"
	"fmt"
	"log"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"event-explorer/models"
	"event-explorer/utils"
)

const (
	CategoryMusic  = "Music"
	CategorySports = "Sports"
)

// Categories: How we will the categories in listing page
var Categories = []string{CategoryMusic, CategorySports}

var countryCodePattern = regexp.MustCompile(`^[A-Za-z]{2}$`)

// Validate city and countryCode 
func ValidateCityQuery(city, country string) error {
	city = strings.TrimSpace(city)
	if city == "" || utf8.RuneCountInString(city) > 100 {
		return fmt.Errorf("%w: city", ErrInvalidInput)
	}
	if !countryCodePattern.MatchString(country) {
		return fmt.Errorf("%w: countryCode", ErrInvalidInput)
	}
	return nil
}

// EventService talk with Ticketmaster Discovery API 
type EventService struct {
	client      *http.Client
	baseURL     string
	apiKey      string
	perCategory int
}

func NewEventService(client *http.Client, baseURL, apiKey string, perCategory int) *EventService {
	return &EventService{
		client:      client,
		baseURL:     baseURL,
		apiKey:      apiKey,
		perCategory: perCategory,
	}
}

// GetListing returns sections for two categories.
// If one category fails, that section contains an Error while the other remains intact.

func (s *EventService) GetListing(ctx context.Context, city, country string) []models.Section {
	start := time.Now()
	sections := make([]models.Section, 0, len(Categories))

	for _, category := range Categories {
		section := models.Section{Category: category}

		events, err := s.fetchCategory(ctx, city, country, category)
		if err != nil {
			log.Printf("[events] %s for %s,%s failed: %v", category, city, country, err)
			section.Error = friendlyError(category, err)
		} else {
			section.Events = events
		}
		sections = append(sections, section)
	}

	log.Printf("[events] listing %s,%s built in %s", city, country, time.Since(start).Round(time.Millisecond))
	return sections
}

// fetchCategory sends a single request to Ticketmaster for a category.
func (s *EventService) fetchCategory(ctx context.Context, city, country, category string) ([]models.Event, error) {
	query := url.Values{}
	query.Set("apikey", s.apiKey)
	query.Set("city", city)
	query.Set("countryCode", country)
	query.Set("classificationName", category)
	query.Set("size", strconv.Itoa(s.perCategory))

	endpoint := s.baseURL + "/events.json?" + query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var resp models.TMEventsResponse
	if err := utils.DoJSON(s.client, req, &resp); err != nil {
		return nil, err
	}
	return resp.ToEvents(s.perCategory), nil
}
// GetEvent fetches the complete details of an event directly from Ticketmaster
// using GET /events/{id}.json.
// It fetches the event by ID instead of relying on the listing, which supports direct links.
func (s *EventService) GetEvent(ctx context.Context, id string) (*models.Event, error) {
	// Validate the ID before using it in the URL to prevent path injection.
	if !models.IsValidEventID(id) {
		return nil, models.ErrInvalidEventID
	}

	query := url.Values{}
	query.Set("apikey", s.apiKey)

	endpoint := s.baseURL + "/events/" + url.PathEscape(id) + ".json?" + query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var raw models.TMEvent
	if err := utils.DoJSON(s.client, req, &raw); err != nil {
		var httpErr *utils.HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			return nil, models.ErrEventNotFound
		}
		return nil, err
	}

	event := raw.ToEvent()
	if err := event.Validate(); err != nil {
		// Ticketmaster responded, but the event data is not usable.
		return nil, models.ErrEventNotFound
	}
	return &event, nil
}
// friendlyError for user
func friendlyError(category string, err error) string {
	if utils.IsTimeout(err) {
		return fmt.Sprintf("%s events took too long to load. Please try again.", category)
	}
	return fmt.Sprintf("%s events are unavailable right now. Please try again later.", category)
}