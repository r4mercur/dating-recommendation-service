package routes

import (
	"github.com/gin-gonic/gin"
)

func RegisterHealthcheckRoutes(r *gin.Engine) {
	r.GET("/healthcheck", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"response": "running",
		})
	})
}
