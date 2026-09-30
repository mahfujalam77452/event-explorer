package controllers

import (
	beego "github.com/beego/beego/v2/server/web"
)

// APIController: শুধু JSON ফেরত দেয়, কোনো HTML না।
type APIController struct {
	beego.Controller
}

// GET /api/locations/autocomplete?input=tor&sessionToken=xxx
func (c *APIController) Autocomplete() {
	c.Data["json"] = map[string]interface{}{
		"stub":  true,
		"input": c.GetString("input"),
	}
	_ = c.ServeJSON()
}

// GET /api/locations/:placeId?sessionToken=xxx
func (c *APIController) PlaceDetails() {
	c.Data["json"] = map[string]interface{}{
		"stub":    true,
		"placeId": c.Ctx.Input.Param(":placeId"),
	}
	_ = c.ServeJSON()
}