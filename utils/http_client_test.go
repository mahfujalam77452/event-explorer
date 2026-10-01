package utils

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDoJSON_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"name":"ok","count":3}`))
	}))
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodGet, ts.URL, nil)
	var out struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	if err := DoJSON(NewHTTPClient(2*time.Second), req, &out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Name != "ok" || out.Count != 3 {
		t.Fatalf("unexpected result: %+v", out)
	}
}

func TestDoJSON_NilOutIsAllowed(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodGet, ts.URL, nil)
	if err := DoJSON(NewHTTPClient(2*time.Second), req, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}



func TestDoJSON_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>not json</html>`))
	}))
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodGet, ts.URL, nil)
	var out map[string]interface{}
	if err := DoJSON(NewHTTPClient(2*time.Second), req, &out); err == nil {
		t.Fatal("expected a JSON error, got nil")
	}
}

func TestDoJSON_ErrorNeverLeaksApiKey(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := ts.URL
	ts.Close() // server off so the connection will fail

	req, _ := http.NewRequest(http.MethodGet, url+"/events.json?apikey=SECRET-KEY-123", nil)
	err := DoJSON(NewHTTPClient(time.Second), req, nil)
	if err == nil {
		t.Fatal("expected a connection error")
	}
	if strings.Contains(err.Error(), "SECRET-KEY-123") {
		t.Fatalf("api key leaked into error message: %v", err)
	}
}

func TestIsTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(2 * time.Second):
		case <-r.Context().Done():
		}
	}))
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodGet, ts.URL, nil)
	err := DoJSON(NewHTTPClient(50*time.Millisecond), req, nil)
	if err == nil {
		t.Fatal("expected a timeout error")
	}
	if !IsTimeout(err) {
		t.Fatalf("IsTimeout should be true for a client timeout, got false (%v)", err)
	}
	if IsTimeout(errors.New("plain error")) {
		t.Fatal("IsTimeout should be false for a plain error")
	}
	if IsTimeout(nil) {
		t.Fatal("IsTimeout(nil) should be false")
	}
}

func TestHTTPErrorMessage(t *testing.T) {
	plain := &HTTPError{StatusCode: 404}
	if got := plain.Error(); got != "upstream returned status 404" {
		t.Fatalf("unexpected message %q", got)
	}

	withBody := &HTTPError{StatusCode: 400, Body: "{\n  \"error\": {\n    \"message\": \"API key not valid\"\n  }\n}"}
	msg := withBody.Error()
	if !strings.Contains(msg, "status 400") || !strings.Contains(msg, "API key not valid") {
		t.Fatalf("the upstream reason should be in the message, got %q", msg)
	}
	if strings.Contains(msg, "\n") {
		t.Fatalf("the message must be a single log line, got %q", msg)
	}
}

