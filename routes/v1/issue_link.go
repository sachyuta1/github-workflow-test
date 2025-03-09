package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/san-data-systems/common/validators"
	v1 "github.com/san-data-systems/project-management-api/controllers/v1"
)

// IssueLinkRoute sets up the routes for IssueLink-related API endpoints.
func IssueLinkRoute(router *gin.RouterGroup, handler ...gin.HandlerFunc) {
	issueLink := router.Group("", handler...)
	{
		issueLink.POST("/project/:project_id/issue/:issue_id/issue-link", validators.ProjectIDValidator(), validators.IssueIDValidator(), validators.CreateIssueLinkValidator(), v1.CreateIssueLink)
		issueLink.GET("/project/:project_id/issue/:issue_id/issue-links", validators.ProjectIDValidator(), validators.IssueIDValidator(), v1.ListIssueLinks)
		issueLink.GET("/project/:project_id/issue/:issue_id/issue-link/:link_id", validators.ProjectIDValidator(), validators.IssueIDValidator(), validators.IssueLinkIDValidator(), v1.GetIssueLinkByID)
		issueLink.PUT("/project/:project_id/issue/:issue_id/issue-link/:link_id", validators.ProjectIDValidator(), validators.IssueIDValidator(), validators.IssueLinkIDValidator(), validators.UpdateIssueLinkValidator(), v1.UpdateIssueLinkByID)
		issueLink.DELETE("/project/:project_id/issue/:issue_id/issue-link/:link_id", validators.ProjectIDValidator(), validators.IssueIDValidator(), validators.IssueLinkIDValidator(), v1.DeleteIssueLink) // Delete a IssueLink entry by ID
	}
}
