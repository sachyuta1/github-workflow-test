package v1

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

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
)

// UploadIssueFiles handles uploading multiple files for a project
func UploadIssueFiles(c *gin.Context) {
	projectID := c.Param("project_id")
	issueID := c.Param("issue_id")

	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return
	}

	convertedProjectID, err := utils.ConvertID(projectID, c, email, "project id")
	if err != nil {
		return
	}

	convertedIssueID, err := utils.ConvertID(issueID, c, email, "issue id")
	if err != nil {
		return
	}

	// Start a database transaction
	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return
	}

	//	Check if the user is authorized to create an Issue
	authorized := utils.CanUserCreateIssue(tx, convertedProjectID, email)
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

	var issue v1.Issue
	if err := tx.Where("id = ? AND project_id = ? AND deleted_at IS NULL", convertedIssueID, convertedProjectID).First(&issue).Error; err != nil {
		tx.Rollback()
		logger.LogError(fmt.Sprintf("Issue not found with ID: %s", issueID), logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusNotFound, "Issue not found")
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

	var uploadedFiles []v1.IssueFileResponse

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

		fileName := fmt.Sprintf("issues/%s/files/%s%s", issueID, fileID.String(), ext)

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

		issueFile := v1.IssueFile{
			ID:         fileID,
			IssueID:    issue.ID,
			ProjectID:  project.ID,
			FileName:   fileHeader.Filename,
			FilePath:   fileName,
			FileSize:   fileHeader.Size,
			FileType:   fileHeader.Header.Get("Content-Type"),
			UploadedBy: email,
			CreatedAt:  time.Now(),
		}

		if err := tx.Create(&issueFile).Error; err != nil {
			tx.Rollback()
			logger.LogError(fmt.Sprintf("Failed to save file metadata: %s", fileHeader.Filename), logrus.Fields{"error": err.Error(), "email": email})
			models.SendErrorResponse(c, http.StatusInternalServerError, "Failed to save file metadata")
			return
		}

		uploadedFiles = append(uploadedFiles, v1.IssueFileResponse{
			ID:         issueFile.ID.String(),
			IssueID:    issueFile.IssueID.String(),
			ProjectID:  issueFile.ProjectID.String(),
			FileName:   issueFile.FileName,
			FilePath:   issueFile.FilePath,
			FileSize:   issueFile.FileSize,
			FileType:   issueFile.FileType,
			UploadedBy: issueFile.UploadedBy,
			CreatedAt:  issueFile.CreatedAt,
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

// GetIssueFiles retrieves all files for a project
func GetIssueFiles(c *gin.Context) {
	projectID := c.Param("project_id")
	issueID := c.Param("issue_id")

	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return
	}

	convertedProjectID, err := utils.ConvertID(projectID, c, email, "project id")
	if err != nil {
		return
	}

	convertedIssueID, err := utils.ConvertID(issueID, c, email, "issue id")
	if err != nil {
		return
	}

	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return
	}

	//	Check if the user is authorized to create an Issue
	authorized := utils.CanUserCreateIssue(tx, convertedProjectID, email)
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

	var issue v1.Issue
	if err := tx.Where("id = ? AND project_id = ? AND deleted_at IS NULL", convertedIssueID, convertedProjectID).First(&issue).Error; err != nil {
		tx.Rollback()
		logger.LogError(fmt.Sprintf("Issue not found with ID: %s", issueID), logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusNotFound, "Issue not found")
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
	query := tx.Model(&v1.IssueFile{}).Where("project_id = ? AND issue_id = ? AND deleted_at IS NULL", convertedProjectID, convertedIssueID)

	// Fetch paginated project files from the database
	var issueFiles []v1.IssueFile
	if err := query.Scopes(utils.Paginate(query, pagination)).Scan(&issueFiles).Error; err != nil {
		tx.Rollback()
		logger.LogError("Failed to retrieve project files", logrus.Fields{"error": err.Error(), "project_id": projectID, "email": email})
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
	// fmt.Sprintf("issues/%s/files/%s%s", issueID, fileID.String(), ext)
	var fileResponses []v1.IssueFileResponse
	for _, file := range issueFiles {
		// presignedURL, err := mcclient.GetPresignedURL(c, projectID, "files/"+file.FileName, time.Duration(24)*time.Hour)
		presignedURL, err := mcclient.GetPresignedURL(c, projectID, fmt.Sprintf("issues/%s/files/", issueID)+file.FileName, time.Duration(24)*time.Hour)
		if err != nil {
			logger.LogError("Failed to generate pre-signed URL", logrus.Fields{"error": err.Error(), "file_name": file.FileName, "email": email})
			continue // Skip this file, but proceed with others
		}

		fileResponse := v1.IssueFileResponse{
			ID:         file.ID.String(),
			IssueID:    file.IssueID.String(),
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

	models.SendPaginatedSuccessResponse(c, fileResponses, meta, "Issues file retrieved successfully.")
}

// DeleteIssueFileByID deletes a file from a project
func DeleteIssueFileByID(c *gin.Context) {
	projectID := c.Param("project_id")
	issueID := c.Param("issue_id")
	fileID := c.Param("file_id")

	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return
	}

	parsedProjectID, err := utils.ConvertID(projectID, c, email, "project id")
	if err != nil {
		return
	}

	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return
	}

	//	Check if the user is authorized to create an Issue
	authorized := utils.CanUserCreateIssue(tx, parsedProjectID, email)
	if !authorized {
		models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
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

	var issue v1.Issue
	if err := tx.Where("id = ? AND project_id = ? AND deleted_at IS NULL", issueID, projectID).First(&issue).Error; err != nil {
		tx.Rollback()
		logger.LogError(fmt.Sprintf("Issue not found with ID: %s", issueID), logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusNotFound, "Issue not found")
		return
	}

	// Soft delete the project state
	now := time.Now()

	var file v1.IssueFile
	if err := tx.Debug().Where("id = ? AND issue_id = ? AND project_id = ? AND uploaded_by = ? AND deleted_at IS NULL", fileID, issueID, projectID, email).First(&file).Error; err != nil {
		tx.Rollback()
		logger.LogError(fmt.Sprintf("File not found with ID: %s", fileID), logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusNotFound, "File not found")
		return
	}

	file.DeletedAt = &now

	if err := tx.Save(&file).Error; err != nil {
		tx.Rollback()
		logger.LogError(fmt.Sprintf("Failed to delete file with ID: %s", fileID), logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusInternalServerError, "Failed to delete file")
		return
	}

	// Commit transaction and respond
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	models.SendSuccessResponse(c, http.StatusOK, nil, "File successfully marked as deleted.")

}
