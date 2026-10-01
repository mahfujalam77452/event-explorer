package routers

import (
	"event-explorer/controllers"
	"event-explorer/filters"

	beego "github.com/beego/beego/v2/server/web"
)

func init() {
	
	// Filter ও error handler
	filters.Register()
	beego.ErrorController(&controllers.ErrorController{})
	
	
	//  Frontend page route 
	beego.Router("/", &controllers.PageController{}, "get:Home")
	beego.Router("/events", &controllers.PageController{}, "get:Listing")
	beego.Router("/events/:eventId", &controllers.PageController{}, "get:Details")

	//  Supporting API route (JSON) 
	beego.Router("/api/locations/autocomplete", &controllers.APIController{}, "get:Autocomplete")
	beego.Router("/api/locations/:placeId", &controllers.APIController{}, "get:PlaceDetails")

	// Ticket redirect (backend action)
	beego.Router("/redirect/:eventId", &controllers.RedirectController{}, "get:GoToTickets")

	//  Cache invalidation endpoint 
	beego.Router("/admin/cache/clear", &controllers.CacheController{}, "get,post:Clear")
	beego.Router("/admin/cache/invalidate", &controllers.CacheController{}, "get,post,delete:Invalidate")
}