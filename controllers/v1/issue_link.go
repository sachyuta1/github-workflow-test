package v1

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/san-data-systems/common/errors"
	"github.com/san-data-systems/common/logger"
	"github.com/san-data-systems/common/models"
	v1 "github.com/san-data-systems/common/models/v1"
	"github.com/san-data-systems/common/utils"
	"github.com/sirupsen/logrus"
)

// CreateIssueLink creates a new IssueLink entry.
func CreateIssueLink(c *gin.Context) {

	// Retrieve email from context
	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return // The response is already sent by the helper, so just return
	}

	issueID := c.Param("issue_id")
	projectID := c.Param("project_id")

	// Convert issue ID and handle any conversion errors
	parsedIssueID, err := utils.ConvertID(issueID, c, email, "issue id")
	if err != nil {
		return // Early return if conversion fails, error response is already handled
	}

	// Convert project ID and handle any conversion errors
	parsedProjectID, err := utils.ConvertID(projectID, c, email, "project id")
	if err != nil {
		return // Early return if conversion fails, error response is already handled
	}

	var request v1.IssueLinkRequest

	// Bind the request to the IssueLink struct
	if !utils.BindJSONRequest(c, &request, email) {
		return
	}

	// Start a transaction using the helper
	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return // Early return if the transaction failed to start
	}

	//	Check if the user is authorized to create an Issue
	authorized := utils.CanUserCreateIssue(tx, parsedProjectID, email)
	if !authorized {
		models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		return
	}

	// Check if the URL already exists for this Issue
	var existingLink v1.IssueLink
	if err := tx.Where("issue_id = ? AND URL = ? AND project_id = ?", issueID, request.URL, projectID).
		First(&existingLink).Error; err == nil {
		tx.Rollback()
		logger.LogError("URL already exists for this Issue", nil)
		models.SendErrorResponse(c, http.StatusConflict, "URL already exists for this Issue")
		return
	}

	// Create the IssueLink entry
	link := v1.IssueLink{
		ProjectID: parsedProjectID,
		Title:     request.Title,
		URL:       request.URL,
		CreatedBy: email,
		IssueID:   parsedIssueID,
	}

	// Create the IssueLink entry
	if !utils.CreateWithRollback(tx, c, &link, "Failed to create link", email) {
		return
	}

	// Commit the transaction
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	// Return the response
	response := v1.IssueLinkResponse{
		ID:        link.ID.String(),
		ProjectID: link.ProjectID.String(),
		Title:     link.Title,
		IssueID:   link.IssueID.String(),
		URL:       link.URL,
		CreatedAt: link.CreatedAt,
		UpdatedAt: link.UpdatedAt,
		CreatedBy: link.CreatedBy,
		UpdatedBy: link.UpdatedBy,
	}

	// Send the response
	models.SendSuccessResponse(c, http.StatusCreated, response, "Link created successfully")

}

// ListIssueLinks retrieves a list of IssueLink entries.
func ListIssueLinks(c *gin.Context) {
	issueID := c.Param("issue_id")
	projectID := c.Param("project_id")

	// Retrieve email from context
	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return // The response is already sent by the helper, so just return
	}

	parsedIssueID, _ := utils.ConvertID(issueID, c, email, "issue id")
	parsedProjectID, _ := utils.ConvertID(projectID, c, email, "project id")

	// Start a transaction using the helper
	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return // Early return if the transaction failed to start
	}

	//	Check if the user is authorized to create an Issue
	authorized, role := utils.IsUserPartOfRole(tx, projectID, email)
	if !authorized || (*role != "Manager" && *role != "Owner") {
		models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		return
	}

	// Parse pagination parameters (page, page_size) using utility function
	pagination, err := utils.ParsePagination(c)
	if err != nil {
		logger.LogError("Invalid pagination parameters.", logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusBadRequest, errors.ErrBadRequest)
		return
	}

	query := tx.Model(&v1.IssueLink{}).Where("issue_id = ? AND project_id = ? AND deleted_at IS NULL", parsedIssueID, parsedProjectID)

	// Retrieve a list of IssueLink entries
	var links []v1.IssueLink
	if err := query.
		Select("id, project_id, title, issue_id, url, created_at, updated_at, created_by, updated_by").
		Scopes(utils.Paginate(query, pagination)).Scan(&links).Error; err != nil {
		tx.Rollback()
		logger.LogError("Failed to retrieve links", nil)
		models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		return
	}

	// Commit the transaction
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	// Create a response object
	var responses []v1.IssueLinkResponse
	for _, link := range links {
		responses = append(responses, v1.IssueLinkResponse{
			ID:        link.ID.String(),
			ProjectID: link.ProjectID.String(),
			Title:     link.Title,
			IssueID:   link.IssueID.String(),
			URL:       link.URL,
			CreatedAt: link.CreatedAt,
			UpdatedAt: link.UpdatedAt,
			CreatedBy: link.CreatedBy,
			UpdatedBy: link.UpdatedBy,
		})
	}

	response := v1.ListIssueLinksResponse{
		Data: responses,
	}
	meta := models.PaginationMeta{
		Total: pagination.TotalCount,
		Page:  pagination.Page,
		Limit: pagination.PageSize,
	}

	// Send success response back to the client
	models.SendPaginatedSuccessResponse(c, response.Data, meta, "Issues retrieved successfully.")

}

// GetIssueLinkByID retrieves a specific IssueLink entry by ID.
func GetIssueLinkByID(c *gin.Context) {
	// Retrieve email from context
	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return // The response is already sent by the helper, so just return
	}

	linkID := c.Param("link_id")
	issueID := c.Param("issue_id")
	projectID := c.Param("project_id")

	// Convert issue ID and handle any conversion errors
	parsedLinkID, _ := utils.ConvertID(linkID, c, email, "link id")
	parsedIssueID, _ := utils.ConvertID(issueID, c, email, "issue id")
	parsedProjectID, _ := utils.ConvertID(projectID, c, email, "project id")

	// Start a transaction using the helper
	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return // Early return if the transaction failed to start
	}

	var link v1.IssueLink
	if err := tx.Where("issue_id = ? AND deleted_at IS NULL AND id = ? AND project_id = ?", parsedIssueID, parsedLinkID, parsedProjectID).First(&link).Error; err != nil {
		tx.Rollback()
		logger.LogError("Failed to retrieve link", nil)
		models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		return
	}

	// Commit the transaction
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	response := v1.IssueLinkResponse{
		ID:        link.ID.String(),
		ProjectID: link.ProjectID.String(),
		Title:     link.Title,
		IssueID:   link.IssueID.String(),
		URL:       link.URL,
		CreatedAt: link.CreatedAt,
		UpdatedAt: link.UpdatedAt,
		CreatedBy: link.CreatedBy,
		UpdatedBy: link.UpdatedBy,
	}

	// Send the response
	models.SendSuccessResponse(c, http.StatusOK, response, "Link retrieved successfully")

}

// UpdateIssueLinkByID updates a specific IssueLink entry by ID.
func UpdateIssueLinkByID(c *gin.Context) {
	// Retrieve email from context
	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return // The response is already sent by the helper, so just return
	}

	// Retrieve the URL parameters
	linkID := c.Param("link_id")
	issueID := c.Param("issue_id")
	projectID := c.Param("project_id")

	// Convert issue ID and handle any conversion errors
	parsedLinkID, _ := utils.ConvertID(linkID, c, email, "link id")
	parsedIssueID, _ := utils.ConvertID(issueID, c, email, "issue id")
	parsedProjectID, _ := utils.ConvertID(projectID, c, email, "project id")

	// Bind the request to the IssueLink struct
	var request v1.IssueLinkRequest
	if !utils.BindJSONRequest(c, &request, email) {
		return
	}

	// Start a transaction using the helper
	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return // Early return if the transaction failed to start
	}

	var link v1.IssueLink
	if err := tx.Debug().Where("issue_id = ? AND deleted_at IS NULL AND id = ? AND project_id = ?", parsedIssueID, parsedLinkID, parsedProjectID).First(&link).Error; err != nil {
		tx.Rollback()
		logger.LogError("Failed to retrieve link", nil)
		models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		return
	}
	// Check if the user is authorized to create an Issue
	authorized := utils.CanUserCreateIssue(tx, parsedProjectID, email)
	if !authorized {
		models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		return
	}

	if request.Title != "" {
		link.Title = request.Title
	}

	if request.URL != "" {
		link.URL = request.URL
	}

	link.UpdatedBy = email

	// Save the updated IssueLink entry
	if err := tx.Save(&link).Error; err != nil {
		tx.Rollback()
		logger.LogError("Failed to update link", nil)
		models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		return
	}

	// Commit the transaction
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	response := v1.IssueLinkResponse{
		ID:        link.ID.String(),
		ProjectID: link.ProjectID.String(),
		Title:     link.Title,
		IssueID:   link.IssueID.String(),
		URL:       link.URL,
		CreatedAt: link.CreatedAt,
		UpdatedAt: link.UpdatedAt,
		CreatedBy: link.CreatedBy,
		UpdatedBy: link.UpdatedBy,
	}

	// Send the response
	models.SendSuccessResponse(c, http.StatusOK, response, "Link updated successfully")

}

// DeleteIssueLink deletes a specific IssueLink entry by ID.
func DeleteIssueLink(c *gin.Context) {
	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return // The response is already sent by the helper, so just return
	}

	// Retrieve the URL parameters
	linkID := c.Param("link_id")
	issueID := c.Param("issue_id")
	projectID := c.Param("project_id")

	// Convert issue ID and handle any conversion errors
	parsedLinkID, _ := utils.ConvertID(linkID, c, email, "link id")
	parsedIssueID, _ := utils.ConvertID(issueID, c, email, "issue id")
	parsedProjectID, _ := utils.ConvertID(projectID, c, email, "project id")

	// Start a transaction using the helper
	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return // Early return if the transaction failed to start
	}

	var link v1.IssueLink
	if err := tx.Where("issue_id = ? AND deleted_at IS NULL AND id = ? AND project_id = ?", parsedIssueID, parsedLinkID, parsedProjectID).First(&link).Error; err != nil {
		logger.LogError("Failed to retrieve link", nil)
		models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		return
	}

	//	Check if the user is authorized to create an Issue
	authorized := utils.CanUserCreateIssue(tx, parsedProjectID, email)
	if !authorized {
		models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		return
	}

	// Soft delete the IssueLink entry
	now := time.Now()
	link.DeletedAt = &now

	// Save the updated IssueLink entry
	if err := tx.Save(&link).Error; err != nil {
		tx.Rollback()
		logger.LogError("Failed to delete link", nil)
		models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		return
	}

	// Commit the transaction
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	// Send the response
	models.SendSuccessResponse(c, http.StatusOK, nil, "Link deleted successfully")

}
