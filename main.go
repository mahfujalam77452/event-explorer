package main

import (
	_ "event-explorer/routers"
	beego "github.com/beego/beego/v2/server/web"
)

func main() {
	beego.Run()
}

