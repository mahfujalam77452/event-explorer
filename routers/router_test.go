package routers

import (
	"net/http/httptest"
	"testing"

	"github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/context"
)

func found(method, path string) bool {
	ctx := context.NewContext()
	ctx.Reset(httptest.NewRecorder(), httptest.NewRequest(method, path, nil))
	_, ok := web.BeeApp.Handlers.FindRouter(ctx)
	return ok
}

func TestAllRoutesAreRegistered(t *testing.T) {
	routes := []struct{ method, path string }{
		{"GET", "/"},
		{"GET", "/events"},
		{"GET", "/events/G5vYZ9abc123"},
		{"GET", "/api/locations/autocomplete"},
		{"GET", "/api/locations/ChIJpTvG15DL1IkRd8S0KlBVNTI"},
		{"GET", "/redirect/G5vYZ9abc123"},
		{"GET", "/admin/cache/clear"},
		{"POST", "/admin/cache/clear"},
				{"GET", "/admin/cache/invalidate"},
		{"POST", "/admin/cache/invalidate"},
		{"DELETE", "/admin/cache/invalidate"},
	}
	for _, r := range routes {
		if !found(r.method, r.path) {
			t.Errorf("%s %s is not registered", r.method, r.path)
		}
	}
}

func TestUnknownRoutesAreNotRegistered(t *testing.T) {
	for _, path := range []string{"/nothing", "/api", "/redirect", "/admin"} {
		if found("GET", path) {
			t.Errorf("GET %s should not match any route", path)
		}
	}
}