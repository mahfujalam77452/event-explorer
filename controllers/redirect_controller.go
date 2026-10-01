package controllers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"

	"event-explorer/services"
	"event-explorer/utils"
)

// RedirectController handles the backend action for the "View Tickets" button.
// It does not render an HTML page on success; it only sends a 302 redirect.
type RedirectController struct {
	BaseController
}

// GET /redirect/:eventId
func (c *RedirectController) GoToTickets() {
	// The only input is the eventId from the URL path.
	// Query parameters such as ?url=... are never read, so users cannot provide their own destination.
	eventID := c.Ctx.Input.Param(":eventId")

	// 1. Fetch the ticket URL from Ticketmaster on the server.
	event, err := services.Events.GetEvent(c.Ctx.Request.Context(), eventID)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		status, message := detailsError(err)
		log.Printf("[redirect] event %q lookup failed (status %d): %v", eventID, status, err)
		c.fail(status, message)
		return
	}

	// 2. Validate the approved HTTPS hostname.
	target, err := utils.ValidateTicketURL(event.TicketURL, utils.Cfg.ApprovedTicketHosts)
	if err != nil {
		log.Printf("[redirect] event %q rejected: %v", eventID, err)
		c.fail(http.StatusBadGateway, "Tickets for this event are not available right now.")
		return
	}

	// 3. Redirect to the validated ticket URL with a 302 status.
	host := ""
	if u, perr := url.Parse(target); perr == nil {
		host = u.Hostname()
	}
	log.Printf("[redirect] event %q -> %s", eventID, host)

	c.Ctx.Output.Header("Cache-Control", "no-store")
	c.Redirect(target, http.StatusFound) // 302
	c.StopRun()
}

// fail renders a clear error page for the user using the error section of details.tpl.
func (c *RedirectController) fail(status int, message string) {
	c.Ctx.Output.Header("Cache-Control", "no-store")

	c.Data["BackURL"] = "/"
	c.Data["BackLabel"] = "Back to search"
	c.Data["PageError"] = message
	c.RenderPageStatus(status, "details.tpl", "Tickets")
}