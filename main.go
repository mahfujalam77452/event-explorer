package main

import (
	"log"

	_ "event-explorer/routers"
	"event-explorer/utils"

	beego "github.com/beego/beego/v2/server/web"
)

func main() {

	cfg,err := utils.LoadConfig()

	if err != nil {
		log.Fatalf("config erro : %v ",err)
	}

	utils.Cfg = cfg

	log.Printf("config loaded: timeout=%s, events per category=%d",
		cfg.HTTPTimeout, cfg.EventsPerCategory)

	beego.Run()
}

