// Package v1 defines the routes for project-related API endpoints in version 1.
// It includes routes for creating, retrieving, updating, deleting, and listing project entries.
package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/san-data-systems/common/validators"
	v1 "github.com/san-data-systems/project-management-api/controllers/v1"
)

// ProjectMember sets up the routes for task-related API endpoints.
func ProjectMember(router *gin.RouterGroup, handlers ...gin.HandlerFunc) {
	projectMember := router.Group("", handlers...)
	{
		projectMember.POST("/project/:project_id/members/operation", validators.ProjectIDValidator(), validators.ProjectMemberOperationsRequestValidator(), v1.AddORRemoveProjectMembers)
		projectMember.GET("/project/:project_id/members", validators.ProjectIDValidator(), v1.GetProjectMembers)

		projectMember.GET("/project/:project_id/member/email/:email", validators.ProjectIDValidator(), validators.EmailValidator(), v1.GetProjectMemberByEmail)
		projectMember.GET("/project/:project_id/member/id/:member_id", validators.ProjectIDValidator(), validators.ProjectMemberIDValidator(), v1.GetProjectMemberByID)

		projectMember.DELETE("/project/:project_id/member/email/:email", validators.ProjectIDValidator(), validators.EmailValidator(), v1.DeleteProjectMemberByEmail)
		projectMember.DELETE("/project/:project_id/member/id/:member_id", validators.ProjectIDValidator(), validators.ProjectMemberIDValidator(), v1.DeleteProjectMemberByID)
		projectMember.POST("/project/:project_id/member", validators.ProjectIDValidator(), validators.CreateProjectMemberValidator(), v1.AddSingleProjectMembers)

	}
}
