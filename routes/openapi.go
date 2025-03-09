package routes

import (
	"fmt"
	"github.com/gin-gonic/gin"
	scalar "github.com/san-data-systems/common/scaler"
	"net/http"
)

// OpenAPISpec registers the /openapi.json endpoint for API documentation
func OpenAPISpec(router *gin.RouterGroup) {
	router.GET("/openapi.json", func(c *gin.Context) {
		htmlContent, err := scalar.ApiReferenceHTML(&scalar.Options{
			SpecURL: "./docs/openapi.json",
			CustomOptions: scalar.CustomOptions{
				PageTitle: "Project Management API",
			},
			DarkMode: true,
		})

		if err != nil {
			c.String(http.StatusInternalServerError, fmt.Sprintf("Error generating API reference: %v", err))
			return
		}

		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(htmlContent))
	})
}
