package controllers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"

	"event-explorer/models"
	"event-explorer/services"
	"event-explorer/utils"
)

// PageController handles all server-side rendered HTML pages.
type PageController struct {
	BaseController
}

// GET /
func (c *PageController) Home() {
	c.RenderPage("home.tpl", "Find events in your city")
}

// GET /events?city=Toronto&countryCode=CA
func (c *PageController) Listing() {
	city := strings.TrimSpace(c.GetString("city"))
	country := strings.ToUpper(strings.TrimSpace(c.GetString("countryCode")))

	if err := services.ValidateCityQuery(city, country); err != nil {
		c.Ctx.Output.SetStatus(http.StatusBadRequest)
		c.RenderPage("listing.tpl", "Events")
		c.Data["PageError"] = "Please choose a city first to see its events."
		return
	}

	sections := services.Events.GetListing(c.Ctx.Request.Context(), city, country)

	c.RenderPage("listing.tpl", "Events in "+city)
	c.Data["City"] = city
	c.Data["CountryCode"] = country
	c.Data["Sections"] = sections
}

// GET /events/:eventId
func (c *PageController) Details() {
	eventID := c.Ctx.Input.Param(":eventId")

	// Build the "Back" link based on the previous listing; use the home page for direct links.
	c.Data["BackURL"], c.Data["BackLabel"] = c.backLink()

	event, err := services.Events.GetEvent(c.Ctx.Request.Context(), eventID)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return // The user left the page before the request completed.
		}

		status, message := detailsError(err)
		log.Printf("[details] event %q failed (status %d): %v", eventID, status, err)

		c.Ctx.Output.SetStatus(status)
		c.RenderPage("details.tpl", "Event")
		c.Data["PageError"] = message
		return
	}

	c.RenderPage("details.tpl", event.Name)
	c.Data["Event"] = event
}

// backLink builds a URL back to the listing page when city and countryCode are provided in the query.
func (c *PageController) backLink() (string, string) {
	city := strings.TrimSpace(c.GetString("city"))
	country := strings.ToUpper(strings.TrimSpace(c.GetString("countryCode")))

	if services.ValidateCityQuery(city, country) == nil {
		q := url.Values{}
		q.Set("city", city)
		q.Set("countryCode", country)
		return "/events?" + q.Encode(), "Back to results"
	}
	return "/", "Back to search"
}

// detailsError maps errors to the appropriate HTTP status and user-friendly message.
func detailsError(err error) (int, string) {
	switch {
	case errors.Is(err, models.ErrInvalidEventID), errors.Is(err, models.ErrEventNotFound):
		return http.StatusNotFound, "We could not find this event. It may have been removed or the link is incorrect."
	case utils.IsTimeout(err):
		return http.StatusGatewayTimeout, "The event details took too long to load. Please try again."
	default:
		return http.StatusBadGateway, "Event details are unavailable right now. Please try again later."
	}
}