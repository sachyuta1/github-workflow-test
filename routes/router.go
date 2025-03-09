// Package routes initializes the Gin router and sets up the API routes for the workflow service.
// It includes middleware for logging, recovery, CORS, and request ID handling, as well as Swagger documentation.
package routes

import (
	"net/http"

	v1 "github.com/san-data-systems/project-management-api/routes/v1"

	"github.com/gin-gonic/gin"
	"github.com/san-data-systems/common/config"
	"github.com/san-data-systems/common/middlewares"
)

// service defines the service name.
var service = "Project Management"

// New initializes a new Gin engine and sets up the routes.
func New() *gin.Engine {
	// Create a new Gin engine instance
	r := gin.New()

	// Initialize routes
	initRoute(r)

	r.Use(gin.CustomRecovery(middlewares.AppRecovery()))
	r.Use(middlewares.CORSMiddleware())
	r.Use(middlewares.RequestIDMiddleware())
	r.Use(middlewares.LoggerMiddleware()) // Log requests to both file and stdout

	docs := r.Group("/docs")
	{
		OpenAPISpec(docs)
	}
	apiV1 := r.Group("/api/v1")
	{
		HealthzRoute(apiV1)
		VersionRoute(apiV1)

		v1.ProjectRoute(apiV1, middlewares.JWTMiddleware())
		v1.ClientRoute(apiV1, middlewares.JWTMiddleware())
		v1.ProjectLabelRoute(apiV1, middlewares.JWTMiddleware())
		v1.ProjectStateRoute(apiV1, middlewares.JWTMiddleware())
		v1.ProjectSlugRoute(apiV1, middlewares.JWTMiddleware())
		v1.ProjectMember(apiV1, middlewares.JWTMiddleware())
		v1.ProjectFileRoute(apiV1, middlewares.JWTMiddleware())
		v1.IssueRoute(apiV1, middlewares.JWTMiddleware())
		v1.IssueLinkRoute(apiV1, middlewares.JWTMiddleware())
		v1.IssueAssigneeRoute(apiV1, middlewares.JWTMiddleware())
		v1.IssueFileRoute(apiV1, middlewares.JWTMiddleware())
		v1.IssueTimeEntryRoute(apiV1, middlewares.JWTMiddleware())
	}
	return r
}

// initRoute sets up the routes for the Gin engine.
func initRoute(r *gin.Engine) {
	_ = r.SetTrustedProxies(nil)
	r.RedirectTrailingSlash = false
	r.HandleMethodNotAllowed = true

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": c.Request.RequestURI + " not found."})
	})

	r.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": c.Request.Method + " method is not allowed."})
	})
}

// InitGin initializes the Gin engine with the appropriate mode.
func InitGin() {
	gin.DisableConsoleColor()
	gin.SetMode(config.Config.Mode)
	// Additional initialization tasks can be added here.
}
