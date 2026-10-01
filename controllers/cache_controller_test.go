package controllers_test

import "testing"

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
		t.Fatalf(
			"POST on an empty cache should clear 0 entries, got %d: %s",
			rec.Code,
			rec.Body.String(),
		)
	}
}