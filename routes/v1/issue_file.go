package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/san-data-systems/common/validators"
	v1 "github.com/san-data-systems/project-management-api/controllers/v1"
)

// IssueFileRoute sets up the routes for task-related API endpoints.
func IssueFileRoute(router *gin.RouterGroup, handlers ...gin.HandlerFunc) {
	issueFile := router.Group("", handlers...)
	{
		// Issue File
		issueFile.POST("/project/:project_id/issue/:issue_id/files", validators.ProjectIDValidator(), validators.IssueIDValidator(), v1.UploadIssueFiles)
		issueFile.GET("/project/:project_id/issue/:issue_id/files", validators.ProjectIDValidator(), validators.IssueIDValidator(), v1.GetIssueFiles)
		issueFile.DELETE("/project/:project_id/issue/:issue_id/file/:file_id", validators.ProjectIDValidator(), validators.IssueIDValidator(), v1.DeleteIssueFileByID)
	}
}
