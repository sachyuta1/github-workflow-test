package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/san-data-systems/common/validators"
	v1 "github.com/san-data-systems/project-management-api/controllers/v1"
)

// IssueAssigneeRoute sets up the routes for IssueAssignee-related API endpoints.
func IssueAssigneeRoute(router *gin.RouterGroup, handler ...gin.HandlerFunc) {
	issueAssignee := router.Group("", handler...)
	{
		issueAssignee.POST("/project/:project_id/issue/:issue_id/assignee", validators.ProjectIDValidator(), validators.IssueIDValidator(), validators.CreateIssueAssigneeValidator(), v1.AddAssigneeToIssue)
		issueAssignee.GET("/project/:project_id/issue/:issue_id/assignees", validators.ProjectIDValidator(), validators.IssueIDValidator(), v1.GetAssignees)
		issueAssignee.DELETE("/project/:project_id/issue/:issue_id/assignee/:assignee_id", validators.ProjectIDValidator(), validators.IssueIDValidator(), validators.IssueAssigneeIDValidator(), v1.DeleteAssigneeByID)
	}
}
