package controllers

import (
	"time"

	beego "github.com/beego/beego/v2/server/web"
)

// BaseController 
type BaseController struct {
	beego.Controller
}

func (c *BaseController) Prepare() {
	c.Data["SiteName"] = "Event Explorer"
	c.Data["CurrentPath"] = c.Ctx.Request.URL.Path
	c.Data["Year"] = time.Now().Year()
	c.Data["Title"] = "Event Explorer"
}


func (c *BaseController) RenderPage(tpl, title string) {
	c.TplName = tpl
	c.Data["Title"] = title + " | Event Explorer"
}