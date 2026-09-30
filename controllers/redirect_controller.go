package controllers

import (
	beego "github.com/beego/beego/v2/server/web"
)

// RedirectController: View Tickets বাটনের backend action (পেজ না)।
type RedirectController struct {
	beego.Controller
}

// GET /redirect/:eventId
func (c *RedirectController) Redirect() {
	c.Ctx.WriteString("redirect (stub): eventId=" + c.Ctx.Input.Param(":eventId"))
}