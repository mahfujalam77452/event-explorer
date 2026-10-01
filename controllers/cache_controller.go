package controllers

import (
	"net/http"
	"strings"

	"event-explorer/services"

	beego "github.com/beego/beego/v2/server/web"
)

// CacheController handles endpoints for invalidating the cache.
type CacheController struct {
	beego.Controller
}

// GET/POST /admin/cache/clear
// Clears the entire cache.
func (c *CacheController) Clear() {
	cleared := services.Events.ClearCache()
	c.respond(http.StatusOK, map[string]interface{}{
		"message": "cache cleared",
		"cleared": cleared,
	})
}

// GET/POST/DELETE /admin/cache/invalidate?city=Toronto&countryCode=CA&category=Music
// Removes only one cache entry (city + country + category); the rest of the cache remains unchanged.
func (c *CacheController) Invalidate() {
	city := strings.TrimSpace(c.GetString("city"))
	country := strings.ToUpper(strings.TrimSpace(c.GetString("countryCode")))
	category := c.GetString("category")

	key, deleted, err := services.Events.InvalidateCategory(city, country, category)
	if err != nil {
		c.respond(http.StatusBadRequest, map[string]interface{}{
			"error": "city, countryCode and category (Music or Sports) are required.",
		})
		return
	}

	message := "cache entry deleted"
	if !deleted {
		message = "no cache entry found for this key"
	}
	c.respond(http.StatusOK, map[string]interface{}{
		"message": message,
		"key":     key,
		"deleted": deleted,
	})
}

// respond sends a JSON response with the given status.
// The response will never be cached.
func (c *CacheController) respond(status int, body interface{}) {
	c.Ctx.Output.SetStatus(status)
	c.Ctx.Output.Header("Cache-Control", "no-store")
	c.Data["json"] = body
	_ = c.ServeJSON()
}