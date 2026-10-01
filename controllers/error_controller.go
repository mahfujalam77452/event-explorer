package controllers

import (
	"time"

	beego "github.com/beego/beego/v2/server/web"
)

// ErrorController handles Beego's built-in error pages,
// such as invalid URLs, unsupported methods, and server errors.
type ErrorController struct {
	beego.Controller
}

func (c *ErrorController) show(heading, message string) {
	c.Data["SiteName"] = "Event Explorer"
	c.Data["Title"] = heading + " | Event Explorer"
	c.Data["CurrentPath"] = c.Ctx.Request.URL.Path
	c.Data["Year"] = time.Now().Year()
	c.Data["Heading"] = heading
	c.Data["Message"] = message
	c.TplName = "error.tpl"
}

// 404: The requested URL does not match any registered route.
func (c *ErrorController) Error404() {
	c.show("Page not found", "The page you are looking for does not exist or has moved.")
}

// 405: The requested HTTP method is not supported for this route.
func (c *ErrorController) Error405() {
	c.show("Action not allowed", "This address does not support that kind of request.")
}

// 500: An unexpected server error occurred.
func (c *ErrorController) Error500() {
	c.show("Something went wrong", "An unexpected error occurred on our side. Please try again in a moment.")
}