// Package v1 defines the routes for label-related API endpoints in version 1.
// It includes routes for creating, retrieving, updating, deleting, and listing labels.
package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/san-data-systems/common/validators"
	v1 "github.com/san-data-systems/project-management-api/controllers/v1"
)

// ProjectLabelRoute sets up the routes for label-related API endpoints.
func ProjectLabelRoute(router *gin.RouterGroup, handlers ...gin.HandlerFunc) {
	projectLabels := router.Group("", handlers...)
	{
		// Create a new label for a project
		projectLabels.POST("/project/:project_id/label",
			validators.ProjectIDValidator(),
			v1.CreateProjectLabel,
		)

		// Get a specific label by ID for a project
		projectLabels.GET("/project/:project_id/label/:label_id",
			validators.ProjectIDValidator(),
			validators.ProjectLabelIDValidator(),
			v1.GetProjectLabelByID,
		)

		// Update a label for a project by ID
		projectLabels.PUT("/project/:project_id/label/:label_id",
			validators.ProjectIDValidator(),
			validators.ProjectLabelIDValidator(),
			v1.UpdateProjectLabelByID,
		)

		// Delete a label for a project by ID
		projectLabels.DELETE("/project/:project_id/label/:label_id",
			validators.ProjectIDValidator(),
			validators.ProjectLabelIDValidator(),
			v1.DeleteProjectLabelByID,
		)

		// List all labels for a project, with pagination
		projectLabels.GET("/project/:project_id/labels",
			validators.ProjectIDValidator(),
			v1.ListProjectLabels,
		)
	}
}
