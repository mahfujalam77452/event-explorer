package utils

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// ErrUnapprovedTicketURL indicates that the ticket URL is unsafe or not approved.
var ErrUnapprovedTicketURL = errors.New("ticket url is not allowed")

// ValidateTicketURL validates a URL before redirecting to it.
// It returns the cleaned URL if all validation checks pass; otherwise, it returns an error.
//
// Requirements:
//   - Must use HTTPS (rejects HTTP, javascript:, data:, etc.)
//   - Must not contain a username or password
//     (prevents tricks such as https://ticketmaster.com@evil.com)
//   - If a port is specified, it must be 443
//   - The hostname must match an approved domain or one of its subdomains
func ValidateTicketURL(raw string, approvedHosts []string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("%w: empty url", ErrUnapprovedTicketURL)
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("%w: unparsable url", ErrUnapprovedTicketURL)
	}

	if u.Scheme != "https" {
		return "", fmt.Errorf("%w: scheme %q is not https", ErrUnapprovedTicketURL, u.Scheme)
	}
	if u.User != nil {
		return "", fmt.Errorf("%w: credentials in url", ErrUnapprovedTicketURL)
	}
	if port := u.Port(); port != "" && port != "443" {
		return "", fmt.Errorf("%w: unexpected port %s", ErrUnapprovedTicketURL, port)
	}

	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	if host == "" {
		return "", fmt.Errorf("%w: empty host", ErrUnapprovedTicketURL)
	}
	if !isApprovedHost(host, approvedHosts) {
		return "", fmt.Errorf("%w: host %q is not approved", ErrUnapprovedTicketURL, host)
	}

	return u.String(), nil
}

// isApprovedHost returns true if the host exactly matches an approved domain
// or is a valid subdomain of an approved domain.
// A simple strings.HasSuffix check would also allow domains such as
// "evilticketmaster.com", so the suffix must be preceded by ".".
func isApprovedHost(host string, approved []string) bool {
	for _, a := range approved {
		a = strings.ToLower(strings.TrimSpace(a))
		if a == "" {
			continue
		}
		if host == a || strings.HasSuffix(host, "."+a) {
			return true
		}
	}
	return false
}