package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthzRoute sets up health check routes
func HealthzRoute(router *gin.RouterGroup) {
	healthz := router.Group("") // Use a dedicated path for health checks
	{
		healthz.GET("/healthz", func(c *gin.Context) {
			// Directly respond with a health check message
			c.JSON(http.StatusOK, gin.H{
				"status":  "ok",
				"message": "Service is running",
			})
		})
	}
}
