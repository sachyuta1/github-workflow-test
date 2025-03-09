package v1

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"github.com/san-data-systems/common/errors"
	"github.com/san-data-systems/common/logger"
	"github.com/san-data-systems/common/models"
	v1 "github.com/san-data-systems/common/models/v1"
	"github.com/san-data-systems/common/utils"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CreateIssue godoc
func CreateIssue(c *gin.Context) {
	var issue v1.Issue

	projectID := c.Param("project_id")

	// Retrieve email from context
	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return
	}

	convertID, err := utils.ConvertID(projectID, c, email, "project id")
	if err != nil {
		return
	}

	var req v1.IssueRequest
	if !utils.BindJSONRequest(c, &req, email) {
		return
	}

	// Start a transaction using the helper
	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return
	}

	//	Check if the user is authorized to create an Issue
	authorized := utils.CanUserCreateIssue(tx, convertID, email)
	if !authorized {
		models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		return
	}

	// Check if the project exists
	err = tx.Where("id = ?", projectID).First(&v1.Project{}).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.LogError(fmt.Sprintf("failed to fetch project with ID %s", req.ProjectID), logrus.Fields{"error": err.Error(), "email": email})
			models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
			return
		}
		logger.LogError("failed to fetch project", logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		return
	}

	// Get next sequence ID
	var maxSeq struct {
		MaxSeq int
	}
	if err := tx.Model(&v1.Issue{}).
		Select("COALESCE(MAX(CAST(sequence_id AS INTEGER)), 0) as max_seq").
		Where("project_id = ? AND deleted_at IS NULL", projectID).
		Scan(&maxSeq).Error; err != nil {
		logger.LogError("Failed to generate sequence ID", logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		return
	}
	sequenceID := maxSeq.MaxSeq + 1

	// Lock the rows for update
	if err := tx.Model(&v1.Issue{}).
		Where("project_id = ? AND deleted_at IS NULL", projectID).
		Clauses(clause.Locking{Strength: "UPDATE"}).Error; err != nil {
		tx.Rollback()
		logger.LogError("Failed to lock rows for update", logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		return
	}

	layout := "2006-01-02"
	startDate, err := time.Parse(layout, req.StartDate)
	if err != nil {
		logger.LogError("Invalid start date format", logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusBadRequest, errors.ErrInternalServer)
		return
	}
	endDate, err := time.Parse(layout, req.EndDate)
	if err != nil {
		logger.LogError("Invalid end date format", logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusBadRequest, errors.ErrInternalServer)
		return
	}
	// Convert the project ID and handle any conversion errors
	parsedrojectID, err := utils.ConvertID(projectID, c, email, "project id")
	if err != nil {
		logger.LogError("Failed to convert project ID", logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		return
	}

	// Convert the state ID and handle any conversion errors
	stateID, err := utils.ConvertID(req.StateID, c, email, "state id")
	if err != nil {
		logger.LogError("Failed to convert state ID", logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		return
	}

	var parentIssue v1.Issue

	if req.ParentID != "" {

		parentID, err := utils.ConvertID(req.ParentID, c, email, "parent id")
		if err != nil {
			logger.LogError("Failed to convert parent ID", logrus.Fields{"error": err.Error(), "email": email})
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
			return
		}

		// check if the parent issue exists
		if err := tx.Where("id = ? AND deleted_at is NULL", parentID).First(&parentIssue).Error; err != nil {
			tx.Rollback()
			logger.LogError("Failed to fetch parent Issues from the database.", logrus.Fields{"error": err.Error(), "email": email})
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
			return
		}
	}

	// Create the Issue model
	issue = v1.Issue{
		Title:               req.Title,
		Description:         req.Description,
		ProjectID:           parsedrojectID,
		CreatedBy:           email,
		Priority:            req.Priority,
		UpdatedBy:           email,
		StartDate:           startDate,
		EndDate:             endDate,
		CompletedPercentage: req.CompletedPercentage,
		Point:               req.Point,
		LabelIDs:            req.LabelIDs,
		ParentID:            parentIssue.ID,
		CompletedAt:         nil,
		StateID:             stateID,
		SequenceID:          int32(sequenceID),
		EstimatedHours:      req.EstimatedHours,
	}

	// Create the Issue in the database
	if !utils.CreateWithRollback(tx, c, &issue, "Failed to create Issue", email) {
		return
	}

	var labels []v1.ProjectLabel
	var formattedLabels []map[string]string

	// Check if there are any label IDs to process
	if len(req.LabelIDs) > 0 {
		if err := tx.Where("id IN ? AND deleted_at is NULL", req.LabelIDs).Find(&labels).Error; err != nil {
			tx.Rollback()
			logger.LogError("Failed to fetch labels from the database.", logrus.Fields{"error": err.Error(), "email": email})
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
			return
		}

		formattedLabels = make([]map[string]string, len(labels))

		for i, label := range labels {
			formattedLabels[i] = map[string]string{
				"name":  label.Name,
				"color": label.Color,
			}
		}
	}

	// Fetch the project state by ID and associated project, ensuring the state exists for the specific project
	var state v1.ProjectState
	if err := tx.Debug().Where("id = ? AND project_id = ? AND deleted_at IS NULL", stateID, projectID).First(&state).Error; err != nil {
		tx.Rollback()
		logger.LogError(fmt.Sprintf("Project state with ID: %s not found for project ID: %s.", stateID, projectID), logrus.Fields{"error": err.Error(), "email": email})
		if err == gorm.ErrRecordNotFound {
			models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		} else {
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		}
		return
	}

	// Commit the transaction
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	// Prepare the response
	response := v1.IssueResponse{
		ID:                  utils.ConvertUUIDToString(issue.ID),
		Title:               issue.Title,
		Description:         issue.Description,
		ProjectID:           utils.ConvertUUIDToString(issue.ProjectID),
		CreatedBy:           email,
		SequenceID:          issue.SequenceID,
		Priority:            issue.Priority,
		UpdatedBy:           email,
		StartDate:           issue.StartDate,
		EndDate:             issue.EndDate,
		CompletedPercentage: issue.CompletedPercentage,
		Point:               issue.Point,
		Labels:              formattedLabels,
		ParentID:            issue.ParentID.String(),
		CreatedAt:           issue.CreatedAt,
		UpdatedAt:           issue.UpdatedAt,
		EstimatedHours:      issue.EstimatedHours,
		State:               v1.ProjectStateResponse(state),
	}
	if issue.CompletedAt != nil {
		response.CompletedAt = *issue.CompletedAt
	}
	models.SendSuccessResponse(c, http.StatusCreated, response, "Issue created successfully")

}

// UpdateIssueByID godoc
func UpdateIssueByID(c *gin.Context) {
	id := c.Param("issue_id")
	projectID := c.Param("project_id")

	// Retrieve email from context
	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return
	}

	convertID, err := utils.ConvertID(projectID, c, email, "project id")
	if err != nil {
		return
	}

	// Parse the request body
	var req v1.UpdateIssueRequest
	if !utils.BindJSONRequest(c, &req, email) {
		return
	}

	// Start a transaction using the helper
	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return
	}

	//	Check if the user is authorized to create an Issue
	authorized := utils.CanUserCreateIssue(tx, convertID, email)
	if !authorized {
		models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		return
	}

	// Check if the Issue exists
	var Issue v1.Issue
	if err := tx.Where("id = ? AND deleted_at is NULL AND project_id = ?", id, projectID).First(&Issue).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.LogError(fmt.Sprintf("failed to fetch Issue with ID %s", id), logrus.Fields{"error": err.Error(), "email": email})
			models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
			return
		}
		logger.LogError("failed to fetch Issue", logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		return
	}

	if req.Title != nil {
		Issue.Title = *req.Title
	}
	if req.Description != nil {
		Issue.Description = *req.Description
	}

	if req.Priority != nil {
		Issue.Priority = *req.Priority
	}

	if req.StartDate != nil {
		startDate, _ := time.Parse("2006-01-02", *req.StartDate)
		Issue.StartDate = startDate
	}
	if req.EndDate != nil {
		endDate, _ := time.Parse("2006-01-02", *req.EndDate)
		Issue.EndDate = endDate
	}
	if req.CompletedPercentage != nil {
		Issue.CompletedPercentage = *req.CompletedPercentage
	}

	if req.Point != nil {
		Issue.Point = *req.Point
	}

	if req.LabelIDs != nil {
		var labels []v1.ProjectLabel

		if err := tx.Where("id IN ? AND deleted_at is NULL", req.LabelIDs).Find(&labels).Error; err != nil {
			tx.Rollback()
			logger.LogError("Failed to fetch labels from the database.", logrus.Fields{"error": err.Error(), "email": email})
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
			return
		}
		Issue.LabelIDs = pq.StringArray(utils.ConvertStringPointersToStrings(req.LabelIDs))

	}

	if req.ParentID != nil {

		var Issues []v1.Issue
		if err := tx.Where("id = ? and deleted_at is NULL", *req.ParentID).Find(&Issues).Error; err != nil {
			tx.Rollback()
			logger.LogError("Failed to fetch parent Issues from the database.", logrus.Fields{"error": err.Error(), "email": email})
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
			return
		}

		parentID, err := utils.ConvertID(*req.ParentID, c, email, "parent id")
		if err != nil {
			logger.LogError("Failed to convert parent ID", logrus.Fields{"error": err.Error(), "email": email})
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
			return
		}

		Issue.ParentID = parentID
	}

	var state v1.ProjectState
	if req.StateID != nil {

		stateID, _ := utils.ConvertID(*req.StateID, c, email, "state")

		// Fetch the project state by ID and associated project, ensuring the state exists for the specific project
		if err := tx.Where("id = ? AND project_id = ? AND deleted_at IS NULL", stateID, projectID).First(&state).Error; err != nil {
			tx.Rollback()
			logger.LogError(fmt.Sprintf("Project state with ID: %s not found for project ID: %s.", Issue.StateID, projectID), logrus.Fields{"error": err.Error(), "email": email})
			if err == gorm.ErrRecordNotFound {
				models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
			} else {
				models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
			}
			return
		}

		Issue.StateID = stateID
	}

	if err := tx.Save(&Issue).Error; err != nil {
		models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		tx.Rollback()
		return
	}

	// Fetch the label
	var newLabels []v1.ProjectLabel
	if err := tx.Where("id = ANY(?) AND deleted_at is NULL", Issue.LabelIDs).Find(&newLabels).Error; err != nil {
		tx.Rollback()
		logger.LogError("Failed to fetch label from the database.", logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
	}

	// Format labels into a list of maps
	formatedLabels := make([]map[string]string, len(newLabels))
	for i, label := range newLabels {
		formatedLabels[i] = map[string]string{
			"name":  label.Name,
			"color": label.Color,
		}
	}

	// Fetch the project state by ID
	if err := tx.Where("id = ? AND project_id = ? AND deleted_at IS NULL", Issue.StateID, Issue.ProjectID).First(&state).Error; err != nil {
		tx.Rollback()
		logger.LogError(fmt.Sprintf("Project state with ID: %s not found for project ID: %s.", Issue.StateID, Issue.ProjectID), logrus.Fields{"error": err.Error(), "email": email})
		if err == gorm.ErrRecordNotFound {
			models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		} else {
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		}
		return
	}

	// Commit the transaction
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	response := v1.IssueResponse{
		ID:                  utils.ConvertUUIDToString(Issue.ID),
		Title:               Issue.Title,
		Description:         Issue.Description,
		ProjectID:           utils.ConvertUUIDToString(Issue.ProjectID),
		CreatedBy:           email,
		Priority:            Issue.Priority,
		UpdatedBy:           email,
		StartDate:           Issue.StartDate,
		EndDate:             Issue.EndDate,
		CompletedPercentage: Issue.CompletedPercentage,
		Point:               Issue.Point,
		ParentID:            utils.ConvertUUIDToString(Issue.ParentID),
		CreatedAt:           Issue.CreatedAt,
		UpdatedAt:           Issue.UpdatedAt,
		EstimatedHours:      Issue.EstimatedHours,
		State:               v1.ProjectStateResponse(state),
		Labels:              formatedLabels,
		SequenceID:          Issue.SequenceID,
	}
	if Issue.CompletedAt != nil {
		response.CompletedAt = *Issue.CompletedAt
	}

	models.SendSuccessResponse(c, http.StatusOK, response, "Issue updated successfully.")

}

// ListIssues godoc
func ListIssues(c *gin.Context) {
	projectID := c.Param("project_id")

	var issues []v1.Issue

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

	// Check if the project exists
	if err := tx.Where("id = ? AND deleted_at is NULL", projectID).First(&v1.Project{}).Error; err != nil {
		logger.LogError(fmt.Sprintf("failed to fetch project with ID %s", projectID), logrus.Fields{"error": err.Error(), "email": email})
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

	// Retrieve filters from query parameters
	title := c.Query("title")
	description := c.Query("description")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	isDraft := c.Query("is_draft")
	priority := c.Query("priority")
	point := c.Query("point")
	comletedPercentage := c.Query("competed_percentage")

	// Check if the user is authorized to list Issues
	authorized, role := utils.IsUserPartOfRole(tx, projectID, email)
	if !authorized || (*role != "Manager" && *role != "Owner") {
		models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		return
	}

	// Build query with filtering by role
	query := tx.Model(&v1.Issue{}).Where("project_id = ? AND deleted_at IS NULL", projectID)

	// Apply filters to the query

	if priority != "" {
		query = query.Where("priority = ?", priority)
	}

	if title != "" {
		query = query.Where("title ILIKE ?", "%"+title+"%")
	}

	if description != "" {
		query = query.Where("description ILIKE ?", "%"+description+"%")
	}

	var start, end time.Time
	layout := "2006-01-02"
	if startDate != "" {
		start, err = time.Parse(layout, startDate)
		logger.LogInfo("Start date parsed", logrus.Fields{"start_date": start})
		if err != nil {
			models.SendErrorResponse(c, http.StatusUnprocessableEntity, "Start date is not in correct format.")
			return
		}
		query = query.Where("start_date >= ?", start)
	}

	if endDate != "" {
		end, err = time.Parse(layout, endDate)
		logger.LogInfo("End date parsed", logrus.Fields{"end_date": end})
		if err != nil {
			models.SendErrorResponse(c, http.StatusUnprocessableEntity, "End date is not in correct format.")
			return
		}
		query = query.Where("end_date <= ?", end)
	}

	if point != "" {
		query = query.Where("point = ?", point)
	}

	if comletedPercentage != "" {
		query = query.Where("competed_percentage = ?", comletedPercentage)
	}

	if isDraft != "" {
		query = query.Where("is_draft = ?", isDraft)
	}

	if err := query.Debug().Select("*").
		Scopes(utils.Paginate(query, pagination)).Scan(&issues).Error; err != nil {
		tx.Rollback()
		logger.LogError("Failed to fetch Issues from the database.", logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		return
	}

	// Prepare response data
	var responses []v1.IssueWithAssignees
	for _, issue := range issues {

		// Fetch the label
		var labels []v1.ProjectLabel
		if err := tx.Where("id = ANY(?) AND deleted_at is NULL", issue.LabelIDs).Find(&labels).Error; err != nil {
			logger.LogError("Failed to fetch label from the database.", logrus.Fields{"error": err.Error(), "email": email})
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
			return
		}

		// Format labels into a list of maps
		formattedLabels := utils.FormatLabelsToMap(labels)

		// Fetch the project state by ID and associated project, ensuring the state exists for the specific project
		var state v1.ProjectState
		if err := tx.Where("id = ? AND project_id = ? AND deleted_at IS NULL", issue.StateID, issue.ProjectID).First(&state).Error; err != nil {
			logger.LogError(fmt.Sprintf("Project state with ID: %s not found for project ID: %s.", issue.StateID, issue.ProjectID), logrus.Fields{"error": err.Error(), "email": email})
			models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
			return
		}

		// Fetch the issue members
		var members []v1.IssueAssignee
		if err := tx.Where("issue_id = ?", issue.ID).Find(&members).Error; err != nil {
			logger.LogError("Failed to fetch issue members from the database.", logrus.Fields{"error": err.Error(), "email": email})
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
			return
		}

		// Fetch sub-issues if this issue is a parent
		var subIssues []v1.Issue
		if err := tx.Where("parent_id = ? AND deleted_at IS NULL", issue.ID).Find(&subIssues).Error; err != nil {
			logger.LogError("Failed to fetch sub-issues", logrus.Fields{"error": err.Error(), "email": email})
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
			return
		}

		// Create sub-issue responses
		var subIssueResponses []v1.IssueResponse
		for _, subIssue := range subIssues {
			// Fetch sub-issue state
			var subState v1.ProjectState
			if err := tx.Where("id = ? AND project_id = ? AND deleted_at IS NULL", subIssue.StateID, subIssue.ProjectID).First(&subState).Error; err != nil {
				logger.LogError("Failed to fetch sub-issue state", logrus.Fields{"error": err.Error(), "email": email})
				models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
				return
			}

			// Fetch the label
			var subLabels []v1.ProjectLabel
			if err := tx.Where("id = ANY(?) AND deleted_at is NULL", subIssue.LabelIDs).Find(&subLabels).Error; err != nil {
				logger.LogError("Failed to fetch sub-issue label from the database.", logrus.Fields{"error": err.Error(), "email": email})
				models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
				return
			}

			// Create sub-issue response
			subResponse := v1.IssueResponse{
				ID:                  utils.ConvertUUIDToString(subIssue.ID),
				ProjectID:           utils.ConvertUUIDToString(subIssue.ProjectID),
				CreatedBy:           subIssue.CreatedBy,
				UpdatedBy:           subIssue.UpdatedBy,
				CreatedAt:           subIssue.CreatedAt,
				UpdatedAt:           subIssue.UpdatedAt,
				ParentID:            utils.ConvertUUIDToString(subIssue.ParentID),
				CompletedPercentage: subIssue.CompletedPercentage,
				Title:               subIssue.Title,
				Description:         subIssue.Description,
				Priority:            subIssue.Priority,
				StartDate:           subIssue.StartDate,
				EndDate:             subIssue.EndDate,
				EstimatedHours:      subIssue.EstimatedHours,
				Point:               subIssue.Point,
				State:               v1.ProjectStateResponse(subState),
				SequenceID:          subIssue.SequenceID,
				Labels:              utils.FormatLabelsToMap(subLabels),
			}
			if subIssue.CompletedAt != nil {
				subResponse.CompletedAt = *subIssue.CompletedAt
			}
			subIssueResponses = append(subIssueResponses, subResponse)
		}

		// Prepare the response
		response := v1.IssueWithAssignees{
			IssueResponse: v1.IssueResponse{
				ID:             utils.ConvertUUIDToString(issue.ID),
				ProjectID:      utils.ConvertUUIDToString(issue.ProjectID),
				Title:          issue.Title,
				Description:    issue.Description,
				State:          v1.ProjectStateResponse(state),
				CreatedAt:      issue.CreatedAt,
				UpdatedAt:      issue.UpdatedAt,
				CreatedBy:      issue.CreatedBy,
				UpdatedBy:      issue.UpdatedBy,
				StartDate:      issue.StartDate,
				EndDate:        issue.EndDate,
				Point:          issue.Point,
				Priority:       issue.Priority,
				EstimatedHours: issue.EstimatedHours,
				ParentID:       utils.ConvertUUIDToString(issue.ParentID),
				Labels:         formattedLabels,
				SequenceID:     issue.SequenceID,
				SubIssues:      subIssueResponses, // Add sub-issues to the response
			},
			Assignees: members,
		}

		responses = append(responses, response)
	}

	// Attempt to commit the transaction
	if !utils.CommitTransaction(tx, c, email) {
		return // Early return if the commit failed
	}

	response := v1.ListIssuesResponse{
		Data: responses,
	}

	if response.Data == nil {
		response.Data = []v1.IssueWithAssignees{}
	}

	meta := models.PaginationMeta{
		Total: pagination.TotalCount,
		Page:  pagination.Page,
		Limit: pagination.PageSize,
	}

	// Send success response back to the client
	models.SendPaginatedSuccessResponse(c, response.Data, meta, "Issues retrieved successfully.")
}

// GetIssueByID godoc
func GetIssueByID(c *gin.Context) {
	var Issue v1.Issue

	id := c.Param("issue_id")
	projectID := c.Param("project_id")

	// Get email from context
	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return
	}

	// start transaction
	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return
	}

	// Check if the user is authorized to view the Issue
	authorized, role := utils.IsUserPartOfRole(tx, projectID, email)
	if !authorized || (*role != "Manager" && *role != "Owner") {
		models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		return
	}

	// Fetch the Issue
	if err := tx.Where("id = ? AND project_id = ? AND deleted_at is NULL", id, projectID).First(&Issue).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.LogError(fmt.Sprintf("failed to fetch Issue with ID %s", id), logrus.Fields{"error": err.Error(), "email": email})
			models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
			return
		}
		logger.LogError("failed to fetch Issue", logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		return
	}

	// Fetch the label
	var labels []v1.ProjectLabel
	if err := tx.Where("id = ANY(?) AND deleted_at is NULL", Issue.LabelIDs).Find(&labels).Error; err != nil {
		tx.Rollback()
		logger.LogError("Failed to fetch label from the database.", logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		return
	}

	// Format labels into a list of maps
	formattedLabels := utils.FormatLabelsToMap(labels)
	// Fetch the project state by ID and associated project, ensuring the state exists for the specific project
	var state v1.ProjectState
	if err := tx.Where("id = ? AND project_id = ? AND deleted_at IS NULL", Issue.StateID, Issue.ProjectID).First(&state).Error; err != nil {
		tx.Rollback()
		logger.LogError(fmt.Sprintf("Project state with ID: %s not found for project ID: %s.", Issue.StateID, Issue.ProjectID), logrus.Fields{"error": err.Error(), "email": email})
		// Check if the error is a record not found error
		if err == gorm.ErrRecordNotFound {
			models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		} else {
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		}
		return
	}

	// Fetch sub-issues if this issue is a parent
	var subIssues []v1.Issue
	if err := tx.Where("parent_id = ? AND deleted_at IS NULL", Issue.ID).Find(&subIssues).Error; err != nil {
		tx.Rollback()
		logger.LogError("Failed to fetch sub-issues", logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		return
	}

	// Create sub-issue responses
	var subIssueResponses []v1.IssueResponse
	for _, subIssue := range subIssues {
		// Fetch sub-issue state
		var subState v1.ProjectState
		if err := tx.Where("id = ? AND project_id = ? AND deleted_at IS NULL", subIssue.StateID, subIssue.ProjectID).First(&subState).Error; err != nil {
			tx.Rollback()
			logger.LogError("Failed to fetch sub-issue state", logrus.Fields{"error": err.Error(), "email": email})
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
			return
		}

		// Create sub-issue response
		subResponse := v1.IssueResponse{
			ID:                  utils.ConvertUUIDToString(subIssue.ID),
			Title:               subIssue.Title,
			Description:         subIssue.Description,
			Priority:            subIssue.Priority,
			StartDate:           subIssue.StartDate,
			EndDate:             subIssue.EndDate,
			CompletedPercentage: subIssue.CompletedPercentage,
			Point:               subIssue.Point,
			State:               v1.ProjectStateResponse(subState),
			SequenceID:          subIssue.SequenceID,
			EstimatedHours:      subIssue.EstimatedHours,
		}
		if subIssue.CompletedAt != nil {
			subResponse.CompletedAt = *subIssue.CompletedAt
		}
		subIssueResponses = append(subIssueResponses, subResponse)
	}

	// commit transaction
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	response := v1.IssueResponse{
		ID:                  utils.ConvertUUIDToString(Issue.ID),
		Title:               Issue.Title,
		Description:         Issue.Description,
		ProjectID:           utils.ConvertUUIDToString(Issue.ProjectID),
		CreatedBy:           email,
		Priority:            Issue.Priority,
		UpdatedBy:           email,
		Labels:              formattedLabels,
		State:               v1.ProjectStateResponse(state),
		StartDate:           Issue.StartDate,
		EndDate:             Issue.EndDate,
		EstimatedHours:      Issue.EstimatedHours,
		CompletedPercentage: Issue.CompletedPercentage,
		Point:               Issue.Point,
		ParentID:            utils.ConvertUUIDToString(Issue.ParentID),
		CreatedAt:           Issue.CreatedAt,
		UpdatedAt:           Issue.UpdatedAt,
		SequenceID:          Issue.SequenceID,
		SubIssues:           subIssueResponses,
	}
	if Issue.CompletedAt != nil {
		response.CompletedAt = *Issue.CompletedAt
	}

	models.SendSuccessResponse(c, http.StatusOK, response, "Issue fetched successfully")
}

// DeleteIssue godoc
func DeleteIssue(c *gin.Context) {
	id := c.Param("issue_id")
	projectID := c.Param("project_id")

	// Get email from context
	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return
	}

	// Start a transaction using the helper
	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return
	}

	convertID, err := utils.ConvertID(projectID, c, email, "project id")
	if err != nil {
		return
	}

	//	Check if the user is authorized to create an Issue
	authorized := utils.CanUserCreateIssue(tx, convertID, email)
	if !authorized {
		models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		return
	}

	// Check if the subIssue exists
	var Issue v1.Issue
	if err := tx.Where("id = ? AND deleted_at is NULL AND project_id = ?", id, projectID).First(&Issue).Error; err != nil {

		tx.Rollback()
		logger.LogError(fmt.Sprintf("Issue not found with ID: %s for user: %s", id, email), logrus.Fields{"error": err.Error(), "email": email})
		if err == gorm.ErrRecordNotFound {
			models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		} else {
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		}
		return
	}

	if err := tx.Model(&Issue).Update("deleted_at", time.Now()).Error; err != nil {
		tx.Rollback()
		logger.LogError(fmt.Sprintf("Failed to delete Issue with ID: %s for user: %s", id, email), logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusNotFound, errors.ErrBadRequest)
		return
	}

	// Attempt to commit the transaction
	if !utils.CommitTransaction(tx, c, email) {
		return // Early return if the commit failed
	}

	models.SendSuccessResponse(c, http.StatusNoContent, nil, "Issue deleted successfully.")
}
