package utils

import (
	"errors"
	"testing"
)

var testApprovedHosts = []string{"ticketmaster.com", "ticketmaster.ca", "livenation.com"}

func TestValidateTicketURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		allowed bool
	}{
		// ---- Allowed URLs ----
		{"exact approved host", "https://ticketmaster.com/event/1", true},
		{"subdomain of approved host", "https://www.ticketmaster.ca/event/123", true},
		{"deep subdomain", "https://tickets.eu.livenation.com/e/9", true},
		{"uppercase host", "https://WWW.TICKETMASTER.COM/e", true},
		{"trailing dot in host", "https://www.ticketmaster.com./e", true},
		{"explicit 443 port", "https://www.ticketmaster.com:443/e", true},
		{"surrounding whitespace", "  https://www.ticketmaster.com/e  ", true},
		{"query string kept", "https://www.ticketmaster.com/e?tm_link=abc", true},

		// ---- Rejected URLs ----
		{"empty", "", false},
		{"only spaces", "   ", false},
		{"http instead of https", "http://www.ticketmaster.com/e", false},
		{"javascript scheme", "javascript:alert(1)", false},
		{"data scheme", "data:text/html,<script>alert(1)</script>", false},
		{"ftp scheme", "ftp://www.ticketmaster.com/e", false},
		{"scheme-relative url", "//evil.com/x", false},
		{"not a url", "not a url", false},
		{"missing host", "https:///path", false},
		{"unapproved domain", "https://evil.com/e", false},
		{"suffix without dot", "https://evilticketmaster.com/e", false},
		{"approved name as subdomain of attacker", "https://ticketmaster.com.evil.com/e", false},
		{"approved name only in path", "https://evil.com/ticketmaster.com", false},
		{"approved name only in query", "https://evil.com/?u=ticketmaster.com", false},
		{"credentials trick", "https://ticketmaster.com@evil.com/", false},
		{"credentials on approved host", "https://user:pw@www.ticketmaster.com/e", false},
		{"unusual port", "https://www.ticketmaster.com:8443/e", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ValidateTicketURL(tc.url, testApprovedHosts)

			if tc.allowed {
				if err != nil {
					t.Fatalf("expected %q to be allowed, got error: %v", tc.url, err)
				}
				if got == "" {
					t.Fatalf("expected a cleaned url for %q, got empty string", tc.url)
				}
				return
			}

			if err == nil {
				t.Fatalf("expected %q to be rejected, but it was allowed (%q)", tc.url, got)
			}
			if !errors.Is(err, ErrUnapprovedTicketURL) {
				t.Fatalf("expected ErrUnapprovedTicketURL, got: %v", err)
			}
			if got != "" {
				t.Fatalf("a rejected url must return an empty string, got %q", got)
			}
		})
	}
}

func TestValidateTicketURL_EmptyApprovedListRejectsEverything(t *testing.T) {
	if _, err := ValidateTicketURL("https://www.ticketmaster.com/e", nil); err == nil {
		t.Fatal("with no approved hosts every url must be rejected")
	}
}

func TestValidateTicketURL_ReturnsSameUrlWhenValid(t *testing.T) {
	const in = "https://www.ticketmaster.ca/event/abc?x=1"

	got, err := ValidateTicketURL(in, testApprovedHosts)
	if err != nil {
		t.Fatal(err)
	}

	if got != in {
		t.Fatalf("want %q, got %q", in, got)
	}
}