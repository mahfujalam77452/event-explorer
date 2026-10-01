package utils

import (
	"fmt"
	"os"
	"strings"
	"time"
	beego "github.com/beego/beego/v2/server/web"
	"github.com/joho/godotenv"
)

type Config struct {
	GoogleAPIKey string
	TicketmasterAPIKey string

	GoogleBaseURL string
	TicketmasterBaseURL string

	HTTPTimeout time.Duration
	EventsPerCategory int

	ApprovedTicketHosts []string
}

var Cfg *Config

func LoadConfig() (*Config,error) {
	_ = godotenv.Load()

	cfg := &Config{
	GoogleAPIKey : strings.TrimSpace(os.Getenv("GOOGLE_API_KEY")),
	TicketmasterAPIKey: strings.TrimSpace(os.Getenv("TICKETMASTER_API_KEY")),

	GoogleBaseURL : strings.TrimSpace(
		beego.AppConfig.DefaultString("google_base_url","https://places.googleapis.com")),
	TicketmasterBaseURL : strings.TrimSpace(
		beego.AppConfig.DefaultString("ticketmaster_base_url","https://app.ticketmaster.com/discovery/v2")),

	HTTPTimeout : time.Duration(
		beego.AppConfig.DefaultInt("http_timeout_seconds", 10))*time.Second,
	EventsPerCategory : 6,
	}

	var missing []string
	if cfg.GoogleAPIKey == "" {
		missing = append(missing, "GOOGLE_API_KEY")
	}
	if cfg.TicketmasterAPIKey == "" {
		missing = append(missing, "TICKETMASTER_API_KEY")
	}
		cfg.ApprovedTicketHosts = splitCSV(beego.AppConfig.DefaultString("approved_ticket_hosts", ""))

	if len(missing) > 0 {
		return cfg,fmt.Errorf("Missing environment variables : %s",strings.Join(missing,", "))
	}

	return  cfg,nil
}

// splitCSV converts "a, b,c" into ["a", "b", "c"] and ignores empty parts.
func splitCSV(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.ToLower(strings.TrimSpace(part)); p != "" {
			out = append(out, p)
		}
	}
	return out
}