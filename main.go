package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/quiknode-labs/chinchillaproxy/proxy"
)

func main() {
	engine := gin.New()
	h := proxy.New()
	engine.POST("/*path", h.TranslateRequest)

	log.Print("Listening on port 8080 - routing to: ", h.Config.Upstream)
	engine.Run(":8080")
}
