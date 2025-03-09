// Package v1 defines the routes for project-related API endpoints in version 1.
// It includes routes for creating, retrieving, updating, deleting, and listing project entries.
package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/san-data-systems/common/validators"
	v1 "github.com/san-data-systems/project-management-api/controllers/v1"
)

// ProjectFileRoute sets up the routes for task-related API endpoints.
func ProjectFileRoute(router *gin.RouterGroup, handlers ...gin.HandlerFunc) {
	projectFile := router.Group("", handlers...)
	{
		// Project File
		projectFile.POST("/project/:project_id/cover", validators.ProjectIDValidator(), v1.ChangeProjectCoverImageByID)
		projectFile.POST("/project/:project_id/files", validators.ProjectIDValidator(), v1.UploadProjectFiles)
		projectFile.GET("/project/:project_id/files", validators.ProjectIDValidator(), v1.GetProjectFiles)
		projectFile.DELETE("/project/:project_id/file/:file_id", validators.ProjectIDValidator(), validators.ProjectFileIDValidator(), v1.DeleteProjectFileByID)
	}
}
