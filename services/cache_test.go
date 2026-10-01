package services

import (
	"fmt"
	"sync"
	"testing"

	"event-explorer/models"
)

func sampleEvents(prefix string, n int) []models.Event {
	out := make([]models.Event, 0, n)
	for i := 1; i <= n; i++ {
		out = append(out, models.Event{ID: fmt.Sprintf("%s%d", prefix, i), Name: fmt.Sprintf("%s %d", prefix, i)})
	}
	return out
}

func TestCacheKey(t *testing.T) {
	base := CacheKey("Toronto", "CA", "Music")

	same := []string{
		CacheKey("toronto", "ca", "music"),
		CacheKey("  TORONTO ", " Ca ", " MUSIC "),
	}
	for _, k := range same {
		if k != base {
			t.Errorf("key %q should equal %q (case/space must not matter)", k, base)
		}
	}

	different := []string{
		CacheKey("London", "CA", "Music"),
		CacheKey("Toronto", "US", "Music"),
		CacheKey("Toronto", "CA", "Sports"),
	}
	for _, k := range different {
		if k == base {
			t.Errorf("key %q must differ from %q", k, base)
		}
	}
}

func TestCache_MissThenHit(t *testing.T) {
	c := NewEventCache()
	key := CacheKey("Toronto", "CA", "Music")

	if _, ok := c.Get(key); ok {
		t.Fatal("empty cache must miss")
	}

	c.Set(key, sampleEvents("m", 6))

	got, ok := c.Get(key)
	if !ok {
		t.Fatal("expected a hit after Set")
	}
	if len(got) != 6 || got[0].ID != "m1" {
		t.Fatalf("unexpected cached events: %+v", got)
	}
	if c.Len() != 1 {
		t.Fatalf("want 1 entry, got %d", c.Len())
	}
}

func TestCache_KeysAreIndependent(t *testing.T) {
	c := NewEventCache()
	music := CacheKey("Toronto", "CA", "Music")
	sports := CacheKey("Toronto", "CA", "Sports")

	c.Set(music, sampleEvents("m", 2))

	if _, ok := c.Get(sports); ok {
		t.Fatal("a different category must not hit")
	}
	if _, ok := c.Get(CacheKey("Ottawa", "CA", "Music")); ok {
		t.Fatal("a different city must not hit")
	}
}

func TestCache_StoresIndependentCopies(t *testing.T) {
	c := NewEventCache()
	key := "k"

	original := sampleEvents("e", 2)
	c.Set(key, original)
	original[0].Name = "changed after Set"

	first, _ := c.Get(key)
	if first[0].Name == "changed after Set" {
		t.Fatal("cache must not share memory with the slice given to Set")
	}

	first[0].Name = "changed after Get"
	second, _ := c.Get(key)
	if second[0].Name == "changed after Get" {
		t.Fatal("cache must not share memory with the slice returned by Get")
	}
}

func TestCache_SetReplacesExistingEntry(t *testing.T) {
	c := NewEventCache()
	c.Set("k", sampleEvents("old", 2))
	c.Set("k", sampleEvents("new", 3))

	got, _ := c.Get("k")
	if len(got) != 3 || got[0].ID != "new1" {
		t.Fatalf("Set should replace the entry, got %+v", got)
	}
	if c.Len() != 1 {
		t.Fatalf("want 1 entry, got %d", c.Len())
	}
}

func TestCache_ClearInvalidatesEverything(t *testing.T) {
	c := NewEventCache()
	c.Set("a", sampleEvents("a", 1))
	c.Set("b", sampleEvents("b", 1))
	c.Set("c", sampleEvents("c", 1))

	if n := c.Clear(); n != 3 {
		t.Fatalf("Clear should report 3 removed entries, got %d", n)
	}
	for _, k := range []string{"a", "b", "c"} {
		if _, ok := c.Get(k); ok {
			t.Fatalf("key %q should be gone after Clear", k)
		}
	}
	if c.Len() != 0 {
		t.Fatalf("want empty cache, got %d entries", c.Len())
	}
	if n := c.Clear(); n != 0 {
		t.Fatalf("clearing an empty cache should report 0, got %d", n)
	}

	// cache is usable even after cleared
	c.Set("a", sampleEvents("a", 1))
	if _, ok := c.Get("a"); !ok {
		t.Fatal("cache should work again after Clear")
	}
}

// Running the test with -race will detect this data race if the code accesses shared data without a mutex.
func TestCache_ConcurrentAccessIsSafe(t *testing.T) {
	c := NewEventCache()
	var wg sync.WaitGroup

	for g := 0; g < 40; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				key := fmt.Sprintf("key-%d", i%7)
				switch (g + i) % 4 {
				case 0:
					c.Set(key, sampleEvents("e", 3))
				case 1:
					c.Get(key)
				case 2:
					c.Len()
				default:
					if i%50 == 0 {
						c.Clear()
					}
				}
			}
		}(g)
	}
	wg.Wait()
}