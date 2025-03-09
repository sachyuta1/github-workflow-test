package v1

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/san-data-systems/common/errors"
	"github.com/san-data-systems/common/logger"
	"github.com/san-data-systems/common/models"
	v1 "github.com/san-data-systems/common/models/v1"
	"github.com/san-data-systems/common/utils"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"net/http"
)

// CheckProjectSlugExists checks if a project with the provided slug exists.
func CheckProjectSlugExists(c *gin.Context) {
	slug := c.Param("slug") // Get the slug from URL parameter

	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return
	}

	var project v1.Project
	// Start a transaction using the helper
	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return // Early return if the transaction failed to start
	}
	if err := tx.Where("slug = ?", slug).First(&project).Error; err != nil {
		tx.Rollback()
		logger.LogError(fmt.Sprintf("Project with slug: %s not found.", slug), logrus.Fields{"error": err.Error(), "email": email})
		if err == gorm.ErrRecordNotFound {
			models.SendErrorResponse(c, http.StatusOK, errors.ErrRecordNotFound)
		} else {
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		}
		return
	}
	// If the project exists, return a success response
	models.SendSuccessResponse(c, http.StatusOK, nil, "Project with the given slug exists.")
}

// GetProjectBySlug checks if a project with the provided slug exists.
func GetProjectBySlug(c *gin.Context) {

	slug := c.Param("slug") // Get the slug from the request URL.
	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return
	}
	var project v1.Project
	// Start a transaction using the helper
	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return // Early return if the transaction failed to start
	}

	if err := tx.Where("slug = ? AND deleted_at IS NULL", slug).First(&project).Error; err != nil {
		tx.Rollback()
		logger.LogError(fmt.Sprintf("Project with slug: %s not found.", slug), logrus.Fields{"error": err.Error(), "email": email})
		if err == gorm.ErrRecordNotFound {
			models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		} else {
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		}
		return
	}
	authorized, role := utils.IsUserPartOfRole(tx, project.ID.String(), email)
	if !authorized && role == nil {
		models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		return
	}

	// Fetch the project only if it exists, is not archived, and is either public, created by the user, or the user is in viewers or editors
	if err := tx.Debug().Preload("Client").Where("slug = ? AND deleted_at IS NULL", slug).First(&project).Error; err != nil {
		tx.Rollback()
		logger.LogError(fmt.Sprintf("Project with slug: %s not found.", slug), logrus.Fields{"error": err.Error(), "email": email})
		// Check if the error is a record not found error
		if err == gorm.ErrRecordNotFound {
			models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		} else {
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		}
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

	// Attempt to commit the transaction
	if !utils.CommitTransaction(tx, c, email) {
		return // Early return if the commit failed
	}

	// Prepare the response
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

	models.SendSuccessResponse(c, http.StatusOK, response, "Project retrieved successfully.")

}
