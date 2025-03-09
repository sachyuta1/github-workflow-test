// Package v1 defines the routes for project-related API endpoints in version 1.
// It includes routes for creating, retrieving, updating, deleting, and listing project entries.
package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/san-data-systems/common/validators"
	v1 "github.com/san-data-systems/project-management-api/controllers/v1"
)

// ProjectStateRoute sets up the routes for task-related API endpoints.
func ProjectStateRoute(router *gin.RouterGroup, handlers ...gin.HandlerFunc) {
	projectState := router.Group("", handlers...)
	{
		projectState.POST("/project/:project_id/state",
			validators.ProjectIDValidator(),
			validators.CreateProjectStateValidator(),
			v1.CreateProjectState,
		)
		projectState.GET("/project/:project_id/state/:state_id",
			validators.ProjectIDValidator(),
			validators.ProjectStateIDValidator(),
			v1.GetProjectStateByID,
		)
		projectState.PUT("/project/:project_id/state/:state_id",
			validators.ProjectIDValidator(),
			validators.ProjectStateIDValidator(),
			validators.UpdateProjectStateValidator(),
			v1.UpdateProjectStateByID,
		)
		projectState.DELETE("/project/:project_id/state/:state_id",
			validators.ProjectIDValidator(),
			validators.ProjectStateIDValidator(),
			v1.DeleteProjectStateByID,
		)
		projectState.GET("/project/:project_id/states", validators.ProjectIDValidator(), v1.ListProjectStates)
		projectState.PUT("/project/:project_id/states", validators.ProjectIDValidator(), validators.ProjectStatesSequenceUpdateValidator(), v1.UpdateProjectStatesSequence)
	}
}
