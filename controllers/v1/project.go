// Package v1 contains API handlers for managing projects and their associated operations.
// This package is part of version 1 of the API, implemented using the Gin framework.
package v1

import (
	"fmt"
	"net/http"
	"time"

	"github.com/lib/pq"

	"github.com/gin-gonic/gin"
	"github.com/san-data-systems/common/errors"
	"github.com/san-data-systems/common/logger"
	"github.com/san-data-systems/common/models"
	v1 "github.com/san-data-systems/common/models/v1"
	"github.com/san-data-systems/common/utils"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// CreateProject handles project creation logic, including file uploads
func CreateProject(c *gin.Context) {
	var req v1.ProjectRequest
	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return
	}

	if !utils.BindJSONRequest(c, &req, email) {
		return
	}

	layout := "2006-01-02"
	startDate, err := time.Parse(layout, req.StartDate)
	if err != nil {
		models.SendErrorResponse(c, http.StatusUnprocessableEntity, "Start date is not in correct format.")
		return
	}

	endDate, err := time.Parse(layout, req.EndDate)
	if err != nil {
		models.SendErrorResponse(c, http.StatusUnprocessableEntity, "End date is not in correct format.")
		return
	}

	// Start transaction
	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return
	}

	ClientID, err := utils.ConvertID(req.ClientID, c, email, "client id")
	if err != nil {
		return
	}

	// Create the project record
	project := v1.Project{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		ClientID:    ClientID,
		StartDate:   startDate,
		EndDate:     endDate,
		Status:      req.Status,
		Tags:        req.Tags,
		CreatedBy:   email,
	}

	var client v1.Client
	if err := tx.First(&client, "id = ?", req.ClientID).Error; err != nil {
		logger.LogError(fmt.Sprintf("Client not found with ID: %s.", req.ClientID), logrus.Fields{"error": err.Error(), "email": email})
		tx.Rollback()
		models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		return
	}

	// Save the project to the database
	if !utils.CreateWithRollback(tx, c, &project, "Failed to create project.", email) {
		return
	}

	// Preload project data
	var projectr v1.Project
	if err := tx.Preload("Client").Where("id = ?", project.ID).First(&projectr).Error; err != nil {
		tx.Rollback()
		logger.LogError(fmt.Sprintf("Project with ID: %s not found.", project.ID), logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		return
	}
	var imageURL string
	if req.CoverURL != "" {
		// Call DownloadFile function
		_, filePath, err := utils.DownloadFileFromURL(req.CoverURL, "cover")
		if err != nil {
			logger.LogError(fmt.Sprintf("Cover image for project %s failed.", projectr.ID), logrus.Fields{"error": err.Error()})
		} else {
			logger.LogInfo(fmt.Sprintf("Cover image for project %s downloaded successfully.", projectr.ID), nil)
		}

		// Handle file upload logic
		_, projectFileID, err := utils.HandleCoverFileUpload(c, filePath, tx, email, projectr.ID)
		if err != nil {
			logger.LogError(fmt.Sprintf("Error during file upload: %v", err), logrus.Fields{"error": err.Error(), "email": email})
		} else {
			err = utils.DeleteFile(filePath)
			if err != nil {
				logger.LogFatal("Failed to delete the file", logrus.Fields{"error": err.Error()})
			} else {
				fmt.Println("Downloaded file deleted successfully.")
			}
		}

		// Update only the "Name" field
		if err := tx.Model(&projectr).Update("cover_page_id", projectFileID).Error; err != nil {
			tx.Rollback()
			logger.LogError(fmt.Sprintf("Error updating project with ID: %s", project.ID), logrus.Fields{"error": err.Error(), "email": email})
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
			return
		}
	}

	// Commit transaction
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	// Prepare response
	response := v1.ProjectResponse{
		ID:          projectr.ID,
		Name:        projectr.Name,
		Slug:        projectr.Slug,
		Description: projectr.Description,
		Client: v1.Client{
			ID:            projectr.Client.ID,
			Name:          projectr.Client.Name,
			ManagerEmails: projectr.Client.ManagerEmails,
			Country:       projectr.Client.Country,
			CreatedAt:     projectr.Client.CreatedAt,
			UpdatedAt:     projectr.Client.UpdatedAt,
		},
		StartDate: projectr.StartDate,
		EndDate:   projectr.EndDate,
		Status:    projectr.Status,
		Tags:      projectr.Tags,
		CreatedAt: projectr.CreatedAt,
		UpdatedAt: projectr.UpdatedAt,
		CreatedBy: projectr.CreatedBy,
		CoverURL:  imageURL,
	}

	models.SendSuccessResponse(c, http.StatusCreated, response, "Project created successfully.")
}

// GetProjectByID godoc
func GetProjectByID(c *gin.Context) {
	id := c.Param("project_id")

	// Retrieve email from context
	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		// Return early if email is not found in context
		return
	}

	// Start a transaction
	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		// Rollback and return if the transaction could not be started
		return
	}

	authorized, role := utils.IsUserPartOfRole(tx, id, email)
	if !authorized && role == nil {
		models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		return
	}

	var project v1.Project
	// Query project by ID ensuring it's not archived and user has necessary permissions
	err := tx.Debug().Preload("Client").Where("id = ? AND deleted_at IS NULL", id).First(&project).Error

	if err != nil {
		// Rollback on error and log the issue
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			logger.LogError(fmt.Sprintf("Project with ID: %s not found.", id), logrus.Fields{"email": email})
			models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		} else {
			logger.LogError(fmt.Sprintf("Error fetching project with ID: %s: %s", id, err.Error()), logrus.Fields{"email": email})
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		}
		return
	}

	// Fetch MinIO client and pre-signed URL for project image
	var imageURL string
	if project.CoverPageID.String() == "00000000-0000-0000-0000-000000000000" {
		imageURL = ""
	} else {
		url, err := utils.FetchPreSignedURLByProjectFileID(c, tx, project.CoverPageID.String(), email)
		if err != nil {
			logger.LogError(fmt.Sprintf("Failed to fetch pre-signed URL for project file: %s", err.Error()), logrus.Fields{"email": email})
		} else {
			imageURL = url
		}
	}

	// Commit transaction
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	// Prepare response
	response := v1.ProjectResponse{
		ID:          project.ID,
		Name:        project.Name,
		Slug:        project.Slug,
		Description: project.Description,
		Client: v1.Client{
			ID:            project.Client.ID,
			Name:          project.Client.Name,
			ManagerEmails: project.Client.ManagerEmails,
			Country:       project.Client.Country,
			CreatedAt:     project.Client.CreatedAt,
			UpdatedAt:     project.Client.UpdatedAt,
		},
		StartDate: project.StartDate,
		EndDate:   project.EndDate,
		Status:    project.Status,
		Tags:      project.Tags,
		CreatedAt: project.CreatedAt,
		UpdatedAt: project.UpdatedAt,
		CreatedBy: project.CreatedBy,
		CoverURL:  imageURL,
	}

	// Send success response
	models.SendSuccessResponse(c, http.StatusOK, response, "Project retrieved successfully.")
}

// UpdateProjectByID godoc
func UpdateProjectByID(c *gin.Context) {
	id := c.Param("project_id")

	// Retrieve email from context
	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return // The response is already sent by the helper, so just return
	}

	// Start a transaction using the helper
	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return // Early return if the transaction failed to start
	}

	var project v1.Project
	if err := tx.Where("id = ? AND deleted_at IS NULL AND created_by = ?", id, email).First(&project).Error; err != nil {
		tx.Rollback()
		logger.LogError(fmt.Sprintf("Project with ID: %s not found.", id), logrus.Fields{"error": err.Error(), "email": email})
		if err == gorm.ErrRecordNotFound {
			models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		} else {
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		}
		return
	}

	var req v1.ProjectRequest
	if !utils.BindJSONRequest(c, &req, email) {
		tx.Rollback()
		return
	}
	if req.ClientID != "" {
		userID, err := utils.ConvertID(req.ClientID, c, email, "client id")
		if err != nil {
			return
		}
		project.ClientID = userID
	}

	layout := "2006-01-02"
	if req.StartDate != "" {
		startDate, err := time.Parse(layout, req.StartDate)
		if err != nil {
			models.SendErrorResponse(c, http.StatusUnprocessableEntity, "Start date is not in correct format..")
			return
		}
		project.StartDate = startDate
	}

	if req.EndDate != "" {
		endDate, err := time.Parse(layout, req.EndDate)
		if err != nil {
			models.SendErrorResponse(c, http.StatusUnprocessableEntity, "End date is not in correct format..")
			return
		}
		project.EndDate = endDate
	}

	if req.Name != "" {
		project.Name = req.Name
	}
	if req.Slug != "" {
		project.Slug = req.Slug
	}
	if req.Description != "" {
		project.Description = req.Description
	}

	if req.Status != "" {
		project.Status = req.Status
	}
	if len(req.Tags) > 0 {
		project.Tags = req.Tags
	}
	if err := tx.Save(&project).Error; err != nil {
		tx.Rollback()
		logger.LogError("Failed to update project.", logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		return
	}
	// Preload the associated Client after updating the ClientID
	var updatedProject v1.Project
	if err := tx.Preload("Client").Where("id = ?", project.ID).First(&updatedProject).Error; err != nil {
		tx.Rollback()
		logger.LogError("Failed to preload the client.", logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		return
	}

	// Fetch MinIO client and pre-signed URL for project image
	var imageURL string
	url, err := utils.FetchPreSignedURLByProjectFileID(c, tx, project.CoverPageID.String(), email)
	if err != nil {
		logger.LogError(fmt.Sprintf("Failed to fetch pre-signed URL for project file: %s", err.Error()), logrus.Fields{"email": email})
	} else {
		imageURL = url
	}
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	// Prepare the response
	response := v1.ProjectResponse{
		ID:          updatedProject.ID,
		Name:        updatedProject.Name,
		Slug:        updatedProject.Slug,
		Description: updatedProject.Description,
		Client: v1.Client{
			ID:            updatedProject.Client.ID,
			Name:          updatedProject.Client.Name,
			ManagerEmails: updatedProject.Client.ManagerEmails,
			Country:       updatedProject.Client.Country,
			CreatedAt:     updatedProject.Client.CreatedAt,
			UpdatedAt:     updatedProject.Client.UpdatedAt,
		},
		StartDate: updatedProject.StartDate,
		EndDate:   updatedProject.EndDate,
		Status:    updatedProject.Status,
		Tags:      updatedProject.Tags,
		CreatedAt: updatedProject.CreatedAt,
		UpdatedAt: updatedProject.UpdatedAt,
		CreatedBy: updatedProject.CreatedBy,
		CoverURL:  imageURL,
	}

	models.SendSuccessResponse(c, http.StatusOK, response, "Project updated successfully.")

}

// DeleteProjectByID godoc
func DeleteProjectByID(c *gin.Context) {
	id := c.Param("project_id")

	// Retrieve email from context
	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return // The response is already sent by the helper, so just return
	}

	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return
	}

	// Find the project by ID and check permissions
	var project v1.Project
	if err := tx.Where("id = ? AND deleted_at IS NULL AND created_by = ?", id, email).First(&project).Error; err != nil {
		tx.Rollback()
		logger.LogError(fmt.Sprintf("Project with ID: %s not found.", id), logrus.Fields{"error": err.Error(), "email": email})
		if err == gorm.ErrRecordNotFound {
			models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		} else {
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		}
		return
	}

	now := time.Now()
	project.DeletedAt = &now
	if err := tx.Save(&project).Error; err != nil {
		tx.Rollback()
		logger.LogError("Failed to archive project.", logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		return
	}

	if !utils.CommitTransaction(tx, c, email) {
		return
	}
	models.SendSuccessResponse(c, http.StatusNoContent, nil, "Project deleted successfully.")
}

// ListProjects godoc
func ListProjects(c *gin.Context) {
	var projects []v1.Project

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

	// Extract query parameters
	name := c.Query("name")
	clientID := c.Query("client_id")
	status := c.Query("status")
	priority := c.Query("priority")
	tag := c.Query("tag")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	// Start a transaction using the helper
	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return // Early return if the transaction failed to start
	}

	query := tx.Model(&v1.Project{}).
		Select("projects.id, projects.name, projects.slug, projects.description, projects.client_id, projects.start_date, projects.end_date, projects.status, projects.tags, projects.created_by, projects.created_at, projects.updated_at, projects.cover_page_id, "+
			"STRING_AGG(project_members.email, ',') AS member_emails, STRING_AGG(project_members.role, ',') AS member_roles").
		Joins("LEFT JOIN project_members ON project_members.project_id = projects.id").
		Where("projects.deleted_at IS NULL").
		Where("projects.created_by = ? OR project_members.email = ?", email, email).
		Where("project_members.role IN (?) OR project_members.role IS NULL", []string{"Manager", "Watcher", "Contributor"}).
		Group("projects.id")

	// Log the raw SQL query
	logger.LogInfo("Generated SQL Query", logrus.Fields{
		"query": query.Debug().Statement.SQL.String(),
	})

	// Apply additional filters based on query parameters
	if name != "" {
		query = query.Where("projects.name ILIKE ?", "%"+name+"%")
	}

	if clientID != "" {
		userID, err := utils.ConvertID(clientID, c, email, "client id")
		if err != nil {
			return
		}
		query = query.Where("projects.client_id = ?", userID)
	}

	if status != "" {
		query = query.Where("projects.status = ?", status)
	}

	if priority != "" {
		query = query.Where("projects.priority = ?", priority)
	}

	if tag != "" {
		query = query.Where("projects.tags @> ?", pq.StringArray{tag})
	}

	layout := "2006-01-02"
	if startDate != "" {
		startDateParsed, err := time.Parse(layout, startDate)
		if err != nil {
			models.SendErrorResponse(c, http.StatusUnprocessableEntity, "Start date is not in correct format..")
			return
		}
		query = query.Where("projects.start_date >= ?", startDateParsed)
	}

	if endDate != "" {
		endDateParsed, err := time.Parse(layout, endDate)
		if err != nil {
			models.SendErrorResponse(c, http.StatusUnprocessableEntity, "End date is not in correct format..")
			return
		}
		query = query.Where("projects.end_date <= ?", endDateParsed)
	}

	// Enable debugging to log the SQL query generated by GORM
	query = query.Debug()

	// Execute the query with pagination
	if err := query.Scopes(utils.Paginate(query, pagination)).Scan(&projects).Error; err != nil {
		tx.Rollback()
		logger.LogError("Failed to list projects.", logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		return
	}

	// Manually load the Client data for each project
	for i := range projects {
		if err := tx.Model(&projects[i]).Association("Client").Find(&projects[i].Client); err != nil {
			tx.Rollback()
			logger.LogError("Failed to load client data.", logrus.Fields{"error": err.Error(), "email": email})
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
			return
		}
	}

	// Prepare response data with image URLs
	var response []v1.ProjectResponse
	for _, project := range projects {
		var imageURL string
		url, err := utils.FetchPreSignedURLByProjectFileID(c, tx, project.CoverPageID.String(), email)
		if err != nil {
			logger.LogError(fmt.Sprintf("Failed to fetch pre-signed URL for project file: %s", err.Error()), logrus.Fields{"email": email})
		} else {
			imageURL = url
		}

		response = append(response, v1.ProjectResponse{
			ID:          project.ID,
			Name:        project.Name,
			Slug:        project.Slug,
			Description: project.Description,
			Client: v1.Client{
				ID:            project.Client.ID,
				Name:          project.Client.Name,
				ManagerEmails: project.Client.ManagerEmails,
				Country:       project.Client.Country,
				CreatedAt:     project.Client.CreatedAt,
				UpdatedAt:     project.Client.UpdatedAt,
			},
			StartDate: project.StartDate,
			EndDate:   project.EndDate,
			Status:    project.Status,
			Tags:      project.Tags,
			CreatedAt: project.CreatedAt,
			UpdatedAt: project.UpdatedAt,
			CreatedBy: project.CreatedBy,
			CoverURL:  imageURL,
		})
	}

	// Commit transaction
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	// Send paginated response
	responses := v1.ListProjectResponse{
		Data: response,
	}

	if responses.Data == nil {
		responses.Data = []v1.ProjectResponse{}
	}

	meta := models.PaginationMeta{
		Total: pagination.TotalCount,
		Page:  pagination.Page,
		Limit: pagination.PageSize,
	}

	models.SendPaginatedSuccessResponse(c, responses.Data, meta, "Projects retrieved successfully.")
}

// GetProjectStatsByID retrieves statistical data for a specific project by its ID.
// This includes metrics like contributions, activity levels, and related stats.
func GetProjectStatsByID(c *gin.Context) {
	// TODO: Implement logic to fetch project statistics by ID.
}
