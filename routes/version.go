package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// VersionRoute sets up the version route
func VersionRoute(router *gin.RouterGroup) {
	version := router.Group("") // Use a dedicated path for versioning
	{
		version.GET("/version", func(c *gin.Context) {
			// Directly respond with the current version information
			c.JSON(http.StatusOK, gin.H{
				"version": "v0.0.2", // Update with your actual version
				"message": "API version information",
			})
		})
	}
}
