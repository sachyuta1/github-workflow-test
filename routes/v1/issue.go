package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/san-data-systems/common/validators"
	v1 "github.com/san-data-systems/project-management-api/controllers/v1"
)

// IssueRoute sets up the routes for Issue-related API endpoints.
func IssueRoute(router *gin.RouterGroup, handler ...gin.HandlerFunc) {
	issue := router.Group("", handler...)
	{
		issue.POST("/project/:project_id/issue", validators.ProjectIDValidator(), validators.CreateIssueValidator(), v1.CreateIssue)
		issue.GET("/project/:project_id/issues", v1.ListIssues)
		issue.GET("/project/:project_id/issue/:issue_id", validators.ProjectIDValidator(), validators.IssueIDValidator(), v1.GetIssueByID)
		issue.PATCH("/project/:project_id/issue/:issue_id", validators.IssueIDValidator(), validators.UpdateIssueValidator(), v1.UpdateIssueByID)
		issue.DELETE("/project/:project_id/issue/:issue_id", validators.ProjectIDValidator(), validators.IssueIDValidator(), v1.DeleteIssue)                     // Delete a Issue entry by ID
		issue.GET("/project/:project_id/issue/:issue_id/activities", validators.ProjectIDValidator(), validators.IssueIDValidator(), v1.ListIssueActivitiesByID) // Delete a Issue entry by ID

	}
}
