package controllers

// PageController: সব HTML (SSR) পেজ।
type PageController struct {
	BaseController
}

// GET /
func (c *PageController) Home() {
	c.RenderPage("home.tpl", "Find events in your city")
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