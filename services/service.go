package services

import "event-explorer/utils"

//calling each service by calling init() from main
var (
	Location *LocationService
	Events   *EventService
)

func Init(cfg *utils.Config) {
	client := utils.NewHTTPClient(cfg.HTTPTimeout)

	Location = NewLocationService(client, cfg.GoogleBaseURL, cfg.GoogleAPIKey)
	Events = NewEventService(client, cfg.TicketmasterBaseURL, cfg.TicketmasterAPIKey, cfg.EventsPerCategory)
}