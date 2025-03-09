// Package v1 defines the routes for project-related API endpoints in version 1.
// It includes routes for creating, retrieving, updating, deleting, and listing project entries.
package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/san-data-systems/common/validators"
	v1 "github.com/san-data-systems/project-management-api/controllers/v1"
)

// ProjectRoute sets up the routes for task-related API endpoints.
func ProjectRoute(router *gin.RouterGroup, handlers ...gin.HandlerFunc) {
	project := router.Group("", handlers...)
	{
		project.POST("/project", validators.CreateProjectValidator(), v1.CreateProject)
		project.GET("/project/:project_id", validators.ProjectIDValidator(), v1.GetProjectByID)
		project.PUT("/project/:project_id", validators.ProjectIDValidator(), v1.UpdateProjectByID)
		project.DELETE("/project/:project_id", validators.ProjectIDValidator(), v1.DeleteProjectByID)
		project.GET("/projects", v1.ListProjects)
		project.GET("/project/:project_id/stats", validators.ProjectIDValidator(), v1.GetProjectStatsByID)
		project.GET("/project/:project_id/activities", validators.ProjectIDValidator(), v1.ListProjectActivitiesByID)
	}
}
