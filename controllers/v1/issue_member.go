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

// AddAssigneeToIssue adds a single Assignee to a project.
func AddAssigneeToIssue(c *gin.Context) {
	var req v1.IssueAssigneeRequest

	// Extract email from context
	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return
	}

	issueID := c.Param("issue_id")
	projectID := c.Param("project_id")

	// Bind and validate JSON request payload
	if !utils.BindJSONRequest(c, &req, email) {
		return
	}

	// Start database transaction
	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return
	}

	// check authorization to add assignee
	authorized, role := utils.IsUserPartOfRole(tx, projectID, email)
	if !authorized || (*role != "Manager" && *role != "Owner") {
		models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		return
	}

	// Check if the assignee is part of the project
	authorized, role = utils.IsUserPartOfRole(tx, projectID, req.Email)
	if !authorized || (*role != "Manager" && *role != "Owner") {
		models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		return
	}

	// Convert issue ID to UUID
	issueUUID, _ := utils.ConvertStringToUUID(issueID)
	projectIDUUID, _ := utils.ConvertStringToUUID(projectID)

	// Check if the assignee is already assigned to the issue
	var Assignee v1.IssueAssignee
	if err := tx.Where("issue_id = ? AND project_id = ? AND email = ?", issueID, projectID, req.Email).First(&Assignee).Error; err == nil {
		// Assignee already exists, return conflict error
		tx.Rollback()
		models.SendErrorResponse(c, http.StatusConflict, errors.ErrConflict)
		return
	}
	assignee := v1.IssueAssignee{
		Email:     req.Email,
		IssueID:   issueUUID,
		ProjectID: projectIDUUID,
	}

	// Create the issue Assignee
	if err := tx.Create(&assignee).Error; err != nil {
		tx.Rollback() // Rollback the transaction
		logger.LogError("Failed to create issue Assignee", logrus.Fields{"error": err.Error()})
		models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		return
	}

	// Commit transaction
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	// Prepare response
	response := v1.IssueAssigneeResponse{
		ID:        assignee.ID.String(),
		Email:     assignee.Email,
		ProjectID: assignee.ProjectID.String(),
		IssueID:   assignee.IssueID.String(),
		CreatedAt: assignee.CreatedAt,
		UpdatedAt: assignee.UpdatedAt,
	}

	// Send response
	models.SendSuccessResponse(c, http.StatusCreated, response, "Assignee added successfully.")

}

// GetAssignees retrieves all assignees for a given issue.
func GetAssignees(c *gin.Context) {
	issueID := c.Param("issue_id")
	projectID := c.Param("project_id")

	// Extract email from context
	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return
	}

	// Parse pagination parameters from the request
	pagination, err := utils.ParsePagination(c)
	if err != nil {
		logger.LogError("Invalid pagination parameters.", map[string]interface{}{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusBadRequest, errors.ErrBadRequest)
		return
	}

	// Start database transaction
	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return
	}

	var assignees []v1.IssueAssignee
	// Start building the query to fetch project Assignees based on the project ID
	query := tx.Model(&v1.IssueAssignee{}).Where("project_id = ? AND issue_id = ?", projectID, issueID)
	if err := query.Scopes(utils.Paginate(query, pagination)).Scan(&assignees).Error; err != nil {
		// Rollback the transaction in case of an error
		tx.Rollback()
		logger.LogError("Failed to fetch assignees from the database.", map[string]interface{}{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		return
	}

	// Commit the transaction if no errors occurred
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	var responses []v1.IssueAssigneeResponse
	for _, assignee := range assignees {
		responses = append(responses, v1.IssueAssigneeResponse{
			ID:        assignee.ID.String(),
			Email:     assignee.Email,
			ProjectID: assignee.ProjectID.String(),
			IssueID:   assignee.IssueID.String(),
			CreatedAt: assignee.CreatedAt,
			UpdatedAt: assignee.UpdatedAt,
		})
	}

	// Prepare paginated response
	response := v1.ListIssueAssigneeResponse{
		Data: responses,
	}

	// If no project Assignees are found, return an empty list
	if response.Data == nil {
		response.Data = []v1.IssueAssigneeResponse{}
	}

	// Define pagination metadata
	meta := models.PaginationMeta{
		Total: pagination.TotalCount,
		Page:  pagination.Page,
		Limit: pagination.PageSize,
	}

	// Send the paginated success response
	models.SendPaginatedSuccessResponse(c, response.Data, meta, "Project Assignees retrieved successfully.")
}

// DeleteAssigneeByID deletes a project Assignee by their ID.
func DeleteAssigneeByID(c *gin.Context) {
	// Extract email from context
	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return
	}

	// Extract project ID and Assignee ID from the request
	projectID := c.Param("project_id")
	issueID := c.Param("issue_id")
	AssigneeID := c.Param("assignee_id")

	// Start database transaction
	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return
	}

	// Check if the Assignee exists
	var Assignee v1.IssueAssignee
	if err := tx.Where("id = ? AND issue_id = ? AND project_id = ?", AssigneeID, issueID, projectID).First(&Assignee).Error; err != nil {
		logger.LogError("Failed to check if project Assignee exists", logrus.Fields{"error": err.Error()})
		models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		return
	}

	// Delete the project Assignee
	if err := tx.Delete(&Assignee).Error; err != nil {
		tx.Rollback() // Rollback the transaction
		logger.LogError("Failed to delete project Assignee", logrus.Fields{"error": err.Error()})
		models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		return
	}

	// Commit transaction
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	// Send success response
	models.SendSuccessResponse(c, http.StatusOK, nil, "Project Assignee deleted successfully.")
}
