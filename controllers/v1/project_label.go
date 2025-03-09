package v1

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/san-data-systems/common/errors"
	"github.com/san-data-systems/common/logger"
	"github.com/san-data-systems/common/models"
	v1 "github.com/san-data-systems/common/models/v1"
	"github.com/san-data-systems/common/utils"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// CreateProjectLabel handles the creation of a new label for a specific project.
func CreateProjectLabel(c *gin.Context) {
	projectID := c.Param("project_id")

	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return
	}

	// Convert project ID and handle any conversion errors
	parsedProjectID, err := utils.ConvertID(projectID, c, email, "project id")
	if err != nil {
		return // Early return if conversion fails, error response is already handled
	}

	var req v1.ProjectLabelRequest
	if !utils.BindJSONRequest(c, &req, email) {
		return
	}

	// Start a transaction using the helper
	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return // Early return if the transaction failed to start
	}

	authorized, role := utils.IsUserPartOfRole(tx, projectID, email)
	if !authorized || role == nil {
		models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		return
	}

	// Create a new Label instance
	label := v1.ProjectLabel{
		Name:      req.Name,
		ProjectID: parsedProjectID,
		Color:     req.Color,
		CreatedBy: email,
	}

	if !utils.CreateWithRollback(tx, c, &label, "Failed to create label", email) {
		return
	}
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	// Prepare response
	response := v1.ProjectLabelResponse{
		ID:        label.ID,
		Name:      label.Name,
		Color:     label.Color,
		CreatedBy: label.CreatedBy,
		ProjectID: label.ProjectID,
		CreatedAt: label.CreatedAt,
		UpdatedAt: label.UpdatedAt,
	}

	// Send success response
	models.SendSuccessResponse(c, http.StatusCreated, response, "Label created successfully")
}

// GetProjectLabelByID handles fetching a label by its ID for a specific project.
func GetProjectLabelByID(c *gin.Context) {
	projectID := c.Param("project_id")
	labelID := c.Param("label_id")

	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return
	}

	// Convert the project and label IDs and handle any conversion errors
	parsedProjectID, err := utils.ConvertID(projectID, c, email, "project id")
	if err != nil {
		return // Early return if conversion fails, error response is already handled
	}

	parsedLabelID, err := utils.ConvertID(labelID, c, email, "label id")
	if err != nil {
		return // Early return if conversion fails, error response is already handled
	}

	// Start a transaction using the helper
	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return // Early return if the transaction failed to start
	}

	authorized, role := utils.IsUserPartOfRole(tx, projectID, email)
	if !authorized || role == nil {
		models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		return
	}

	var label v1.ProjectLabel

	// Fetch the label by ID and ensure the user has access
	if err := tx.Debug().Where("id = ? AND project_id = ? AND deleted_at IS NULL", parsedLabelID, parsedProjectID).First(&label).Error; err != nil {
		tx.Rollback()
		logger.LogError(fmt.Sprintf("Label with ID: %s not found for project ID: %s.", labelID, projectID), logrus.Fields{"error": err.Error(), "email": email})
		if err == gorm.ErrRecordNotFound {
			models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		} else {
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		}
		return
	}

	// Prepare response
	response := v1.ProjectLabelResponse{
		ID:        label.ID,
		Name:      label.Name,
		Color:     label.Color,
		CreatedBy: label.CreatedBy,
		ProjectID: label.ProjectID,
		CreatedAt: label.CreatedAt,
		UpdatedAt: label.UpdatedAt,
	}

	// Commit the transaction
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	// Send success response
	models.SendSuccessResponse(c, http.StatusOK, response, "Label retrieved successfully")
}

// UpdateProjectLabelByID handles the updating of a label by its ID for a specific project.
func UpdateProjectLabelByID(c *gin.Context) {
	projectID := c.Param("project_id")
	labelID := c.Param("label_id")

	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return
	}

	// Convert the project and label IDs and handle any conversion errors
	parsedProjectID, err := utils.ConvertID(projectID, c, email, "project id")
	if err != nil {
		return // Early return if conversion fails, error response is already handled
	}

	parsedLabelID, err := utils.ConvertID(labelID, c, email, "label id")
	if err != nil {
		return // Early return if conversion fails, error response is already handled
	}

	// Start a transaction using the helper
	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return // Early return if the transaction failed to start
	}

	authorized, role := utils.IsUserPartOfRole(tx, projectID, email)
	if !authorized || (*role != "Manager" && *role != "Owner") {
		models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		return
	}

	var label v1.ProjectLabel
	var req v1.ProjectLabelRequest

	// Bind the request data to the struct
	if !utils.BindJSONRequest(c, &req, email) {
		return
	}

	// Fetch the label by ID
	if err := tx.Debug().Where("id = ? AND project_id = ? AND deleted_at IS NULL", parsedLabelID, parsedProjectID).First(&label).Error; err != nil {
		tx.Rollback()
		logger.LogError(fmt.Sprintf("Label with ID: %s not found for project ID: %s.", labelID, projectID), logrus.Fields{"error": err.Error(), "email": email})
		if err == gorm.ErrRecordNotFound {
			models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		} else {
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		}
		return
	}

	// Update the label with new values
	label.Name = req.Name
	label.UpdatedAt = time.Now()

	// Save the updated label
	if err := tx.Save(&label).Error; err != nil {
		tx.Rollback()
		logger.LogError(fmt.Sprintf("Failed to update label with ID: %s", labelID), logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		return
	}

	// Commit the transaction
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	// Prepare response
	response := v1.ProjectLabelResponse{
		ID:        label.ID,
		Name:      label.Name,
		Color:     label.Color,
		CreatedBy: label.CreatedBy,
		ProjectID: label.ProjectID,
		CreatedAt: label.CreatedAt,
		UpdatedAt: label.UpdatedAt,
	}
	// Send success response
	models.SendSuccessResponse(c, http.StatusOK, response, "Label updated successfully")
}

// DeleteProjectLabelByID handles the deletion of a label by its ID for a specific project.
func DeleteProjectLabelByID(c *gin.Context) {
	projectID := c.Param("project_id")
	labelID := c.Param("label_id")

	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return
	}

	// Convert the project and label IDs and handle any conversion errors
	parsedProjectID, err := utils.ConvertID(projectID, c, email, "project id")
	if err != nil {
		return // Early return if conversion fails, error response is already handled
	}

	parsedLabelID, err := utils.ConvertID(labelID, c, email, "label id")
	if err != nil {
		return // Early return if conversion fails, error response is already handled
	}

	// Start a transaction using the helper
	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return // Early return if the transaction failed to start
	}

	authorized, role := utils.IsUserPartOfRole(tx, projectID, email)
	if !authorized || (*role != "Manager" && *role != "Owner") {
		models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		return
	}

	var label v1.ProjectLabel

	// Fetch the label by ID
	if err := tx.Debug().Where("id = ? AND project_id = ? AND deleted_at IS NULL", parsedLabelID, parsedProjectID).First(&label).Error; err != nil {
		tx.Rollback()
		logger.LogError(fmt.Sprintf("Label with ID: %s not found for project ID: %s.", labelID, projectID), logrus.Fields{"error": err.Error(), "email": email})
		if err == gorm.ErrRecordNotFound {
			models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		} else {
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		}
		return
	}

	// Mark the label as deleted
	now := time.Now()
	label.DeletedAt = &now
	if err := tx.Save(&label).Error; err != nil {
		tx.Rollback()
		logger.LogError(fmt.Sprintf("Failed to delete label with ID: %s", labelID), logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		return
	}

	// Commit the transaction
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	// Send success response
	models.SendSuccessResponse(c, http.StatusNoContent, nil, "Label deleted successfully")
}

// ListProjectLabels handles the retrieval of all project states for a specific project in a paginated format.
func ListProjectLabels(c *gin.Context) {
	var projectLabels []v1.ProjectLabel

	// Retrieve email from the context to ensure the user is authenticated
	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return
	}

	// Parse pagination parameters from the request
	pagination, err := utils.ParsePagination(c)
	if err != nil {
		logger.LogError("Invalid pagination parameters.", logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusBadRequest, errors.ErrBadRequest)
		return
	}

	// Retrieve the project ID from the URL parameters
	projectID := c.Param("project_id")

	// Start a transaction using the helper
	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return // Early return if the transaction failed to start
	}

	authorized, role := utils.IsUserPartOfRole(tx, projectID, email)
	if !authorized || role == nil {
		models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		return
	}

	// Retrieve project states associated with the project ID
	query := tx.Model(&v1.ProjectLabel{}).
		Where("project_id = ? AND deleted_at IS NULL", projectID)

	// Execute the query to retrieve the project label with pagination

	if err := query.Scopes(utils.Paginate(query, pagination)).Scan(&projectLabels).Error; err != nil {
		tx.Rollback()
		logger.LogError("Failed to fetch clients from the database.", map[string]interface{}{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		return
	}

	// Attempt to commit the transaction
	if !utils.CommitTransaction(tx, c, email) {
		return // Early return if the commit failed
	}

	// Prepare the response data
	var responses []v1.ProjectLabelResponse
	for _, projectState := range projectLabels {
		responses = append(responses, v1.ProjectLabelResponse{
			ID:        projectState.ID,
			ProjectID: projectState.ProjectID,
			Name:      projectState.Name,
			Color:     projectState.Color,
			CreatedAt: projectState.CreatedAt,
			UpdatedAt: projectState.UpdatedAt,
			CreatedBy: projectState.CreatedBy,
		})
	}

	// Prepare the paginated response
	response := v1.ListProjectLabelResponse{
		Data: responses,
	}

	// Ensure the response Data is not nil
	if response.Data == nil {
		response.Data = []v1.ProjectLabelResponse{} // Ensure Data is not nil
	}

	// Prepare pagination meta data
	meta := models.PaginationMeta{
		Total: pagination.TotalCount, // You might want to fetch total count separately
		Page:  pagination.Page,
		Limit: pagination.PageSize,
	}

	// Send the paginated success response
	models.SendPaginatedSuccessResponse(c, response.Data, meta, "Project labels retrieved successfully.")
}
