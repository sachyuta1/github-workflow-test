package v1

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/san-data-systems/common/clients/minio"
	"github.com/san-data-systems/common/errors"
	"github.com/san-data-systems/common/logger"
	"github.com/san-data-systems/common/models"
	v1 "github.com/san-data-systems/common/models/v1"
	"github.com/san-data-systems/common/utils"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

// ChangeProjectCoverImageByID handles the file upload for changing a project's cover image by ID
func ChangeProjectCoverImageByID(c *gin.Context) {
	id := c.Param("project_id")

	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return
	}

	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return
	}

	// Retrieve the project from the database
	var project v1.Project
	if err := tx.Where("id = ? AND created_by = ?", id, email).First(&project).Error; err != nil {
		tx.Rollback()
		logger.LogError(fmt.Sprintf("Project not found with ID: %s", id), logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusNotFound, "Project not found")
		return
	}

	ProjectID, err := utils.ConvertID(id, c, email, "project id")
	if err != nil {
		return // Early return if conversion fails, error response is already handled
	}
	file, fileHeader, err := c.Request.FormFile("image")
	if err != nil {
		models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		return
	}
	defer file.Close()
	fileType := fileHeader.Header.Get("Content-Type") // Get the file type
	if fileType != "image/jpeg" && fileType != "image/png" {
		models.SendErrorResponse(c, http.StatusBadRequest, errors.ErrBadRequest)
	}

	// Handle file upload logic
	imageURL, projectFileID, err := utils.UpdateCoverFileUpload(c, tx, email, ProjectID, *project.CoverPageID)
	if err != nil {
		logger.LogError(fmt.Sprintf("Error during file upload: %v", err), logrus.Fields{"error": err.Error(), "email": email})
	}

	// Update only the "Name" field
	if err := tx.Model(&project).Update("cover_page_id", projectFileID).Error; err != nil {
		tx.Rollback()
		logger.LogError(fmt.Sprintf("Error updating project with ID: %s", project.ID), logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		return
	}

	// Commit transaction
	if !utils.CommitTransaction(tx, c, email) {
		return
	}
	data := map[string]string{
		"image": imageURL,
	}

	models.SendSuccessResponse(c, http.StatusCreated, data, "Project cover page is changed.")
}

// UploadProjectFiles handles uploading multiple files for a project
func UploadProjectFiles(c *gin.Context) {
	projectID := c.Param("project_id")

	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return
	}

	convertedProjectID, err := utils.ConvertID(projectID, c, email, "project id")
	if err != nil {
		return
	}

	// Start a database transaction
	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return
	}

	authorized, _ := utils.IsUserPartOfRole(tx, projectID, email)
	if !authorized {
		models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		return
	}

	// Retrieve the project to ensure it exists
	var project v1.Project
	if err := tx.Where("id = ?", convertedProjectID).First(&project).Error; err != nil {
		tx.Rollback()
		logger.LogError(fmt.Sprintf("Project not found with ID: %s", projectID), logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusNotFound, "Project not found")
		return
	}

	// Retrieve multiple files from the form
	form, err := c.MultipartForm()
	if err != nil {
		tx.Rollback()
		logger.LogError("Unable to parse form", logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusBadRequest, "Unable to parse form")
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		tx.Rollback()
		models.SendSuccessResponse(c, http.StatusCreated, nil, "No files are selected.")
		return
	}

	var uploadedFiles []v1.ProjectFileResponse

	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			tx.Rollback()
			logger.LogError(fmt.Sprintf("Failed to open file: %s", fileHeader.Filename), logrus.Fields{"error": err.Error(), "email": email})
			models.SendErrorResponse(c, http.StatusInternalServerError, "Failed to process file")
			return
		}
		ext := strings.ToLower(filepath.Ext(fileHeader.Filename))

		defer file.Close()

		fileContent, err := io.ReadAll(file)
		if err != nil {
			tx.Rollback()
			logger.LogError(fmt.Sprintf("Failed to read file: %s", fileHeader.Filename), logrus.Fields{"error": err.Error(), "email": email})
			models.SendErrorResponse(c, http.StatusInternalServerError, "Failed to read file")
			return
		}

		mcclient, err := minio.GetMinIOClient()
		if err != nil {
			tx.Rollback()
			logger.LogError("Failed to get MinIO client", logrus.Fields{"error": err.Error(), "email": email})
			models.SendErrorResponse(c, http.StatusInternalServerError, "Failed to initialize storage")
			return
		}
		fileID := uuid.New()

		fileName := "files/" + fmt.Sprintf("%s%s", fileID.String(), ext)

		// Upload file to MinIO
		err = mcclient.UploadFile(c, projectID, fileName, fileContent)
		if err != nil {
			tx.Rollback()
			logger.LogError(fmt.Sprintf("Failed to upload file: %s", fileName), logrus.Fields{"error": err.Error(), "email": email})
			models.SendErrorResponse(c, http.StatusInternalServerError, "Failed to upload file")
			return
		}

		// Get presigned URL for the uploaded file
		fileURL, err := mcclient.GetPresignedURL(c, projectID, fileName, time.Duration(24)*time.Hour)
		if err != nil {
			tx.Rollback()
			logger.LogError(fmt.Sprintf("Failed to get presigned URL for file: %s", fileName), logrus.Fields{"error": err.Error(), "email": email})
			models.SendErrorResponse(c, http.StatusInternalServerError, "Failed to generate file URL")
			return
		}

		projectFile := v1.ProjectFile{
			ID:         fileID,
			ProjectID:  project.ID,
			FileName:   fileHeader.Filename,
			FilePath:   fileName,
			FileSize:   fileHeader.Size,
			FileType:   fileHeader.Header.Get("Content-Type"),
			UploadedBy: email,
			CreatedAt:  time.Now(),
		}

		if err := tx.Create(&projectFile).Error; err != nil {
			tx.Rollback()
			logger.LogError(fmt.Sprintf("Failed to save file metadata: %s", fileHeader.Filename), logrus.Fields{"error": err.Error(), "email": email})
			models.SendErrorResponse(c, http.StatusInternalServerError, "Failed to save file metadata")
			return
		}

		uploadedFiles = append(uploadedFiles, v1.ProjectFileResponse{
			ID:         projectFile.ID.String(),
			ProjectID:  projectFile.ProjectID.String(),
			FileName:   projectFile.FileName,
			FilePath:   projectFile.FilePath,
			FileSize:   projectFile.FileSize,
			FileType:   projectFile.FileType,
			UploadedBy: projectFile.UploadedBy,
			CreatedAt:  projectFile.CreatedAt,
			URL:        fileURL,
		})
	}

	// Commit transaction
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	// Send success response
	models.SendSuccessResponse(c, http.StatusCreated, uploadedFiles, "Files uploaded successfully.")
}

// GetProjectFiles retrieves files associated with a given project ID, with pagination support.
func GetProjectFiles(c *gin.Context) {
	projectID := c.Param("project_id")

	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return
	}

	ProjectID, err := utils.ConvertID(projectID, c, email, "project id")
	if err != nil {
		return // Error response is already handled in ConvertID
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

	// Retrieve project to ensure the user has access
	var project v1.Project
	if err := tx.Where(
		"id = ? AND deleted_at IS NULL",
		projectID,
	).First(&project).Error; err != nil {
		tx.Rollback()
		logger.LogError(fmt.Sprintf("Project not found with ID: %s", projectID), logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusNotFound, "Project not found")
		return
	}

	// Parse pagination parameters
	pagination, err := utils.ParsePagination(c)
	if err != nil {
		logger.LogError("Failed to parse pagination parameters", logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusBadRequest, "Invalid pagination parameters")
		tx.Rollback()
		return
	}
	query := tx.Debug().Model(&v1.ProjectFile{}).Where(
		"deleted_at IS NULL",
	).Where(
		"id != ? AND project_id = ?  ", project.CoverPageID, projectID,
	)

	// Fetch paginated project files from the database
	var projectFiles []v1.ProjectFile
	if err := query.Scopes(utils.Paginate(query, pagination)).Scan(&projectFiles).Error; err != nil {
		tx.Rollback()
		logger.LogError("Failed to retrieve project files", logrus.Fields{"error": err.Error(), "project_id": ProjectID, "email": email})
		models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		return
	}

	// Generate pre-signed URLs for the files
	mcclient, err := minio.GetMinIOClient()
	if err != nil {
		tx.Rollback()
		logger.LogError("Failed to initialize MinIO client", logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		return
	}

	var fileResponses []v1.ProjectFileResponse
	for _, file := range projectFiles {
		presignedURL, err := mcclient.GetPresignedURL(c, projectID, "files/"+file.FileName, time.Duration(24)*time.Hour)
		if err != nil {
			logger.LogError("Failed to generate pre-signed URL", logrus.Fields{"error": err.Error(), "file_name": file.FileName, "email": email})
			continue // Skip this file, but proceed with others
		}

		fileResponse := v1.ProjectFileResponse{
			ID:         file.ID.String(),
			ProjectID:  file.ProjectID.String(),
			FileName:   file.FileName,
			FilePath:   file.FilePath,
			FileSize:   file.FileSize,
			FileType:   file.FileType,
			UploadedBy: file.UploadedBy,
			URL:        presignedURL,
			CreatedAt:  file.CreatedAt,
		}
		fileResponses = append(fileResponses, fileResponse)
	}

	// Commit the transaction
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	meta := models.PaginationMeta{
		Total: pagination.TotalCount,
		Page:  pagination.Page,
		Limit: pagination.PageSize,
	}

	models.SendPaginatedSuccessResponse(c, fileResponses, meta, "Projects file retrieved successfully.")
}

// DeleteProjectFileByID retrieves files associated with a given project ID, with pagination support.
func DeleteProjectFileByID(c *gin.Context) {
	projectID := c.Param("project_id")
	fileID := c.Param("file_id")

	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return
	}

	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return
	}

	// Find the project by ID and check permissions
	var project v1.Project
	if err := tx.Where("id = ? AND deleted_at IS NULL", projectID).First(&project).Error; err != nil {
		tx.Rollback()
		logger.LogError(fmt.Sprintf("Project with ID: %s not found.", projectID), logrus.Fields{"error": err.Error(), "email": email})
		if err == gorm.ErrRecordNotFound {
			models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		} else {
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		}
		return
	}

	// Soft delete the project state
	now := time.Now()

	// Find the projectFile and update is_deleted to true
	if err := tx.Model(&v1.ProjectFile{}).
		Where("id = ? AND project_id = ? AND uploaded_by = ? AND deleted_at IS NULL", fileID, projectID, email).
		Update("deleted_at", now).Error; err != nil {
		tx.Rollback()
		logger.LogError(fmt.Sprintf("Failed to update file with ID: %s.", fileID), logrus.Fields{"error": err.Error(), "email": email})
		if err == gorm.ErrRecordNotFound {
			models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		} else {
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		}
		return
	}

	// Commit transaction and respond
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	models.SendSuccessResponse(c, http.StatusOK, nil, "File successfully marked as deleted.")

}
