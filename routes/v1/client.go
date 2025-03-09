package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/san-data-systems/common/validators"
	v1 "github.com/san-data-systems/project-management-api/controllers/v1"
)

// ClientRoute sets up the routes for task-related API endpoints.
func ClientRoute(router *gin.RouterGroup, handlers ...gin.HandlerFunc) {
	client := router.Group("", handlers...)
	{
		client.POST("/client", validators.CreateClientValidator(), v1.CreateClient)
		client.GET("/client/:id", validators.ClientIDValidator(), v1.GetClientByID)
		client.PUT("/client/:id", validators.ClientIDValidator(), validators.UpdateClientValidator(), v1.UpdateClientByID)
		client.DELETE("/client/:id", validators.ClientIDValidator(), v1.DeleteClientByID)
		client.GET("/clients", v1.ListClients)

	}
}
