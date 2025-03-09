package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/san-data-systems/common/errors"
	"github.com/san-data-systems/common/logger"
	"github.com/san-data-systems/common/models"
	v1 "github.com/san-data-systems/common/models/v1"
	"github.com/san-data-systems/common/utils"
	"github.com/sirupsen/logrus"
)

// ListIssueActivitiesByID checks if a project with the provided slug exists.
func ListIssueActivitiesByID(c *gin.Context) {
	var issueActivities []v1.IssueActivity
	projectID := c.Param("project_id")
	issueID := c.Param("issue_id")

	// Retrieve email from context
	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return // The response is already sent by the helper, so just return
	}

	// Parse pagination parameters (page, page_size) using utility function
	pagination, err := utils.ParsePagination(c)
	if err != nil {
		logger.LogError("Invalid pagination parameters.", logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusBadRequest, errors.ErrBadRequest)
		return
	}

	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return
	}

	authorized, _ := utils.IsUserPartOfRole(tx, projectID, email)
	if !authorized {
		models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		return
	}

	// Start building the query
	query := tx.Model(&v1.IssueActivity{}).Where("project_id = ? AND issue_id = ?", projectID, issueID)

	// Execute the query with pagination
	if err := query.Scopes(utils.Paginate(query, pagination)).Scan(&issueActivities).Error; err != nil {
		tx.Rollback()
		logger.LogError("Failed to list projects.", logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		return
	}

	// Commit transaction
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	// Convert clients to responses
	var responses []v1.IssueActivityResponse
	for _, activity := range issueActivities {
		responses = append(responses, v1.IssueActivityResponse{
			ID:        activity.ID.String(),
			IssueID:   activity.IssueID.String(),
			ProjectID: activity.ProjectID.String(),
			Email:     activity.Email,
			Action:    activity.Action,
			Entity:    activity.Entity,
			Column:    activity.Column,
			OldValue:  activity.OldValue,
			NewValue:  activity.NewValue,
			CreatedAt: activity.CreatedAt,
		},
		)
	}

	// Prepare paginated response
	response := v1.ListIssueActivityResponse{
		Data: responses,
	}

	// If no clients found, return an empty list
	if response.Data == nil {
		response.Data = []v1.IssueActivityResponse{}
	}

	meta := models.PaginationMeta{
		Total: pagination.TotalCount,
		Page:  pagination.Page,
		Limit: pagination.PageSize,
	}

	// Send successful response
	models.SendPaginatedSuccessResponse(c, response.Data, meta, "Issue activities retrieved successfully.")

}
