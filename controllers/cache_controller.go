package controllers

import (
	"net/http"

	"event-explorer/services"

	beego "github.com/beego/beego/v2/server/web"
)

// CacheController handles endpoints related to cache management.
type CacheController struct {
	beego.Controller
}

// GET/POST /admin/cache/clear
func (c *CacheController) Clear() {
	cleared := services.Events.ClearCache()

	c.Ctx.Output.SetStatus(http.StatusOK)
	c.Ctx.Output.Header("Cache-Control", "no-store")
	c.Data["json"] = map[string]interface{}{
		"message": "cache cleared",
		"cleared": cleared,
	}
	_ = c.ServeJSON()
}