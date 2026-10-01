package controllers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	_ "event-explorer/routers" // route ও filter 
	"event-explorer/services"
	"event-explorer/utils"

	"github.com/beego/beego/v2/server/web"
)

// fake API 
var fake struct {
	googleDown atomic.Bool
	googleSlow atomic.Bool
	tmDown     atomic.Bool
	sportsDown atomic.Bool
	tmEmpty    atomic.Bool
}

func TestMain(m *testing.M) {
	
	_, file, _, _ := runtime.Caller(0)
	root, _ := filepath.Abs(filepath.Join(filepath.Dir(file), ".."))
	web.TestBeegoInit(root)

	google := httptest.NewServer(http.HandlerFunc(googleHandler))
	tm := httptest.NewServer(http.HandlerFunc(tmHandler))

	cfg := &utils.Config{
		GoogleAPIKey:        "test-google-key",
		TicketmasterAPIKey:  "test-tm-key",
		GoogleBaseURL:       google.URL,
		TicketmasterBaseURL: tm.URL,
		HTTPTimeout:         500 * time.Millisecond,
		EventsPerCategory:   6,
		ApprovedTicketHosts: []string{"ticketmaster.com"},
	}
	utils.Cfg = cfg
	services.Init(cfg)

	code := m.Run()
	google.Close()
	tm.Close()
	os.Exit(code)
}

// helper 

func do(method, target string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	rec := httptest.NewRecorder()
	web.BeeApp.Handlers.ServeHTTP(rec, req)
	return rec
}


func resetFake(t *testing.T) {
	t.Helper()
	reset := func() {
		fake.googleDown.Store(false)
		fake.googleSlow.Store(false)
		fake.tmDown.Store(false)
		fake.sportsDown.Store(false)
		fake.tmEmpty.Store(false)
		services.Events.ClearCache()
	}
	reset()
	t.Cleanup(reset)
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var out map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("response is not valid JSON: %v\nbody: %s", err, rec.Body.String())
	}
	return out
}

func writeJSON(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(body))
}

func writeValue(w http.ResponseWriter, v interface{}) {
	b, _ := json.Marshal(v)
	writeJSON(w, string(b))
}

// Fake Google 

func googleHandler(w http.ResponseWriter, r *http.Request) {
	if fake.googleDown.Load() {
		http.Error(w, `{"error":{"message":"API key not valid"}}`, http.StatusForbidden)
		return
	}
	if fake.googleSlow.Load() {
		select {
		case <-time.After(3 * time.Second):
		case <-r.Context().Done():
		}
		return
	}

	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/v1/places:autocomplete":
		writeJSON(w, `{"suggestions":[{"placePrediction":{"placeId":"ChIJtoronto",
			"text":{"text":"Toronto, ON, Canada"},
			"structuredFormat":{"mainText":{"text":"Toronto"},"secondaryText":{"text":"ON, Canada"}}}}]}`)

	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/places/"):
		if strings.HasSuffix(r.URL.Path, "ChIJnocity") {
			writeJSON(w, `{"addressComponents":[{"longText":"Canada","shortText":"CA","types":["country"]}]}`)
			return
		}
		writeJSON(w, `{"addressComponents":[
			{"longText":"Toronto","shortText":"Toronto","types":["locality"]},
			{"longText":"Canada","shortText":"CA","types":["country"]}]}`)

	default:
		http.NotFound(w, r)
	}
}

// Fake Ticketmaster 

func tmEvent(id, ticketURL string) map[string]interface{} {
	e := map[string]interface{}{
		"id":          id,
		"name":        "Show " + id,
		"description": "Description for " + id,
		"images": []map[string]interface{}{
			{"url": "https://img.example/" + id + ".jpg", "ratio": "16_9", "width": 640, "height": 360},
		},
		"dates": map[string]interface{}{
			"start": map[string]interface{}{"localDate": "2026-10-12", "localTime": "19:30:00"},
		},
		"_embedded": map[string]interface{}{
			"venues": []map[string]interface{}{
				{"name": "Test Arena", "city": map[string]interface{}{"name": "Toronto"}},
			},
		},
	}
	if ticketURL != "" {
		e["url"] = ticketURL
	}
	return e
}

func tmHandler(w http.ResponseWriter, r *http.Request) {
	if fake.tmDown.Load() {
		http.Error(w, "down", http.StatusInternalServerError)
		return
	}

	switch {
	case r.URL.Path == "/events.json":
		category := strings.ToLower(r.URL.Query().Get("classificationName"))
		if category == "sports" && fake.sportsDown.Load() {
			http.Error(w, "down", http.StatusInternalServerError)
			return
		}
		if fake.tmEmpty.Load() {
			writeJSON(w, `{"page":{"totalElements":0}}`) // "_embedded" নেই
			return
		}
		events := make([]map[string]interface{}, 0, 6)
		for i := 1; i <= 6; i++ {
			id := fmt.Sprintf("%s%d", category, i)
			events = append(events, tmEvent(id, "https://www.ticketmaster.com/event/"+id))
		}
		writeValue(w, map[string]interface{}{"_embedded": map[string]interface{}{"events": events}})

	case strings.HasPrefix(r.URL.Path, "/events/"):
		id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/events/"), ".json")
		switch id {
		case "missing":
			http.NotFound(w, r)
		case "evil1": 
			writeValue(w, tmEvent(id, "https://evil.com/tickets"))
		case "nourl1": 
			writeValue(w, tmEvent(id, ""))
		default:
			writeValue(w, tmEvent(id, "https://www.ticketmaster.com/event/"+id))
		}

	default:
		http.NotFound(w, r)
	}
}