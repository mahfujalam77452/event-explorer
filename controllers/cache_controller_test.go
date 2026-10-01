package controllers_test

import (
	"strings"
	"testing"
)

func TestCacheClear_GetAndPost(t *testing.T) {
	resetFake(t)
	do("GET", "/events?city=Toronto&countryCode=CA") // Adds 2 entries to the cache.

	rec := do("GET", "/admin/cache/clear")
	if rec.Code != 200 {
		t.Fatalf("want 200, got %d", rec.Code)
	}
	body := decode(t, rec)
	if body["cleared"] != float64(2) || body["message"] != "cache cleared" {
		t.Fatalf("unexpected body: %v", body)
	}

	rec = do("POST", "/admin/cache/clear")
	if rec.Code != 200 || decode(t, rec)["cleared"] != float64(0) {
		t.Fatalf("POST on an empty cache should clear 0 entries, got %d: %s", rec.Code, rec.Body.String())
	}
}

// Returns the number of remaining cache entries by clearing the cache and checking the count.
func entriesLeft(t *testing.T) float64 {
	t.Helper()
	return decode(t, do("POST", "/admin/cache/clear"))["cleared"].(float64)
}

func TestCacheInvalidate_DeletesOnlyOneCategory(t *testing.T) {
	resetFake(t)
	do("GET", "/events?city=Toronto&countryCode=CA") // Adds toronto|CA|music and sports entries.

	rec := do("DELETE", "/admin/cache/invalidate?city=Toronto&countryCode=CA&category=Music")
	if rec.Code != 200 {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
	body := decode(t, rec)
	if body["deleted"] != true || body["key"] != "toronto|CA|music" || body["message"] != "cache entry deleted" {
		t.Fatalf("unexpected body: %v", body)
	}

	if left := entriesLeft(t); left != 1 {
		t.Fatalf("only the sports entry should be left, got %v entries", left)
	}
}

func TestCacheInvalidate_AcceptsGetPostAndDelete(t *testing.T) {
	for _, method := range []string{"GET", "POST", "DELETE"} {
		t.Run(method, func(t *testing.T) {
			resetFake(t)
			do("GET", "/events?city=Toronto&countryCode=CA")

			rec := do(method, "/admin/cache/invalidate?city=Toronto&countryCode=CA&category=Sports")
			if rec.Code != 200 || decode(t, rec)["deleted"] != true {
				t.Fatalf("%s: want 200 with deleted=true, got %d: %s", method, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestCacheInvalidate_IsCaseInsensitive(t *testing.T) {
	resetFake(t)
	do("GET", "/events?city=Toronto&countryCode=CA")

	rec := do("GET", "/admin/cache/invalidate?city=TORONTO&countryCode=ca&category=sports")
	body := decode(t, rec)
	if rec.Code != 200 || body["deleted"] != true || body["key"] != "toronto|CA|sports" {
		t.Fatalf("want a case-insensitive match, got %d: %v", rec.Code, body)
	}
}

func TestCacheInvalidate_NothingToDeleteIsNotAnError(t *testing.T) {
	resetFake(t)

	rec := do("GET", "/admin/cache/invalidate?city=Toronto&countryCode=CA&category=Music")
	body := decode(t, rec)
	if rec.Code != 200 || body["deleted"] != false {
		t.Fatalf("want 200 with deleted=false, got %d: %v", rec.Code, body)
	}
	if !strings.Contains(body["message"].(string), "no cache entry") {
		t.Fatalf("unexpected message: %v", body["message"])
	}

	// Deleting the same key again should produce the same result.
	do("GET", "/events?city=Toronto&countryCode=CA")
	do("DELETE", "/admin/cache/invalidate?city=Toronto&countryCode=CA&category=Music")
	body = decode(t, do("DELETE", "/admin/cache/invalidate?city=Toronto&countryCode=CA&category=Music"))
	if body["deleted"] != false {
		t.Fatalf("a second delete of the same key should report deleted=false, got %v", body)
	}
}

func TestCacheInvalidate_LeavesOtherCitiesAlone(t *testing.T) {
	resetFake(t)
	do("GET", "/events?city=Toronto&countryCode=CA")
	do("GET", "/events?city=London&countryCode=GB") // Adds 4 entries in total.

	do("DELETE", "/admin/cache/invalidate?city=Toronto&countryCode=CA&category=Music")

	if left := entriesLeft(t); left != 3 {
		t.Fatalf("want 3 entries left (1 deleted), got %v", left)
	}
}

func TestCacheInvalidate_RejectsInvalidInput(t *testing.T) {
	targets := map[string]string{
		"nothing at all":   "/admin/cache/invalidate",
		"missing category": "/admin/cache/invalidate?city=Toronto&countryCode=CA",
		"unknown category": "/admin/cache/invalidate?city=Toronto&countryCode=CA&category=Theatre",
		"missing city":     "/admin/cache/invalidate?countryCode=CA&category=Music",
		"bad country":      "/admin/cache/invalidate?city=Toronto&countryCode=Canada&category=Music",
	}

	for name, target := range targets {
		t.Run(name, func(t *testing.T) {
			resetFake(t)
			do("GET", "/events?city=Toronto&countryCode=CA")

			rec := do("DELETE", target)
			if rec.Code != 400 {
				t.Fatalf("want 400, got %d: %s", rec.Code, rec.Body.String())
			}
			msg, _ := decode(t, rec)["error"].(string)
			if !strings.Contains(msg, "category") {
				t.Fatalf("the error should explain what is required, got %q", msg)
			}
			if left := entriesLeft(t); left != 2 {
				t.Fatalf("invalid input must not delete anything, got %v entries", left)
			}
		})
	}
}