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

// CreateIssueTimeEntry creates a new IssueTimeEntry entry.
func CreateIssueTimeEntry(c *gin.Context) {
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

	// Start a database transaction
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

	var request v1.TimeEntryRequest
	// Bind the request to the Issuete struct
	if !utils.BindJSONRequest(c, &request, email) {
		return
	}

	// Check if the Issue exists
	var issue v1.Issue
	if err := tx.Debug().Where("id = ? AND deleted_at is NULL AND project_id = ?", issueID, projectID).First(&issue).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.LogError(fmt.Sprintf("failed to fetch Issue with ID %s", issueID), logrus.Fields{"error": err.Error(), "email": email})
			models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
			return
		}
		logger.LogError("failed to fetch Issue", logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		return
	}

	// Định nghĩa layout cho chuỗi thời gian
	timeLayout := "15:04:05"   // Layout cho thời gian (giờ:phút:giây)
	dateLayout := "2006-01-02" // Layout cho ngày (năm-tháng-ngày)

	// Parse chuỗi thời gian thành kiểu time.Time
	startTime, err := time.Parse(timeLayout, request.StartTime)
	if err != nil {
		logger.LogError("Failed to parse start time.", logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusBadRequest, errors.ErrBadRequest)
		return
	}

	endTime, err := time.Parse(timeLayout, request.EndTime)
	if err != nil {
		logger.LogError("Failed to parse end time.", logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusBadRequest, errors.ErrBadRequest)
		return
	}

	parsedDate, err := time.Parse(dateLayout, request.Date)
	if err != nil {
		logger.LogError("Failed to parse date.", logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusBadRequest, errors.ErrBadRequest)
		return
	}

	// Combine date with start_time and end_time to create timestamps
	startDateTime := time.Date(
		parsedDate.Year(), parsedDate.Month(), parsedDate.Day(),
		startTime.Hour(), startTime.Minute(), startTime.Second(), 0,
		parsedDate.Location(), // Use the same time zone as the parsed date
	)

	endDateTime := time.Date(
		parsedDate.Year(), parsedDate.Month(), parsedDate.Day(),
		endTime.Hour(), endTime.Minute(), endTime.Second(), 0,
		parsedDate.Location(), // Use the same time zone as the parsed date
	)

	// Kiểm tra xem time entry có nằm trong khoảng thời gian của issue không
	if parsedDate.Before(issue.StartDate) || parsedDate.After(issue.EndDate) {
		tx.Rollback()
		logger.LogError("Time entry is not within the issue's date range.", logrus.Fields{
			"email":      email,
			"issue_id":   issueID,
			"start_date": issue.StartDate,
			"end_date":   issue.EndDate,
			"entry_date": parsedDate,
		})
		models.SendErrorResponse(c, http.StatusBadRequest, errors.ErrBadRequest)
		return
	}

	// Create a new IssueTimeEntry entry
	issueTimeEntry := v1.TimeEntry{
		ProjectID: parsedProjectID,
		IssueID:   parsedIssueID,
		CreatedBy: email,
		Date:      parsedDate,
		StartTime: startDateTime,
		EndTime:   endDateTime,
		Notes:     request.Notes,
	}

	// Create the Issuete entry
	if !utils.CreateWithRollback(tx, c, &issueTimeEntry, "Failed to create time entry", email) {
		return
	}

	// Commit the transaction
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	res := v1.TimeEntryResponse{
		ID:                  issueTimeEntry.ID.String(),
		ProjectID:           issueTimeEntry.ProjectID.String(),
		IssueID:             issueTimeEntry.IssueID.String(),
		CreatedBy:           issueTimeEntry.CreatedBy,
		Date:                parsedDate,
		StartTime:           startDateTime,
		EndTime:             endDateTime,
		Hours:               issueTimeEntry.Hours,
		Notes:               issueTimeEntry.Notes,
		CreatedAt:           issueTimeEntry.CreatedAt,
		IsTimeCardGenerated: issueTimeEntry.IsTimeCardGenerated,
	}

	// Send the response
	models.SendSuccessResponse(c, http.StatusCreated, res, "Time entry created successfully")

}

// ListIssueTimeEntries lists all IssueTimeEntry entries.
func ListIssueTimeEntries(c *gin.Context) {
	// Retrieve email from context
	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return // The response is already sent by the helper, so just return
	}

	issueID := c.Param("issue_id")
	projectID := c.Param("project_id")

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

	// Retrieve filters from query parameters
	date := c.Query("date")
	startTime := c.Query("start_time")
	endTime := c.Query("end_time")

	query := tx.Model(&v1.TimeEntry{}).
		Where("issue_id = ? AND project_id = ?", issueID, projectID)

	var dateRes time.Time

	if date != "" {
		// Parse the date string into a time.Time value
		parsedDate, err := time.Parse("2006-01-02", date)
		if err != nil {
			logger.LogError("Failed to parse date.", logrus.Fields{"error": err.Error(), "email": email})
			models.SendErrorResponse(c, http.StatusBadRequest, errors.ErrBadRequest)
			return
		}

		// Use the parsed date in the query
		query = query.Where("date = ?", parsedDate)
		dateRes = parsedDate
	}

	// Filter by start_time and end_time
	if startTime != "" && endTime != "" {

		// Parse the start_time and end_time strings into time.Time values
		parsedStartTime, err := time.Parse("15:04:05", startTime)
		if err != nil {
			logger.LogError("Failed to parse start_time.", logrus.Fields{"error": err.Error(), "email": email})
			models.SendErrorResponse(c, http.StatusBadRequest, errors.ErrBadRequest)
			return
		}

		parsedEndTime, err := time.Parse("15:04:05", endTime)
		if err != nil {
			logger.LogError("Failed to parse end_time.", logrus.Fields{"error": err.Error(), "email": email})
			models.SendErrorResponse(c, http.StatusBadRequest, errors.ErrBadRequest)
			return
		}

		// Combine date with start_time and end_time to create timestamps
		startDateTime := time.Date(
			dateRes.Year(), dateRes.Month(), dateRes.Day(),
			parsedStartTime.Hour(), parsedStartTime.Minute(), parsedStartTime.Second(), 0,
			dateRes.Location(), // Use the same time zone as the parsed date
		)

		endDateTime := time.Date(
			dateRes.Year(), dateRes.Month(), dateRes.Day(),
			parsedEndTime.Hour(), parsedEndTime.Minute(), parsedEndTime.Second(), 0,
			dateRes.Location(), // Use the same time zone as the parsed date
		)

		// Use the timestamps in the query
		query = query.Where("start_time >= ? AND end_time <= ?", startDateTime, endDateTime)
	}

	var timeEntries []v1.TimeEntry
	if err := query.Select("*").
		Scopes(utils.Paginate(query, pagination)).Scan(&timeEntries).Error; err != nil {
		tx.Rollback()
		logger.LogError("Failed to list time entries", nil)
		models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		return
	}

	// Group time entries by email
	timeEntriesByEmail := make(map[string][]v1.TimeEntryResponse)
	for _, te := range timeEntries {
		response := v1.TimeEntryResponse{
			ID:                  te.ID.String(),
			ProjectID:           te.ProjectID.String(),
			IssueID:             te.IssueID.String(),
			CreatedBy:           te.CreatedBy,
			Date:                te.Date,
			StartTime:           te.StartTime,
			EndTime:             te.EndTime,
			Hours:               te.Hours,
			Notes:               te.Notes,
			CreatedAt:           te.CreatedAt,
			IsTimeCardGenerated: te.IsTimeCardGenerated,
		}

		// Append the time entry to the slice for the corresponding email
		timeEntriesByEmail[te.CreatedBy] = append(timeEntriesByEmail[te.CreatedBy], response)
	}

	meta := models.PaginationMeta{
		Total: pagination.TotalCount,
		Page:  pagination.Page,
		Limit: pagination.PageSize,
	}

	// Send the response
	models.SendPaginatedSuccessResponse(c, timeEntriesByEmail, meta, "Time entries retrieved successfully.")

}

// GetIssueTimeEntryByID retrieves a specific IssueTimeEntry entry by ID.
func GetIssueTimeEntryByID(c *gin.Context) {
	// Retrieve email from context
	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return // The response is already sent by the helper, so just return
	}

	teID := c.Param("te_id")
	issueID := c.Param("issue_id")
	projectID := c.Param("project_id")

	// Convert issue ID and handle any conversion errors
	parsedteID, _ := utils.ConvertID(teID, c, email, "time entry id")
	parsedIssueID, _ := utils.ConvertID(issueID, c, email, "issue id")
	parsedProjectID, _ := utils.ConvertID(projectID, c, email, "project id")

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

	var te v1.TimeEntry
	if err := tx.Where("id = ? AND issue_id = ? AND project_id = ?", parsedteID, parsedIssueID, parsedProjectID).First(&te).Error; err != nil {
		tx.Rollback()
		logger.LogError("Failed to retrieve time entry", nil)
		models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		return
	}

	// Commit the transaction
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	response := v1.TimeEntryResponse{
		ID:                  te.ID.String(),
		ProjectID:           te.ProjectID.String(),
		IssueID:             te.IssueID.String(),
		CreatedBy:           te.CreatedBy,
		Date:                te.Date,
		StartTime:           te.StartTime,
		EndTime:             te.EndTime,
		Hours:               te.Hours,
		Notes:               te.Notes,
		CreatedAt:           te.CreatedAt,
		IsTimeCardGenerated: te.IsTimeCardGenerated,
	}

	// Send the response
	models.SendSuccessResponse(c, http.StatusOK, response, "Time entry retrieved successfully")
}

// UpdateIssueTimeEntryByID updates a specific IssueTimeEntry entry by ID.
func UpdateIssueTimeEntryByID(c *gin.Context) {
	// Retrieve email from context
	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return // The response is already sent by the helper, so just return
	}

	var request v1.TimeEntryRequest
	// Bind the request to the Issuete struct
	if !utils.BindJSONRequest(c, &request, email) {
		return
	}

	teID := c.Param("te_id")
	issueID := c.Param("issue_id")
	projectID := c.Param("project_id")

	// Convert issue ID and handle any conversion errors
	parsedteID, _ := utils.ConvertID(teID, c, email, "time entry id")
	parsedIssueID, _ := utils.ConvertID(issueID, c, email, "issue id")
	parsedProjectID, _ := utils.ConvertID(projectID, c, email, "project id")

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

	var issue v1.Issue
	if err := tx.Debug().Where("id = ? AND deleted_at is NULL AND project_id = ?", issueID, projectID).First(&issue).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.LogError(fmt.Sprintf("failed to fetch Issue with ID %s", issueID), logrus.Fields{"error": err.Error(), "email": email})
			models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
			return
		}
		logger.LogError("failed to fetch Issue", logrus.Fields{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		return
	}

	var te v1.TimeEntry
	if err := tx.Where("id = ? AND issue_id = ? AND project_id = ?", parsedteID, parsedIssueID, parsedProjectID).First(&te).Error; err != nil {
		tx.Rollback()
		logger.LogError("Failed to retrieve time entry", nil)
		models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		return
	}

	if request.Date != "" {
		// Định nghĩa layout cho chuỗi thời gian
		dateLayout := "2006-01-02" // Layout cho ngày (năm-tháng-ngày)

		// Parse chuỗi thời gian thành kiểu time.Time
		date, err := time.Parse(dateLayout, request.Date)
		if err != nil {
			logger.LogError("Failed to parse date.", logrus.Fields{"error": err.Error(), "email": email})
			models.SendErrorResponse(c, http.StatusBadRequest, errors.ErrBadRequest)
			return
		}

		// Kiểm tra xem time entry có nằm trong khoảng thời gian của issue không
		if date.Before(issue.StartDate) || date.After(issue.EndDate) {
			tx.Rollback()
			logger.LogError("Time entry is not within the issue's date range.", logrus.Fields{
				"email":      email,
				"issue_id":   issueID,
				"entry_date": date,
			})
			models.SendErrorResponse(c, http.StatusBadRequest, errors.ErrBadRequest)
			return
		}

		te.Date = date
	}
	timeLayout := "15:04:05"

	if request.StartTime != "" {

		startTime, err := time.Parse(timeLayout, request.StartTime)
		if err != nil {
			logger.LogError("Failed to parse start time.", logrus.Fields{"error": err.Error(), "email": email})
			models.SendErrorResponse(c, http.StatusBadRequest, errors.ErrBadRequest)
			return
		}

		te.StartTime = startTime
	}

	if request.EndTime != "" {

		endTime, err := time.Parse(timeLayout, request.EndTime)
		if err != nil {
			logger.LogError("Failed to parse end time.", logrus.Fields{"error": err.Error(), "email": email})
			models.SendErrorResponse(c, http.StatusBadRequest, errors.ErrBadRequest)
			return
		}

		te.EndTime = endTime
	}

	if request.StartTime != "" || request.EndTime != "" {
		te.Hours = te.EndTime.Sub(te.StartTime).Hours()
	}

	// Save the updated time entry
	if err := tx.Save(&te).Error; err != nil {
		tx.Rollback()
		logger.LogError("Failed to update time entry", nil)
		models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		return
	}

	// Commit the transaction
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	response := v1.TimeEntryResponse{
		ID:                  te.ID.String(),
		ProjectID:           te.ProjectID.String(),
		IssueID:             te.IssueID.String(),
		CreatedBy:           te.CreatedBy,
		Date:                te.Date,
		StartTime:           te.StartTime,
		EndTime:             te.EndTime,
		Hours:               te.Hours,
		Notes:               te.Notes,
		CreatedAt:           te.CreatedAt,
		IsTimeCardGenerated: te.IsTimeCardGenerated,
	}

	// Send the response
	models.SendSuccessResponse(c, http.StatusOK, response, "Time entry updated successfully")
}

// DeleteIssueTimeEntry deletes a specific IssueTimeEntry entry by ID.
func DeleteIssueTimeEntry(c *gin.Context) {
	// Retrieve email from context
	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return // The response is already sent by the helper, so just return
	}

	teID := c.Param("te_id")
	issueID := c.Param("issue_id")
	projectID := c.Param("project_id")

	// Convert issue ID and handle any conversion errors
	parsedteID, _ := utils.ConvertID(teID, c, email, "time entry id")
	parsedIssueID, _ := utils.ConvertID(issueID, c, email, "issue id")
	parsedProjectID, _ := utils.ConvertID(projectID, c, email, "project id")

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

	var te v1.TimeEntry
	if err := tx.Where("id = ? AND issue_id = ? AND project_id = ?", parsedteID, parsedIssueID, parsedProjectID).First(&te).Error; err != nil {
		tx.Rollback()
		logger.LogError("Failed to retrieve time entry", nil)
		models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		return
	}

	// Soft delete the time entry
	if err := tx.Delete(&te).Error; err != nil {
		tx.Rollback()
		logger.LogError("Failed to delete time entry", nil)
		models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		return
	}

	// Commit the transaction
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	// Send the response
	models.SendSuccessResponse(c, http.StatusOK, nil, "Time entry deleted successfully")
}
