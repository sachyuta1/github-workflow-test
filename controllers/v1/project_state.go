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

// CreateProjectState handles the creation of a new project state.
func CreateProjectState(c *gin.Context) {
	projectID := c.Param("project_id")

	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return
	}

	ProjectID, err := utils.ConvertID(projectID, c, email, "project id")
	if err != nil {
		return // Early return if conversion fails, error response is already handled
	}

	var req v1.ProjectStateRequest
	if !utils.BindJSONRequest(c, &req, email) {
		return
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

	// Count existing states in the project to determine the sequence
	var existingCount int64
	if err := tx.Debug().Model(&v1.ProjectState{}).
		Where("project_id = ?", ProjectID).
		Count(&existingCount).Error; err != nil {
		tx.Rollback()
		logger.LogError("Failed to count existing project states.", logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		return
	}

	// Create a new ProjectState instance
	projectState := v1.ProjectState{
		Name:      req.Name,
		ProjectID: ProjectID,
		Sequence:  int32(existingCount + 1), // Set sequence to 1 + existing state count
		CreatedBy: email,
	}

	// Save the new state with rollback on failure
	if !utils.CreateWithRollback(tx, c, &projectState, "Failed to create project state", email) {
		return
	}

	// Commit the transaction
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	// Prepare the response
	response := v1.ProjectStateResponse{
		ID:        projectState.ID,
		Name:      projectState.Name,
		CreatedBy: projectState.CreatedBy,
		ProjectID: projectState.ProjectID,
		Sequence:  projectState.Sequence,
		CreatedAt: projectState.CreatedAt,
		UpdatedAt: projectState.UpdatedAt,
		DeletedAt: nil,
	}

	// Send success response
	models.SendSuccessResponse(c, http.StatusCreated, response, "Project State created successfully")
}

// GetProjectStateByID handles fetching a project state by its ID for a specific project.
func GetProjectStateByID(c *gin.Context) {
	projectID := c.Param("project_id")
	stateID := c.Param("state_id")

	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return
	}

	// Convert the project and state IDs and handle any conversion errors
	parsedProjectID, err := utils.ConvertID(projectID, c, email, "project id")
	if err != nil {
		return // Early return if conversion fails, error response is already handled
	}

	parsedStateID, err := utils.ConvertID(stateID, c, email, "state id")
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

	var projectState v1.ProjectState

	// Fetch the project state by ID and associated project, ensuring the state exists for the specific project
	if err := tx.Debug().Where("id = ? AND project_id = ? AND deleted_at IS NULL", parsedStateID, parsedProjectID).First(&projectState).Error; err != nil {
		tx.Rollback()
		logger.LogError(fmt.Sprintf("Project state with ID: %s not found for project ID: %s.", stateID, projectID), logrus.Fields{"error": err.Error(), "email": email})
		// Check if the error is a record not found error
		if err == gorm.ErrRecordNotFound {
			models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		} else {
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		}
		return
	}

	// Prepare the response object
	response := v1.ProjectStateResponse{
		ID:        projectState.ID,
		Name:      projectState.Name,
		CreatedBy: projectState.CreatedBy,
		ProjectID: projectState.ProjectID,
		Sequence:  projectState.Sequence,
		CreatedAt: projectState.CreatedAt,
		UpdatedAt: projectState.UpdatedAt,
		DeletedAt: projectState.DeletedAt,
	}

	// Commit the transaction
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	// Send the success response
	models.SendSuccessResponse(c, http.StatusOK, response, "Project State retrieved successfully")
}

// UpdateProjectStateByID handles the updating of a project state by its ID for a specific project.
func UpdateProjectStateByID(c *gin.Context) {
	projectID := c.Param("project_id")
	stateID := c.Param("state_id")

	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return
	}

	// Convert the project and state IDs and handle any conversion errors
	parsedProjectID, err := utils.ConvertID(projectID, c, email, "project id")
	if err != nil {
		return // Early return if conversion fails, error response is already handled
	}

	parsedStateID, err := utils.ConvertID(stateID, c, email, "state id")
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

	var projectState v1.ProjectState
	var req v1.ProjectStateRequest

	// Bind the request data to the struct
	if !utils.BindJSONRequest(c, &req, email) {
		return
	}

	// Fetch the project state by ID and associated project
	if err := tx.Debug().Preload("Project").Where("id = ? AND project_id = ? AND deleted_at IS NULL", parsedStateID, parsedProjectID).First(&projectState).Error; err != nil {
		tx.Rollback()
		logger.LogError(fmt.Sprintf("Project state with ID: %s not found for project ID: %s.", stateID, projectID), logrus.Fields{"error": err.Error(), "email": email})
		// Check if the error is a record not found error
		if err == gorm.ErrRecordNotFound {
			models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		} else {
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		}
		return
	}

	// Update the fields with new values
	projectState.Name = req.Name
	projectState.UpdatedAt = time.Now()

	// Save the updated project state
	if err := tx.Save(&projectState).Error; err != nil {
		tx.Rollback()
		logger.LogError(fmt.Sprintf("Failed to update project state with ID: %s", stateID), logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		return
	}

	// Commit the transaction
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	// Prepare the response object
	response := v1.ProjectStateResponse{
		ID:        projectState.ID,
		Name:      projectState.Name,
		CreatedBy: projectState.CreatedBy,
		ProjectID: projectState.ProjectID,
		Sequence:  projectState.Sequence,
		CreatedAt: projectState.CreatedAt,
		UpdatedAt: projectState.UpdatedAt,
		DeletedAt: projectState.DeletedAt,
	}

	// Send the success response
	models.SendSuccessResponse(c, http.StatusOK, response, "Project State updated successfully")
}

// DeleteProjectStateByID handles the deletion of a project state by its ID for a specific project.
func DeleteProjectStateByID(c *gin.Context) {
	projectID := c.Param("project_id")
	stateID := c.Param("state_id")

	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return
	}

	// Convert the project and state IDs and handle any conversion errors
	parsedProjectID, err := utils.ConvertID(projectID, c, email, "project id")
	if err != nil {
		return
	}

	parsedStateID, err := utils.ConvertID(stateID, c, email, "state id")
	if err != nil {
		return
	}

	// Start a transaction using the helper
	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return
	}

	authorized, role := utils.IsUserPartOfRole(tx, projectID, email)
	if !authorized || (*role != "Manager" && *role != "Owner") {
		models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		return
	}

	var projectState v1.ProjectState

	// Fetch the project state by ID and associated project
	if err := tx.Debug().Where("id = ? AND project_id = ? AND deleted_at IS NULL", parsedStateID, parsedProjectID).First(&projectState).Error; err != nil {
		tx.Rollback()
		logger.LogError(fmt.Sprintf("Project state with ID: %s not found for project ID: %s.", stateID, projectID), logrus.Fields{"error": err.Error(), "email": email})
		if err == gorm.ErrRecordNotFound {
			models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		} else {
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		}
		return
	}

	// Check if the state ID is used in the Issue table
	var issueCount int64
	if err := tx.Debug().Model(&v1.Issue{}).Where("state_id = ? AND deleted_at IS NULL", parsedStateID).Count(&issueCount).Error; err != nil {
		tx.Rollback()
		logger.LogError("Failed to check issues for the state.", logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		return
	}

	if issueCount > 0 {
		tx.Rollback()
		logger.LogWarning(fmt.Sprintf("State with ID: %s cannot be deleted as it is referenced in %d issues.", stateID, issueCount), logrus.Fields{"email": email})
		models.SendErrorResponse(c, http.StatusConflict, errors.ErrConflict)
		return
	}

	// Soft delete the project state
	now := time.Now()
	projectState.DeletedAt = &now
	if err := tx.Save(&projectState).Error; err != nil {
		tx.Rollback()
		logger.LogError(fmt.Sprintf("Failed to delete project state with ID: %s", stateID), logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		return
	}

	// Reset sequence numbers for remaining states
	var remainingStates []v1.ProjectState
	if err := tx.Debug().
		Where("project_id = ? AND deleted_at IS NULL", parsedProjectID).
		Order("sequence ASC").
		Find(&remainingStates).Error; err != nil {
		tx.Rollback()
		logger.LogError("Failed to fetch remaining project states for sequence reset.", logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		return
	}

	for i, state := range remainingStates {
		state.Sequence = int32(i + 1)
		if err := tx.Save(&state).Error; err != nil {
			tx.Rollback()
			logger.LogError("Failed to reset sequence numbers.", logrus.Fields{"error": err.Error(), "email": email})
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
			return
		}
	}

	// Commit the transaction
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	// Send success response
	models.SendSuccessResponse(c, http.StatusNoContent, nil, "Project state deleted successfully and sequence reset")
}

// ListProjectStates handles the retrieval of all project states for a specific project in a paginated format.
func ListProjectStates(c *gin.Context) {
	var projectStates []v1.ProjectState

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
	query := tx.Model(&v1.ProjectState{}).Where("project_id = ? AND deleted_at IS NULL", projectID).Order("sequence ASC")

	// Execute the query to retrieve the project states with pagination
	if err := query.Scopes(utils.Paginate(query, pagination)).
		Scan(&projectStates).Error; err != nil {
		tx.Rollback()
		logger.LogError("Failed to fetch project states from the database.", logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		return
	}

	// Attempt to commit the transaction
	if !utils.CommitTransaction(tx, c, email) {
		return // Early return if the commit failed
	}

	// Prepare the response data
	var responses []v1.ProjectStateResponse
	for _, projectState := range projectStates {
		responses = append(responses, v1.ProjectStateResponse{
			ID:        projectState.ID,
			ProjectID: projectState.ProjectID,
			Name:      projectState.Name,
			Sequence:  projectState.Sequence,
			CreatedAt: projectState.CreatedAt,
			UpdatedAt: projectState.UpdatedAt,
			CreatedBy: projectState.CreatedBy,
		})
	}

	// Prepare the paginated response
	response := v1.ListProjectStateResponse{
		Data: responses,
	}

	// Ensure the response Data is not nil
	if response.Data == nil {
		response.Data = []v1.ProjectStateResponse{} // Ensure Data is not nil
	}

	// Prepare pagination meta data
	meta := models.PaginationMeta{
		Total: pagination.TotalCount, // You might want to fetch total count separately
		Page:  pagination.Page,
		Limit: pagination.PageSize,
	}

	// Send the paginated success response
	models.SendPaginatedSuccessResponse(c, response.Data, meta, "Project states retrieved successfully.")
}

// UpdateProjectStatesSequence handles the retrieval of all project states for a specific project in a paginated format.
func UpdateProjectStatesSequence(c *gin.Context) {
	projectID := c.Param("project_id")

	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return
	}

	var req v1.UpdateStatesSequenceRequest

	// Bind the request data to the struct
	if !utils.BindJSONRequest(c, &req, email) {
		return
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
	// Iterate through the stage_order and update StageOrder in the database
	for index, stageID := range req.StageSequence {

		var projectState v1.ProjectState
		if err := tx.Where("id = ? AND project_id = ?", stageID, projectID).First(&projectState).Error; err != nil {
			tx.Rollback()
			logger.LogError("Failed to update project states in the database.", logrus.Fields{"error": err.Error(), "email": email})
			models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
			return
		}

		projectState.Sequence = int32(index + 1)
		if err := tx.Save(&projectState).Error; err != nil {
			tx.Rollback()
			logger.LogError("Failed to update project states in the database.", logrus.Fields{"error": err.Error(), "email": email})
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
			return
		}

	}

	// Commit the transaction
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	// Send the success response
	models.SendSuccessResponse(c, http.StatusOK, nil, "Project state sequence updated successfully.")

}
