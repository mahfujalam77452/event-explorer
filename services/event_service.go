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

// categoryResult represents the result produced by a goroutine.
// This result is sent through a channel.
type categoryResult struct {
	index    int    // Position in the Categories list, used to preserve the original order.
	category string
	events   []models.Event
	err      error
}

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
	cache *EventCache
}

func NewEventService(client *http.Client, baseURL, apiKey string, perCategory int,cache *EventCache) *EventService {
	return &EventService{
		client:      client,
		baseURL:     baseURL,
		apiKey:      apiKey,
		perCategory: perCategory,
		cache: cache,
	}
}

// GetListing fetches both categories concurrently.
// If one category fails, that section contains an error while the other section remains available.
func (s *EventService) GetListing(ctx context.Context, city, country string) []models.Section {
	
	start := time.Now()

	// Buffered channel with a capacity equal to the number of goroutines.
	// This prevents goroutines from getting blocked while sending their results
	// and helps avoid goroutine leaks.
	results := make(chan categoryResult, len(Categories))

	// Step 1: Start all goroutines first without waiting for any of them to finish.
	for i, category := range Categories {
		go s.fetchCategoryAsync(ctx, i, category, city, country, results)
	}
	log.Printf("[events] started %d goroutines for %s,%s", len(Categories), city, country)

	// Step 2: Collect exactly one result from the channel for each category.
	sections := make([]models.Section, len(Categories))
	for range Categories {
		r := <-results

		section := models.Section{Category: r.category}
		if r.err != nil {
			log.Printf("[events] %s for %s,%s failed: %v", r.category, city, country, r.err)
			section.Error = friendlyError(r.category, r.err)
		} else {
			section.Events = r.events
		}

		// Store the section at its original index, regardless of the order
		// in which the goroutines finish.
		sections[r.index] = section
	}

	log.Printf("[events] listing %s,%s built in %s", city, country, time.Since(start).Round(time.Millisecond))
	return sections
}

// fetchCategoryAsync runs inside a goroutine and always sends exactly one result to the channel.
func (s *EventService) fetchCategoryAsync(
	ctx context.Context, index int, category, city, country string, out chan<- categoryResult,
) {
	res := categoryResult{index: index, category: category}

	// Recover from any panic so that it does not crash the entire server.
	// The deferred function also guarantees that a result is sent to the channel,
	// preventing the receiver from waiting indefinitely.
	defer func() {
		if r := recover(); r != nil {
			res.events = nil
			res.err = fmt.Errorf("panic in %s worker: %v", category, r)
		}
		out <- res
	}()

	begin := time.Now()
	log.Printf("[events] %s: fetching...", category)

	res.events, res.err = s.getCategory(ctx, city, country, category)

	log.Printf("[events] %s: done in %s", category, time.Since(begin).Round(time.Millisecond))
}
// getCategory first checks the cache.
// If the data is not found, it fetches the events from Ticketmaster and stores
// the successful result in the cache.
// Only successful results are cached; errors are never cached.
func (s *EventService) getCategory(ctx context.Context, city, country, category string) ([]models.Event, error) {
	key := CacheKey(city, country, category)

	if events, ok := s.cache.Get(key); ok {
		log.Printf("[cache] HIT   key=%q items=%d", key, len(events))
		return events, nil
	}
	log.Printf("[cache] MISS  key=%q", key)

	events, err := s.fetchCategory(ctx, city, country, category)
	if err != nil {
		return nil, err // Do not cache failed results.
	}

	s.cache.Set(key, events)
	log.Printf("[cache] STORE key=%q items=%d", key, len(events))
	return events, nil
}

// ClearCache invalidates the entire cache and returns the number of entries removed.
func (s *EventService) ClearCache() int {
	n := s.cache.Clear()
	log.Printf("[cache] CLEARED %d entries", n)
	return n
}
// NormalizeCategory converts "music" or " SPORTS " to the correct form
// ("Music" or "Sports").
// It returns false for any value other than Music or Sports.
func NormalizeCategory(input string) (string, bool) {
	input = strings.TrimSpace(input)
	for _, c := range Categories {
		if strings.EqualFold(c, input) {
			return c, true
		}
	}
	return "", false
}

// InvalidateCategory removes only one cache entry identified by
// city, country, and category.
// It returns the key and whether an entry was actually deleted.
func (s *EventService) InvalidateCategory(city, country, category string) (string, bool, error) {
	city = strings.TrimSpace(city)
	country = strings.ToUpper(strings.TrimSpace(country))

	if err := ValidateCityQuery(city, country); err != nil {
		return "", false, err
	}
	canonical, ok := NormalizeCategory(category)
	if !ok {
		return "", false, fmt.Errorf("%w: category must be one of %s",
			ErrInvalidInput, strings.Join(Categories, ", "))
	}

	key := CacheKey(city, country, canonical)
	deleted := s.cache.Delete(key)
	log.Printf("[cache] DELETE key=%q found=%t", key, deleted)
	return key, deleted, nil
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