package main

import (
	"flag"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	var port string
	flag.StringVar(&port, "port", ":12345", "Specify a port")
	flag.Parse()

	r := gin.Default()

	r.LoadHTMLGlob("templates/*")

	r.GET("/", showRegisterPage)
	r.POST("/register", registerHandler)

	r.GET("/query", showQueryPage)
	r.POST("/query", queryHandler)

	r.GET("/renew", showRenewPage)
	r.POST("/renew", renewHandler)

	go updatePasswordFile()

	if err := r.Run(port); err != nil {
		log.Fatal(err)
	}
}
