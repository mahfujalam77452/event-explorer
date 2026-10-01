package models

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestIsValidEventID(t *testing.T) {
	valid := []string{"G5vYZ9abc123", "Z7r9jZ1A7-abc", "a_b-c", "1", strings.Repeat("a", 64)}
	invalid := []string{"", " ", "bad id", "a/b", "../x", "id?x=1", "id<script>", "é", strings.Repeat("a", 65)}

	for _, id := range valid {
		if !IsValidEventID(id) {
			t.Errorf("%q should be valid", id)
		}
	}
	for _, id := range invalid {
		if IsValidEventID(id) {
			t.Errorf("%q should be invalid", id)
		}
	}
}

func TestEventValidate(t *testing.T) {
	if err := (Event{ID: "abc123", Name: "Show"}).Validate(); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}
	if err := (Event{ID: "", Name: "Show"}).Validate(); !errors.Is(err, ErrInvalidEventID) {
		t.Fatalf("missing id: want ErrInvalidEventID, got %v", err)
	}
	if err := (Event{ID: "bad id!", Name: "Show"}).Validate(); !errors.Is(err, ErrInvalidEventID) {
		t.Fatalf("malformed id: want ErrInvalidEventID, got %v", err)
	}
	if err := (Event{ID: "abc123", Name: "   "}).Validate(); err == nil {
		t.Fatal("blank name should be rejected")
	}
}

func TestSectionStates(t *testing.T) {
	failed := Section{Category: "Music", Error: "boom"}
	if !failed.HasError() || failed.IsEmpty() || failed.HasEvents() {
		t.Fatalf("failed section flags wrong: %+v", failed)
	}

	empty := Section{Category: "Music"}
	if empty.HasError() || !empty.IsEmpty() || empty.HasEvents() {
		t.Fatalf("empty section flags wrong: %+v", empty)
	}

	full := Section{Category: "Music", Events: []Event{{ID: "a", Name: "A"}}}
	if full.HasError() || full.IsEmpty() || !full.HasEvents() {
		t.Fatalf("full section flags wrong: %+v", full)
	}
}

func TestEventTicketURLIsNeverSerialised(t *testing.T) {
	b, err := json.Marshal(Event{ID: "e1", Name: "n", TicketURL: "https://secret.example/tickets"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "secret.example") {
		t.Fatalf("ticket url leaked into JSON: %s", b)
	}
}