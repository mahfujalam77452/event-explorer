package controllers_test

import (
	"net/http/httptest"
	"strings"
	"testing"

	"event-explorer/controllers"

	"github.com/beego/beego/v2/server/web/context"
)

func TestUnknownURLShowsStyledNotFoundPage(t *testing.T) {
	resetFake(t)
	rec := do("GET", "/no-such-page")

	if rec.Code != 404 {
		t.Fatalf("want 404, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Page not found") || !strings.Contains(body, "Go to home") {
		t.Fatalf("want our own error page, got:\n%s", body)
	}
}

func TestErrorControllerPages(t *testing.T) {
	tests := []struct {
		name    string
		call    func(*controllers.ErrorController)
		heading string
	}{
		{"404", (*controllers.ErrorController).Error404, "Page not found"},
		{"405", (*controllers.ErrorController).Error405, "Action not allowed"},
		{"500", (*controllers.ErrorController).Error500, "Something went wrong"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.NewContext()
			ctx.Reset(httptest.NewRecorder(), httptest.NewRequest("GET", "/x", nil))

			c := &controllers.ErrorController{}
			c.Init(ctx, "ErrorController", "Error"+tc.name, nil)
			tc.call(c)

			if c.TplName != "error.tpl" {
				t.Fatalf("want error.tpl, got %q", c.TplName)
			}
			if c.Data["Heading"] != tc.heading {
				t.Fatalf("want heading %q, got %v", tc.heading, c.Data["Heading"])
			}
			if msg, _ := c.Data["Message"].(string); msg == "" {
				t.Fatal("the message must not be empty")
			}
		})
	}
}