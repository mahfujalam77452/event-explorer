package controllers

import (
	beego "github.com/beego/beego/v2/server/web"
)

// PageController: সব HTML (SSR) পেজ।
type PageController struct {
	beego.Controller
}

// GET /
func (c *PageController) Home() {
	c.Ctx.WriteString("home page (stub)")
}

// GET /events?city=Toronto&countryCode=CA
func (c *PageController) Listing() {
	c.Ctx.WriteString("listing page (stub): city=" + c.GetString("city") +
		", countryCode=" + c.GetString("countryCode"))
}

// GET /events/:eventId
func (c *PageController) Details() {
	c.Ctx.WriteString("details page (stub): eventId=" + c.Ctx.Input.Param(":eventId"))
}