package main

import (
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.LoadHTMLGlob("templates/*")

	r.GET("/", showRegisterPage)
	r.POST("/register", registerHandler)

	r.GET("/query", showQueryPage)
	r.POST("/query", queryHandler)

	r.GET("/renew", showRenewPage)
	r.POST("/renew", renewHandler)

	go updatePasswordFile()

	if err := r.Run(":12345"); err != nil {
		log.Fatal(err)
	}
}
