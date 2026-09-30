package controllers

import (
	beego "github.com/beego/beego/v2/server/web"
)

// CacheController: সব cache invalid করার endpoint।
type CacheController struct {
	beego.Controller
}

// GET/POST /admin/cache/clear
func (c *CacheController) Clear() {
	c.Data["json"] = map[string]interface{}{"stub": true}
	_ = c.ServeJSON()
}