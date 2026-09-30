package controllers

import (
	"net/http"
	"strings"

	"event-explorer/services"
)

// PageController: All HTML (SSR) page
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
	c.Ctx.WriteString("details page (stub): eventId=" + c.Ctx.Input.Param(":eventId"))
}