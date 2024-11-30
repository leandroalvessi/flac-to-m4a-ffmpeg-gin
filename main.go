package main

import (
	"github.com/gin-gonic/gin"
)

func main() {
	// Cria um roteador Gin
	router := gin.Default()

	// Define uma rota GET
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello, World!",
		})
	})

	// Define uma rota POST
	router.POST("/post", func(c *gin.Context) {
		var json map[string]interface{}
		if err := c.ShouldBindJSON(&json); err == nil {
			c.JSON(200, gin.H{
				"status":  "received",
				"message": json,
			})
		} else {
			c.JSON(400, gin.H{
				"status":  "error",
				"message": "Invalid JSON",
			})
		}
	})

	// Inicia o servidor na porta 8080
	router.Run(":8080")
}
