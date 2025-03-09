package v1

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/san-data-systems/common/errors"
	"github.com/san-data-systems/common/logger"
	"github.com/san-data-systems/common/models"
	v1 "github.com/san-data-systems/common/models/v1"
	"github.com/san-data-systems/common/utils"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"net/http"
)

// AddSingleProjectMembers adds a single member to a project.
func AddSingleProjectMembers(c *gin.Context) {
	var req v1.ProjectMemberRequest

	// Extract email from context
	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return
	}

	// Retrieve and validate project ID
	projectID := c.Param("project_id")
	ProjectID, err := utils.ConvertID(projectID, c, email, "project id")
	if err != nil {
		return // Early return if conversion fails, error response is already handled
	}

	// Bind and validate JSON request payload
	if !utils.BindJSONRequest(c, &req, email) {
		return
	}

	// Start database transaction
	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return
	}

	// Check user authorization to add members
	if !utils.IsUserAuthorizedToAddMember(tx, ProjectID, email) {
		models.SendSuccessResponse(c, http.StatusForbidden, nil, "User is not authorized to add members.")
		return
	}

	// Create a new ProjectMember instance
	projectMember := v1.ProjectMember{
		Email:     req.Email,
		Role:      req.Role,
		ProjectID: ProjectID,
	}

	var existingMember v1.ProjectMember

	// Check if a member with the same email already exists in the project
	if err := tx.Model(&v1.ProjectMember{}).Where("email = ? AND project_id = ?", projectMember.Email, ProjectID).First(&existingMember).Error; err != nil {
		if err != gorm.ErrRecordNotFound { // Proceed only if the user is not found
			tx.Rollback()
			logger.LogError("Failed to check existing member.", logrus.Fields{"error": err.Error(), "email": email})
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
			return
		}
	}

	// If the member already exists with the same role, return a conflict response
	if existingMember.ID != uuid.Nil && existingMember.Role == projectMember.Role {
		tx.Rollback()
		logger.LogError("User already exists with the same role.", logrus.Fields{"email": projectMember.Email, "projectID": projectID})
		models.SendErrorResponse(c, http.StatusConflict, "User with the same role already exists in the project.")
		return
	}

	// If the member exists with a different role, update the role
	if existingMember.ID != uuid.Nil && existingMember.Role != projectMember.Role {
		existingMember.Role = projectMember.Role
		if err := tx.Save(&existingMember).Error; err != nil {
			tx.Rollback()
			logger.LogError("Failed to update project member role.", logrus.Fields{"error": err.Error(), "email": projectMember.Email})
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
			return
		}
	}

	// If no existing member, create a new project member
	if existingMember.ID == uuid.Nil {
		// Save the project to the database
		if !utils.CreateWithRollback(tx, c, &projectMember, "Failed to create project memebr.", email) {
			return
		}
	}

	// Commit transaction
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	// Prepare response
	response := v1.ProjectMemberResponse{
		ID:        projectMember.ID.String(),
		ProjectID: projectMember.ProjectID.String(),
		Email:     projectMember.Email,
		Role:      projectMember.Role,
		CreatedAt: projectMember.CreatedAt,
		UpdatedAt: projectMember.UpdatedAt,
	}

	// Send success response
	models.SendSuccessResponse(c, http.StatusCreated, response, "Project Member added successfully.")
}

// DeleteProjectMemberByID deletes a project member by their ID.
func DeleteProjectMemberByID(c *gin.Context) {
	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return
	}

	// Retrieve and validate project ID
	projectID := c.Param("project_id")
	ProjectID, err := utils.ConvertID(projectID, c, email, "project id")
	if err != nil {
		return // Early return if conversion fails, error response is already handled
	}

	// Retrieve member ID from parameters
	memberID := c.Param("member_id")

	// Start database transaction
	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return
	}

	// Check user authorization to delete members
	if !utils.IsUserAuthorizedToAddMember(tx, ProjectID, email) {
		// Send forbidden response
		models.SendSuccessResponse(c, http.StatusForbidden, nil, "User is not authorized to delete member.")
		return
	}

	// Attempt to delete project member
	if err := tx.Model(&v1.ProjectMember{}).Where("id = ? AND project_id = ?", memberID, projectID).Delete(&v1.ProjectMember{}).Error; err != nil {
		tx.Rollback()
		logger.LogError(fmt.Sprintf("Failed to delete project member with ID: %s.", memberID), logrus.Fields{
			"error": err.Error(),
			"email": email,
		})
		if err == gorm.ErrRecordNotFound {
			models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		} else {
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		}
		return
	}

	// Commit transaction
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	// Send success response
	models.SendSuccessResponse(c, http.StatusNoContent, nil, "Project Member deleted successfully.")
}

// DeleteProjectMemberByEmail deletes a project member by their email address.
func DeleteProjectMemberByEmail(c *gin.Context) {
	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return
	}

	// Retrieve and validate project ID
	projectID := c.Param("project_id")
	ProjectID, err := utils.ConvertID(projectID, c, email, "project id")
	if err != nil {
		return // Early return if conversion fails, error response is already handled
	}

	// Retrieve member email from parameters
	memberEmail := c.Param("email")

	// Start database transaction
	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return
	}

	// Check user authorization to delete members
	if !utils.IsUserAuthorizedToAddMember(tx, ProjectID, email) {
		// Send forbidden response
		models.SendSuccessResponse(c, http.StatusForbidden, nil, "User is not authorized to delete member.")
		return
	}

	// Attempt to delete project member
	if err := tx.Model(&v1.ProjectMember{}).Where("email = ? AND project_id = ?", memberEmail, projectID).Delete(&v1.ProjectMember{}).Error; err != nil {
		tx.Rollback()
		logger.LogError(fmt.Sprintf("Failed to delete project member with email: %s.", memberEmail), logrus.Fields{
			"error": err.Error(),
			"email": email,
		})
		if err == gorm.ErrRecordNotFound {
			models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		} else {
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		}
		return
	}

	// Commit transaction
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	// Send success response
	models.SendSuccessResponse(c, http.StatusNoContent, nil, "Project Member deleted successfully.")
}

// GetProjectMemberByID retrieves a project member by their ID and project ID.
// It starts a database transaction, queries for the project member based on the provided memberID and projectID,
// and returns the project member's details in the response. If an error occurs, it handles the error appropriately,
// rolling back the transaction if necessary and sending the corresponding error response.
func GetProjectMemberByID(c *gin.Context) {
	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return
	}

	// Extract projectID and memberID from URL parameters
	projectID := c.Param("project_id")
	memberID := c.Param("member_id")

	// Start a new transaction
	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return
	}

	var projectMember v1.ProjectMember

	// Query the project member from the database
	if err := tx.Model(&v1.ProjectMember{}).Where("id = ? AND project_id = ?", memberID, projectID).First(&projectMember).Error; err != nil {
		tx.Rollback()
		logger.LogError(fmt.Sprintf("Failed to get project member with ID: %s in project: %s.", memberID, projectID), logrus.Fields{
			"error": err.Error(),
			"email": email,
		})
		// Handle the error based on the type of error
		if err == gorm.ErrRecordNotFound {
			models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		} else {
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		}
		return
	}

	// Commit the transaction if no errors occurred
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	// Prepare response with project member details
	response := v1.ProjectMemberResponse{
		ID:        projectMember.ID.String(),
		ProjectID: projectMember.ProjectID.String(),
		Email:     projectMember.Email,
		Role:      projectMember.Role,
		CreatedAt: projectMember.CreatedAt,
		UpdatedAt: projectMember.UpdatedAt,
	}

	// Send success response with project member details
	models.SendSuccessResponse(c, http.StatusOK, response, "Project Member retrieved successfully.")
}

// GetProjectMemberByEmail retrieves a project member by their ID and project ID using the provided email context.
// It starts a transaction, queries the database to find the project member, and returns the member details in the response.
// If the member is not found, it sends a 404 error response. In case of any internal error, a 500 error response is sent.
func GetProjectMemberByEmail(c *gin.Context) {
	// Extract the email from the context
	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return
	}

	// Get the project and member IDs from the URL parameters
	projectID := c.Param("project_id")
	memberEmail := c.Param("email")

	// Start a transaction for the current operation
	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return
	}

	// Query the project member from the database
	var projectMember v1.ProjectMember
	if err := tx.Model(&v1.ProjectMember{}).Where("email = ? AND project_id = ?", memberEmail, projectID).First(&projectMember).Error; err != nil {
		// Rollback the transaction in case of an error
		tx.Rollback()
		logger.LogError(fmt.Sprintf("Failed to get project member with ID: %s in project: %s.", memberEmail, projectID), logrus.Fields{
			"error": err.Error(),
			"email": email,
		})

		// Send appropriate error response based on the error type
		if err == gorm.ErrRecordNotFound {
			models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		} else {
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		}
		return
	}

	// Commit the transaction if no errors occurred
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	// Prepare the response with the retrieved project member data
	response := v1.ProjectMemberResponse{
		ID:        projectMember.ID.String(),
		ProjectID: projectMember.ProjectID.String(),
		Email:     projectMember.Email,
		Role:      projectMember.Role,
		CreatedAt: projectMember.CreatedAt,
		UpdatedAt: projectMember.UpdatedAt,
	}

	// Send a success response with the project member details
	models.SendSuccessResponse(c, http.StatusOK, response, "Project member retrieved successfully.")
}

// GetProjectMembers retrieves a list of project members for a given project.
// It supports pagination and returns project member details in a paginated response.
// If any error occurs, appropriate error responses are sent, including 400 for invalid pagination and 500 for database failures.
func GetProjectMembers(c *gin.Context) {
	// Extract the email from the context
	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return
	}

	// Get the project ID from the URL parameters
	projectID := c.Param("project_id")

	// Parse pagination parameters from the request
	pagination, err := utils.ParsePagination(c)
	if err != nil {
		logger.LogError("Invalid pagination parameters.", map[string]interface{}{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusBadRequest, errors.ErrBadRequest)
		return
	}

	// Start a transaction for the current operation
	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return
	}

	var projectMembers []v1.ProjectMember

	// Start building the query to fetch project members based on the project ID
	query := tx.Model(&v1.ProjectMember{}).Where("project_id = ?", projectID)
	if err := query.Scopes(utils.Paginate(query, pagination)).Scan(&projectMembers).Error; err != nil {
		// Rollback the transaction in case of an error
		tx.Rollback()
		logger.LogError("Failed to fetch project members from the database.", map[string]interface{}{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		return
	}

	// Commit the transaction if no errors occurred
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	// Convert the project members to response objects
	var responses []v1.ProjectMemberResponse
	for _, projectMember := range projectMembers {
		responses = append(responses, v1.ProjectMemberResponse{
			ID:        projectMember.ID.String(),
			ProjectID: projectMember.ProjectID.String(),
			Email:     projectMember.Email,
			Role:      projectMember.Role,
			CreatedAt: projectMember.CreatedAt,
			UpdatedAt: projectMember.UpdatedAt,
		})
	}

	// Prepare paginated response
	response := v1.ListProjectMemberResponse{
		Data: responses,
	}

	// If no project members are found, return an empty list
	if response.Data == nil {
		response.Data = []v1.ProjectMemberResponse{}
	}

	// Define pagination metadata
	meta := models.PaginationMeta{
		Total: pagination.TotalCount,
		Page:  pagination.Page,
		Limit: pagination.PageSize,
	}

	// Send the paginated success response
	models.SendPaginatedSuccessResponse(c, response.Data, meta, "Project members retrieved successfully.")
}

// AddORRemoveProjectMembers handles batch operations (add/remove) for project members.
// It processes the provided operations, updates the project members accordingly, and returns a success response.
func AddORRemoveProjectMembers(c *gin.Context) {
	// Extract the email from the context
	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return
	}
	// Get the project ID from the URL parameters
	projectID := c.Param("project_id")
	ProjectID, err := utils.ConvertID(projectID, c, email, "project id")
	if err != nil {
		return // Early return if conversion fails, error response is already handled
	}

	// Parse the request body to get the list of operations
	var req v1.ProjectMemberOperationsRequest
	// Bind and validate JSON request payload
	if !utils.BindJSONRequest(c, &req, email) {
		return
	}

	fmt.Println(req)

	// Start a transaction for the current operation
	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return
	}

	// Iterate over the operations and apply them
	for _, operation := range req.Operations {
		// Validate operation type
		if operation.Operation != "add" && operation.Operation != "remove" {
			logger.LogError("Invalid operation type.", logrus.Fields{"operation": operation.Operation, "email": email})
			models.SendErrorResponse(c, http.StatusBadRequest, errors.ErrBadRequest)
			return
		}

		// Process each email in the operation
		for _, email := range operation.Emails {
			// Handle the 'add' operation
			if operation.Operation == "add" {
				// Check if the user already exists in the project
				var existingMember v1.ProjectMember
				if err := tx.Model(&v1.ProjectMember{}).Where("email = ? AND project_id = ?", email, projectID).First(&existingMember).Error; err != nil {
					if err != gorm.ErrRecordNotFound { // Proceed only if the user is not found, i.e., no existing record
						tx.Rollback()
						logger.LogError("Failed to check existing member.", logrus.Fields{"error": err.Error(), "email": email})
						models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
						return
					}
					// Create a new member if not found
					newMember := v1.ProjectMember{
						Email:     email,
						Role:      operation.Role,
						ProjectID: ProjectID,
					}
					if err := tx.Create(&newMember).Error; err != nil {
						tx.Rollback()
						logger.LogError("Failed to add new project member.", logrus.Fields{"error": err.Error(), "email": email})
						models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
						return
					}
				} else {
					// If the member exists, update the role
					existingMember.Role = operation.Role
					if err := tx.Save(&existingMember).Error; err != nil {
						tx.Rollback()
						logger.LogError("Failed to update project member role.", logrus.Fields{"error": err.Error(), "email": email})
						models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
						return
					}
				}
			}

			// Handle the 'remove' operation
			if operation.Operation == "remove" {
				// Find the project member to remove
				var projectMember v1.ProjectMember
				if err := tx.Model(&v1.ProjectMember{}).Where("email = ? AND project_id = ?", email, projectID).First(&projectMember).Error; err != nil {
					if err == gorm.ErrRecordNotFound {
						// Don't return error if member is not found, just skip removing
						continue
					}
					tx.Rollback()
					logger.LogError("Failed to find project member for removal.", logrus.Fields{"error": err.Error(), "email": email})
					models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
					return
				}

				// Remove the member from the project
				if err := tx.Delete(&projectMember).Error; err != nil {
					tx.Rollback()
					logger.LogError("Failed to remove project member.", logrus.Fields{"error": err.Error(), "email": email})
					models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
					return
				}
			}
		}
	}

	// Commit the transaction if no errors occurred
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	// Send success response
	models.SendSuccessResponse(c, http.StatusOK, nil, "Project members updated successfully.")
}
