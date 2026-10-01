package filters

import (
	"bytes"
	"log"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/beego/beego/v2/server/web/context"
)

func newCtx(target string) (*context.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	ctx := context.NewContext()
	ctx.Reset(rec, httptest.NewRequest("GET", target, nil))
	return ctx, rec
}

func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	return &buf
}

func TestBeforeRouterSetsSecurityHeadersAndStartTime(t *testing.T) {
	ctx, rec := newCtx("/events")
	beforeRouter(ctx)

	want := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}
	for name, value := range want {
		if got := rec.Header().Get(name); got != value {
			t.Errorf("header %s: want %q, got %q", name, value, got)
		}
	}
	if _, ok := ctx.Input.GetData(startKey).(time.Time); !ok {
		t.Fatal("start time should be stored for finishRouter")
	}
}

func TestFinishRouterLogsMethodPathStatusButNotQuery(t *testing.T) {
	logs := captureLog(t)
	ctx, _ := newCtx("/events?sessionToken=SECRET123")

	beforeRouter(ctx)
	finishRouter(ctx)

	out := logs.String()
	if !strings.Contains(out, "[http] GET /events -> 200") {
		t.Fatalf("unexpected log: %q", out)
	}
	if strings.Contains(out, "SECRET123") {
		t.Fatalf("the query string must never be logged: %q", out)
	}
}

func TestFinishRouterLogsRealStatus(t *testing.T) {
	logs := captureLog(t)
	ctx, _ := newCtx("/missing")

	beforeRouter(ctx)
	ctx.ResponseWriter.Status = 404
	finishRouter(ctx)

	if !strings.Contains(logs.String(), "-> 404") {
		t.Fatalf("unexpected log: %q", logs.String())
	}
}

func TestFinishRouterWithoutStartLogsNothing(t *testing.T) {
	logs := captureLog(t)
	ctx, _ := newCtx("/events")

	finishRouter(ctx)

	if logs.Len() != 0 {
		t.Fatalf("nothing should be logged, got %q", logs.String())
	}
}

func TestRegisterDoesNotPanic(t *testing.T) {
	Register()
}