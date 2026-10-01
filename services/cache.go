package services

import (
	"strings"
	"sync"

	"event-explorer/models"
)

// EventCache is a shared in-memory map.
// Since multiple goroutines and requests may access it concurrently,
// all access is protected by a mutex.
type EventCache struct {
	mu    sync.RWMutex
	items map[string][]models.Event
}

func NewEventCache() *EventCache {
	return &EventCache{items: make(map[string][]models.Event)}
}

// CacheKey creates a unique key from the city, country, and category.
// The key is case-insensitive: "Toronto" and "toronto" produce the same key.
func CacheKey(city, country, category string) string {
	return strings.ToLower(strings.TrimSpace(city)) + "|" +
		strings.ToUpper(strings.TrimSpace(country)) + "|" +
		strings.ToLower(strings.TrimSpace(category))
}

// Get returns a copy of the event list and true if the key exists.
// RLock is used because this operation only reads data,
// allowing multiple goroutines to read concurrently.
func (c *EventCache) Get(key string) ([]models.Event, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	events, ok := c.items[key]
	if !ok {
		return nil, false
	}
	return copyEvents(events), true
}

// Set stores the event list in the cache.
// Lock is used because this operation modifies the shared map,
// preventing other goroutines from reading or writing at the same time.
func (c *EventCache) Set(key string, events []models.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = copyEvents(events)
}
//Delete cache category wish
func (c *EventCache) Delete(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, existed := c.items[key]
	delete(c.items, key)
	return existed
}
// Clear removes all entries from the cache and returns the number of entries removed.
func (c *EventCache) Clear() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	n := len(c.items)
	c.items = make(map[string][]models.Event)
	return n
}

// Len returns the current number of entries in the cache.
func (c *EventCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// copyEvents creates a separate copy of the slice so that external code
// cannot modify the data stored inside the cache.
func copyEvents(in []models.Event) []models.Event {
	out := make([]models.Event, len(in))
	copy(out, in)
	return out
}