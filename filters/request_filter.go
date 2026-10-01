package filters

import (
	"log"
	"time"

	beego "github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/context"
)

const startKey = "request_start"

// Register enables all application filters.
// It is called once from routers/router.go.
func Register() {
	beego.InsertFilter("/*", beego.BeforeRouter, beforeRouter)
	beego.InsertFilter("/*", beego.FinishRouter, finishRouter)
}

// beforeRouter runs before routing.
// It records the request start time and sets security headers.
func beforeRouter(ctx *context.Context) {
	ctx.Input.SetData(startKey, time.Now())

	ctx.Output.Header("X-Content-Type-Options", "nosniff")
	ctx.Output.Header("X-Frame-Options", "DENY")
	ctx.Output.Header("Referrer-Policy", "strict-origin-when-cross-origin")
}

// finishRouter runs after the response is completed.
// It logs the HTTP method, path, status code, and request duration.
// The query string is intentionally excluded to prevent sensitive data,
// such as session tokens, from being written to logs.
func finishRouter(ctx *context.Context) {
	start, ok := ctx.Input.GetData(startKey).(time.Time)
	if !ok {
		return
	}

	status := ctx.ResponseWriter.Status
	if status == 0 {
		status = 200
	}

	log.Printf("[http] %s %s -> %d (%s)",
		ctx.Request.Method, ctx.Request.URL.Path, status,
		time.Since(start).Round(time.Millisecond))
}