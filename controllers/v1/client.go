// Package v1 provides API endpoints and handlers for version 1 of the application.
package v1

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"github.com/san-data-systems/common/errors"
	"github.com/san-data-systems/common/logger"
	"github.com/san-data-systems/common/models"
	v1 "github.com/san-data-systems/common/models/v1"
	"github.com/san-data-systems/common/utils"
	"gorm.io/gorm"
	"net/http"
	"strings"
	"time"
)

// CreateClient creates a new client based on the request payload.
// It validates the request, creates the client in the database, and commits the transaction.
// Responds with the created client's details.
func CreateClient(c *gin.Context) {
	var req v1.ClientRequest

	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return
	}

	if !utils.BindJSONRequest(c, &req, email) {
		return
	}

	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return
	}

	client := v1.Client{
		Name:          req.Name,
		ManagerEmails: req.ManagerEmails,
		Country:       req.Country,
		CreatedBy:     email,
	}

	if !utils.CreateWithRollback(tx, c, &client, "Failed to create Client.", email) {
		return
	}

	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	response := v1.ClientResponse{
		ID:            client.ID.String(),
		Name:          client.Name,
		ManagerEmails: client.ManagerEmails,
		Country:       client.Country,
		CreatedAt:     client.CreatedAt,
		UpdatedAt:     client.UpdatedAt,
		CreatedBy:     client.CreatedBy,
	}

	models.SendSuccessResponse(c, http.StatusCreated, response, "Client created successfully.")
}

// GetClientByID retrieves a client by its ID.
// Ensures the client is not archived and is accessible to the requesting user.
// Responds with the client's details.
func GetClientByID(c *gin.Context) {
	id := c.Param("id")

	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return
	}

	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return
	}

	var client v1.Client

	if err := tx.Where("id = ? AND deleted_at IS NULL", id).First(&client).Error; err != nil {
		tx.Rollback()
		logger.LogError(fmt.Sprintf("Client with ID: %s not found.", id), map[string]interface{}{"error": err.Error(), "email": email})
		if err == gorm.ErrRecordNotFound {
			models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		} else {
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		}
		return
	}

	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	response := v1.ClientResponse{
		ID:            client.ID.String(),
		Name:          client.Name,
		ManagerEmails: client.ManagerEmails,
		Country:       client.Country,
		CreatedAt:     client.CreatedAt,
		UpdatedAt:     client.UpdatedAt,
		CreatedBy:     client.CreatedBy,
	}
	models.SendSuccessResponse(c, http.StatusOK, response, "Client retrieved successfully.")
}

// UpdateClientByID updates an existing client by its ID.
// Validates the request, ensures the client exists and is accessible, then updates the client details.
// Responds with the updated client's details.
func UpdateClientByID(c *gin.Context) {
	var req v1.ClientRequest
	id := c.Param("id")

	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return
	}

	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return
	}

	var client v1.Client

	if err := tx.Where("id = ? AND created_by = ? AND deleted_at IS NULL", id, email).First(&client).Error; err != nil {
		tx.Rollback()
		logger.LogError(fmt.Sprintf("Client with ID: %s not found.", id), map[string]interface{}{"error": err.Error(), "email": email})
		if err == gorm.ErrRecordNotFound {
			models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		} else {
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		}
		return
	}

	if !utils.BindJSONRequest(c, &req, email) {
		return
	}

	if req.Name != "" {
		client.Name = req.Name
	}
	if req.Country != "" {
		client.Country = req.Country
	}
	if len(req.ManagerEmails) > 0 {
		client.ManagerEmails = req.ManagerEmails
	}

	if err := tx.Save(&client).Error; err != nil {
		tx.Rollback()
		logger.LogError(fmt.Sprintf("Error in updating client with ID: %s for user: %s", id, email), map[string]interface{}{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusInternalServerError, "Failed to update client.")
		return
	}

	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	response := v1.ClientResponse{
		ID:            client.ID.String(),
		Name:          client.Name,
		ManagerEmails: client.ManagerEmails,
		Country:       client.Country,
		CreatedAt:     client.CreatedAt,
		UpdatedAt:     client.UpdatedAt,
		CreatedBy:     client.CreatedBy,
	}

	models.SendSuccessResponse(c, http.StatusOK, response, "Client updated successfully.")
}

// DeleteClientByID archives a client by setting its archive status to true.
// Ensures the client is accessible to the requesting user before updating.
// Responds with a success message upon completion.
func DeleteClientByID(c *gin.Context) {
	id := c.Param("id")

	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return
	}

	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return
	}

	// Soft delete the project state
	now := time.Now()

	if err := tx.Model(&v1.Client{}).Where("id = ? AND created_by = ? AND deleted_at IS NULL", id, email).Update("deleted_at", now).Error; err != nil {
		tx.Rollback()
		logger.LogError(fmt.Sprintf("Failed to archive client with ID: %s.", id), map[string]interface{}{"error": err.Error(), "email": email})
		if err == gorm.ErrRecordNotFound {
			models.SendErrorResponse(c, http.StatusNotFound, errors.ErrRecordNotFound)
		} else {
			models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		}
		return
	}

	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	models.SendSuccessResponse(c, http.StatusNoContent, nil, "Client deleted successfully.")
}

// ListClients retrieves all clients based on pagination and optional filters.
// Ensures the data is accessible to the requesting user.
// Responds with a paginated list of clients.
func ListClients(c *gin.Context) {
	var clients []v1.Client

	email, valid := utils.GetEmailFromContext(c)
	if !valid {
		return
	}

	pagination, err := utils.ParsePagination(c)
	if err != nil {
		logger.LogError("Invalid pagination parameters.", map[string]interface{}{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusBadRequest, errors.ErrBadRequest)
		return
	}

	// Get query parameters for filtering
	managerEmails := c.Query("manager_emails")
	name := c.Query("name")
	country := c.Query("country")

	tx, ok := utils.StartTransaction(c, email)
	if !ok {
		return
	}

	// Start building the query
	query := tx.Model(&v1.Client{}).Where("deleted_at IS NULL")

	// If manager_emails is provided, filter by it
	if managerEmails != "" {
		// Split the managerEmails string into a slice of strings
		managerEmailList := strings.Split(managerEmails, ",")
		// Ensure the query uses the correct PostgreSQL array filtering
		query = query.Where("manager_emails @> ?", pq.StringArray(managerEmailList))
	}

	// Other filters
	if name != "" {
		query = query.Where("name ILIKE ?", "%"+name+"%")
	}
	if country != "" {
		query = query.Where("country ILIKE ?", "%"+country+"%")
	}

	if err := query.Scopes(utils.Paginate(query, pagination)).Scan(&clients).Error; err != nil {
		tx.Rollback()
		logger.LogError("Failed to fetch clients from the database.", map[string]interface{}{"error": err.Error(), "email": email})
		models.SendErrorResponse(c, http.StatusInternalServerError, errors.ErrInternalServer)
		return
	}

	// Commit transaction
	if !utils.CommitTransaction(tx, c, email) {
		return
	}

	// Convert clients to responses
	var responses []v1.ClientResponse
	for _, client := range clients {
		responses = append(responses, v1.ClientResponse{
			ID:            client.ID.String(),
			Name:          client.Name,
			CreatedBy:     client.CreatedBy,
			CreatedAt:     client.CreatedAt,
			UpdatedAt:     client.UpdatedAt,
			ManagerEmails: client.ManagerEmails,
			Country:       client.Country,
		})
	}

	// Prepare paginated response
	response := v1.ListClientResponse{
		Data: responses,
	}

	// If no clients found, return an empty list
	if response.Data == nil {
		response.Data = []v1.ClientResponse{}
	}

	meta := models.PaginationMeta{
		Total: pagination.TotalCount,
		Page:  pagination.Page,
		Limit: pagination.PageSize,
	}

	// Send successful response
	models.SendPaginatedSuccessResponse(c, response.Data, meta, "Clients retrieved successfully.")
}
