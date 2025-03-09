package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/san-data-systems/common/errors"
	"github.com/san-data-systems/common/logger"
	"github.com/san-data-systems/common/models"
	v1 "github.com/san-data-systems/common/models/v1"
	"github.com/san-data-systems/common/utils"
	"github.com/sirupsen/logrus"
	"net/http"
)

// ListProjectActivitiesByID checks if a project with the provided slug exists.
func ListProjectActivitiesByID(c *gin.Context) {
	var projectActivities []v1.ProjectActivity
	id := c.Param("project_id")

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

	authorized, role := utils.IsUserPartOfRole(tx, id, email)
	if !authorized || (*role != "Manager" && *role != "Owner") {
		models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		return
	}

	// Start building the query
	query := tx.Model(&v1.ProjectActivity{}).Where("project_id = ?", id)

	// Execute the query with pagination
	if err := query.Scopes(utils.Paginate(query, pagination)).Scan(&projectActivities).Error; err != nil {
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
	var responses []v1.ProjectActivityResponse
	for _, activity := range projectActivities {
		responses = append(responses, v1.ProjectActivityResponse{
			ID:        activity.ID.String(),
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
	response := v1.ListProjectActivityResponse{
		Data: responses,
	}

	// If no clients found, return an empty list
	if response.Data == nil {
		response.Data = []v1.ProjectActivityResponse{}
	}

	meta := models.PaginationMeta{
		Total: pagination.TotalCount,
		Page:  pagination.Page,
		Limit: pagination.PageSize,
	}

	// Send successful response
	models.SendPaginatedSuccessResponse(c, response.Data, meta, "Project activities retrieved successfully.")

}
