package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/san-data-systems/common/validators"
	v1 "github.com/san-data-systems/project-management-api/controllers/v1"
)

// IssueTimeEntryRoute sets up the routes for IssueTimeEntry-related API endpoints.
func IssueTimeEntryRoute(router *gin.RouterGroup, handler ...gin.HandlerFunc) {
	issueTimeEntry := router.Group("", handler...)
	{
		issueTimeEntry.POST("/project/:project_id/issue/:issue_id/time-entry", validators.ProjectIDValidator(), validators.IssueIDValidator(), validators.CreateTimeEntryValidator(), v1.CreateIssueTimeEntry)
		issueTimeEntry.GET("/project/:project_id/issue/:issue_id/time-entries", validators.ProjectIDValidator(), validators.IssueIDValidator(), v1.ListIssueTimeEntries)
		issueTimeEntry.GET("/project/:project_id/issue/:issue_id/time-entry/:te_id", validators.ProjectIDValidator(), validators.IssueIDValidator(), validators.TimeEntryIDValidator(), v1.GetIssueTimeEntryByID)
		issueTimeEntry.PUT("/project/:project_id/issue/:issue_id/time-entry/:te_id", validators.ProjectIDValidator(), validators.IssueIDValidator(), validators.TimeEntryIDValidator(), validators.CreateTimeEntryValidator(), v1.UpdateIssueTimeEntryByID)
		issueTimeEntry.DELETE("/project/:project_id/issue/:issue_id/time-entry/:te_id", validators.ProjectIDValidator(), validators.IssueIDValidator(), validators.TimeEntryIDValidator(), v1.DeleteIssueTimeEntry) // Delete a IssueTimeEntry entry by ID
	}
}
