package services

import "event-explorer/utils"

// calling each service  by Init() from  main.go 
var Location *LocationService

func Init(cfg *utils.Config) {
	client := utils.NewHTTPClient(cfg.HTTPTimeout)
	Location = NewLocationService(client, cfg.GoogleBaseURL, cfg.GoogleAPIKey)
}