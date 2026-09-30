package models

import (
	"errors"
	"regexp"
	"strings"
)

var (
	// ErrInvalidEventID: event ID 
	ErrInvalidEventID = errors.New("invalid event id")
	// ErrEventNotFound: Ticketmaster doesn't has this event
	ErrEventNotFound = errors.New("event not found")
)


var eventIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

//  URL has a valid id or not
func IsValidEventID(id string) bool {
	return eventIDPattern.MatchString(id)
}

// The infomations hold by an event
type Event struct {
	ID          string
	Name        string
	ImageURL    string
	Date        string 
	Venue       string 
	Location    string 
	Description string
	TicketURL   string `json:"-"` //not for HTML
}

// Validate ID and event without name is not granted
func (e Event) Validate() error {
	if !IsValidEventID(e.ID) {
		return ErrInvalidEventID
	}
	if strings.TrimSpace(e.Name) == "" {
		return errors.New("event has no name")
	}
	return nil
}

// listing page (music/sports)
type Section struct {
	Category string
	Events   []Event
	Error    string //message for user if applicable
}

func (s Section) HasError() bool  { return s.Error != "" }
func (s Section) HasEvents() bool { return len(s.Events) > 0 }

// IsEmpty: no error but no event 
func (s Section) IsEmpty() bool { return s.Error == "" && len(s.Events) == 0 }

// ListingData: all info for  listing.tpl 
type ListingData struct {
	City        string
	CountryCode string
	Sections    []Section
}