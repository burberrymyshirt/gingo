package main

import (
	"github.com/burberrymyshirt/gingo"
	"github.com/gin-gonic/gin"
)

func main() {
	router := gingo.Default()

	router.GET("ping", func(c *gingo.Context) {
		var request struct {
			Name string `json:"name" binding:"required"`
		}

		if err := c.ShouldBindJSON(request); err != nil {
			c.JSON(422, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"message": "pong"})
	})

	router.Run()
}
