package controllers

import (
	"context"
	"errors"
	"log"
	"net/http"

	"event-explorer/models"
	"event-explorer/services"
	"event-explorer/utils"

	beego "github.com/beego/beego/v2/server/web"
)

// APIController: Just returning JSON
type APIController struct {
	beego.Controller
}

// GET /api/locations/autocomplete?input=tor&sessionToken=xxx
func (c *APIController) Autocomplete() {
	suggestions, err := services.Location.Autocomplete(
		c.Ctx.Request.Context(),
		c.GetString("input"),
		c.GetString("sessionToken"),
	)
	if err != nil {
		c.fail(err, "autocomplete")
		return
	}
	c.ok(map[string]interface{}{"suggestions": suggestions})
}

// GET /api/locations/:placeId?sessionToken=xxx
func (c *APIController) PlaceDetails() {
	city, err := services.Location.PlaceDetails(
		c.Ctx.Request.Context(),
		c.Ctx.Input.Param(":placeId"),
		c.GetString("sessionToken"),
	)
	if err != nil {
		c.fail(err, "place details")
		return
	}
	c.ok(city) // {"city":"Toronto","countryCode":"CA"}
}

// Sends a successful JSON response. Google data will not be cached, so use no-store.
func (c *APIController) ok(data interface{}) {
	c.Ctx.Output.Header("Cache-Control", "no-store")
	c.Data["json"] = data
	_ = c.ServeJSON()
}

// Converts the error into the appropriate HTTP status and a clear message.

func (c *APIController) fail(err error, action string) {
	status := http.StatusBadGateway
	message := "City search is unavailable right now. Please try again."

	switch {
	case errors.Is(err, context.Canceled):
		// user cencel the request himself
		return
	case errors.Is(err, services.ErrInvalidInput):
		status = http.StatusBadRequest
		message = "Invalid request."
	case errors.Is(err, models.ErrCityNotFound):
		status = http.StatusNotFound
		message = "Could not find a city for this place. Please choose another suggestion."
	case utils.IsTimeout(err):
		status = http.StatusGatewayTimeout
		message = "The city search took too long. Please try again."
	}

	log.Printf("[api] %s failed (status %d): %v", action, status, err)

	c.Ctx.Output.SetStatus(status)
	c.Ctx.Output.Header("Cache-Control", "no-store")
	c.Data["json"] = map[string]string{"error": message}
	_ = c.ServeJSON()
}