// Package v1 defines the routes for project-related API endpoints in version 1.
// It includes routes for creating, retrieving, updating, deleting, and listing project entries.
package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/san-data-systems/common/validators"
	v1 "github.com/san-data-systems/project-management-api/controllers/v1"
)

// ProjectSlugRoute sets up the routes for task-related API endpoints.
func ProjectSlugRoute(router *gin.RouterGroup, handlers ...gin.HandlerFunc) {
	projectSlug := router.Group("", handlers...)
	{
		// Project slug
		projectSlug.GET("/projects/slug/:slug/check", validators.ProjectSlugValidator(), v1.CheckProjectSlugExists)
		projectSlug.GET("/projects/slug/:slug", validators.ProjectSlugValidator(), v1.GetProjectBySlug)
	}
}
