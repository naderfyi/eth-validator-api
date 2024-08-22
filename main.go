package main

import (
	"eth-validator-api/handlers"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	// Route for block reward
	router.GET("/blockreward/:slot", handlers.GetBlockReward)

	// Route for sync duties
	router.GET("/syncduties/:slot", handlers.GetSyncDuties)

	// Start the server
	router.Run(":8080")
}
