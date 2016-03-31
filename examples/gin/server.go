package main

import (
	"net/http"

	"github.com/rs/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()
	router.GET("/", func(context *gin.Context) {
		context.JSON(http.StatusOK, gin.H{"hello":"world"})
	})
	// OPTIONS handler MUST BE set
	router.OPTIONS("/", func(context *gin.Context) {
	})

	// Use default options
	router.Use(cors.Default().Gin())
	router.Run(":8080")
}
