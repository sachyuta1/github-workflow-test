# Project Management API

[![Go Lint](https://github.com/san-data-systems/project-management-api/actions/workflows/go.yml/badge.svg)](https://github.com/san-data-systems/project-management-api/actions/workflows/go.yml)

The `project-management-api` project provides a RESTful API for managing tasks and their associated comments. Built using the [Gin Web Framework](https://github.com/gin-gonic/gin), this project follows best practices for API design and implementation.

## Table of Contents
- [Description](#description)
- [Features](#features)
- [Architecture](#architecture)
- [Folder Structure](#folder-structure)
- [Installation](#installation)
- [Configuration](#configuration)
- [Running the Project](#running-the-project)
- [Usage](#usage)
   - [Task Endpoints](#task-endpoints)
   - [Task Comment Endpoints](#task-comment-endpoints)
   - [SubTask Endpoints](#subtask-endpoints)
   - [SubTask Comment Endpoints](#subtask-comment-endpoints)
- [Validators](#validators)
- [Testing](#testing)
- [Dependencies](#dependencies)
- [Contributing](#contributing)
- [Contact](#contact)

## Description

The `project-management-api` project allows users to create, retrieve, update, and delete tasks, as well as manage comments associated with those tasks. The API is versioned, and this documentation covers version 1 (`v1`).

## Features

- **Project Management:**
   - Create a new task
   - Retrieve task details
   - Update an existing task
   - Delete a task
   - List all tasks

- **Project Comment Management:**
   - Create a new comment for a specific task
   - Retrieve a specific comment by its ID
   - Update a comment by its ID
   - Delete a comment by its ID
   - List all comments for a specific task

## Architecture

The project follows a modular architecture with the separation of concerns:

1. **[Controllers](controllers/v1):** Handle the HTTP requests and responses.
2. **[Routes](routes/v1):** Define the API endpoints and link them to the appropriate controllers.
3. **[Validators](https://github.com/san-data-systems/common/tree/main/validators):** Ensure that incoming requests meet the required criteria.
4. **[Configuration](https://github.com/san-data-systems/common/tree/main/config):** Manage application settings and environment variables.
5. **[Database](https://github.com/san-data-systems/common/tree/main/databases):** Manage connections and interactions with PostgreSQL and Redis.

## Folder Structure

The following is an overview of the project's folder structure:

```plaintext
project-management-api/
├── Dockerfile        # Dockerfile defining the environment for running the application
├── README.md         # Project documentation providing an overview of the application
├── controllers
│   └── v1
│       ├── subtask.go            # Controller for handling subtask-related requests (CRUD operations)
│       ├── subtask_comment.go    # Controller for managing comments on subtasks
│       ├── task.go               # Controller for handling task-related requests (CRUD operations)
│       └── task_comment.go       # Controller for managing comments on tasks
├── deploy
│   ├── docker-compose
│   │   ├── docker-compose.yml # Docker Compose configuration for multi-container setup
│   │   └── nats-server.conf   # NATS server configuration for message brokering
│   └── k8s
│       # Kubernetes deployment files for orchestrating containers in a cluster
├── docs
│   ├── docs.go                # Auto-generated Go documentation
│   ├── swagger.json           # Swagger API specification in JSON format
│   └── swagger.yaml           # Swagger API specification in YAML format
├── go.mod             # Go module dependencies
├── go.sum             # Checksums of the Go module dependencies
├── logs
│   └── gin.log         # Log file for Gin framework activities
├── main.go            # Entry point of the application, initializing routes and starting the server
├── routes
│   ├── healthz.go              # Health check route to monitor API health
│   ├── router.go               # API router setup for managing route definitions
│   ├── v1
│   │   ├── subtask.go          # Route definitions for subtask-related endpoints
│   │   ├── subtask_comment.go  # Route definitions for subtask comment endpoints
│   │   ├── task.go             # Route definitions for task-related endpoints
│   │   └── task_comment.go     # Route definitions for task comment endpoints
│   └── version.go              # API versioning route
└── scripts
    └── fix_go_issues.sh        # Script for automatically fixing Go lint and formatting issues

```

## Installation

To set up the project locally, follow these steps:

1. Clone the repository:
   ```sh
   git clone https://github.com/san-data-systems/project-management-api.git
   cd project-management-api
   ```

2. Install dependencies:
   ```sh
   go env -w GOPRIVATE=github.com/san-data-systems/common
   go get github.com/san-data-systems/common
   go mod tidy
   ```

## Configuration

Configuration variables are managed using a `config` package. You can set your own configuration parameters in a `.env` file.

Example `.env`:

```plaintext
# SERVER
SERVER_HOST=localhost
SERVER_PORT=9194

# DATABASE
DATABASE_HOST=localhost
DATABASE_PORT=27017
DATABASE_NAME=1View
DATABASE_USER=1View
DATABASE_PASSWORD=1View

# REDIS
USE_REDIS=false
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=6379

#NATS
NATS_HOST=localhost
NATS_PORT=4222
NATS_USERNAME=nats_user
NATS_PASSWORD=nats_password


#JWT
JWT_SECRET="Sample"
# debug or release
MODE=debug
```

## Running the Project

To run the project:

```sh
go run main.go
```

Access the Swagger Document on http://localhost:9194/swagger/index.html.

This will start the server on the specified host and port defined in the configuration.

## Usage

### Task Endpoints

- **Create a new task:**
  ```http
  POST /v1/task
  ```
  **Payload:**
  ```json
  {
    "title": "My First Task",
    "description": "This is the description of my first task.",
    "viewers": ["viewer1@example.com", "viewer2@example.com"],
    "tags": ["go", "api", "tutorial"]
  }
  ```

- **Retrieve details of a specific task by its ID:**
  ```http
  GET /v1/task/:id
  ```

- **Update an existing task by its ID:**
  ```http
  PUT /v1/task/:id
  ```
  **Payload:**
  ```json
  {
    "title": "Updated Task Title",
    "description": "This is the updated description of the task.",
    "viewers": ["viewer3@example.com"],
    "tags": ["update", "task"]
  }
  ```

- **Delete a specific task by its ID:**
  ```http
  DELETE /v1/task/:id
  ```

- **List all tasks:**
  ```http
  GET /v1/tasks
  ```

### SubTask Endpoints

- **Create a new subtask for a task:**
  ```http
  POST /v1/task/:id/subtask
  ```
  **Payload:**
  ```json
  {
    "title": "My First SubTask",
    "description": "This is the description of my first subtask.",
    "viewers": ["viewer1@example.com", "viewer2@example.com"],
    "tags": ["go", "api", "tutorial"]
  }
  ```

- **Retrieve details of a specific subtask by its ID:**
  ```http
  GET /v1/task/:id/subtask/:sid
  ```

- **Update an existing subtask by its ID:**
  ```http
  PUT /v1/task/:id/subtask/:sid
  ```
  **Payload:**
  ```json
  {
    "title": "Updated SubTask Title",
    "description": "This is the updated description of the subtask.",
    "viewers": ["viewer3@example.com"],
    "tags": ["update", "subtask"]
  }
  ```

- **Delete a specific subtask by its ID:**
  ```http
  DELETE /v1/task/:id/subtask/:sid
  ```

- **List all subtasks for a task:**
  ```http
  GET /v1/task/:id/subtasks
  ```

### Task Comment Endpoints

- **Create a new comment for a specific task:**
  ```http
  POST /v1/tasks/:id/comments
  ```
  **Payload:**
  ```json
  {
    "comment": "This is a comment."
  }
  ```

- **Retrieve a specific comment by its ID associated with a task:**
  ```http
  GET /v1/tasks/:id/comments/:cid
  ```

- **Update a specific comment by its ID associated with a task:**
  ```http
  PUT /v1/tasks/:id/comments/:cid
  ```
  **Payload:**
  ```json
  {
    "comment": "This is the updated description of the comment."
  }
  ```

- **Delete a specific comment by its ID associated with a task:**
  ```http
  DELETE /v1/tasks/:id/comments/:cid
  ```

- **List all comments associated with a specific task by its ID:**
  ```http
  GET /v1/tasks/:id/comments
  ```

### SubTask Comment Endpoints

- **Create a new comment for a specific subtask:**
  ```http
  POST /v1/task/:id/subtask/:sid/comments
  ```
  **Payload:**
  ```json
  {
    "comment": "This is a comment."
  }
  ```

- **Retrieve a specific comment by its ID associated with a subtask:**
  ```http
  GET /v1/task/:id/subtask/:sid/comments/:cid
  ```

- **Update a specific comment by its ID associated with a subtask:**
  ```http
  PUT /v1/task/:id/subtask/:sid/comments/:cid
  ```
  **Payload:**
  ```json
  {
    "comment": "This is the updated description of the comment."
  }
  ```

- **Delete a specific comment by its ID associated with a subtask:**
  ```http
  DELETE /v1/task/:id/subtask/:sid/comments/:cid
  ```

- **List all comments associated with a specific subtask by its ID:**
  ```http
  GET /v1/task/:id/subtask/:sid/comments
  ```

## Validators

Validators are used to ensure incoming requests are correctly formatted and contain valid data:

- `CreateTaskValidator`
- `CreateSubTaskValidator`
- `TaskIDValidator`
- `SubTaskIDValidator`
- `UpdateTaskValidator`
- `UpdateSubTaskValidator`
- `CreateTaskCommentValidator`
- `CreateSubTaskCommentValidator`
- `TaskCommentIDValidator`
- `SubTaskCommentIDValidator`

## Testing

To run tests for the project:

```sh
go test ./...
```

Ensure that your test configurations are correctly set to avoid conflicts with your development environment.

## Dependencies

The project relies on several external libraries and packages:

- [Gin](https://github.com/gin-gonic/gin) - Web framework
- [GORM](https://gorm.io/) - ORM for interacting with the database
- [Viper](https://github.com/spf13/viper) - Configuration management
- [GoDotEnv](https://github.com/joho/godotenv) - Loading environment variables from `.env` file
- [Swagger](https://github.com/swaggo/swag) - API documentation
- [Common](https://github.com/san-data-systems/common) - common package

Make sure to check the `go.mod` file for the exact versions of these dependencies.

## Contributing

Contributions are welcome! To contribute:

1. Fork the repository.
2. Create a new branch.
3. Make your changes.
4. Submit a pull request.


## Contact

For questions or concerns, please reach out to us at [support@san-data-systems.com](mailto:support@redcloudcomputing.com).

## Implementation Details

The main entry point for the `1View` portal API server is defined in the `main.go` file.

Here is an overview:

```go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/san-data-systems/project-management-api/routes"
	"github.com/san-data-systems/common/config"
	"github.com/san-data-systems/common/databases"
)

// @title 1View Task Portal
// @version 1.0
// @description 1View Task Portal Rest API documentation
// @termsOfService https://swagger.io/terms/

// @contact.name Vivek Singh
// @contact.url https://github.com/san-data-systems
// @contact.email vbhadauriya@redcloudcomputing.com

// @license.name MIT License
// @license.url https://opensource.org/licenses/MIT

// @host localhost:9194
// @BasePath /
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
// @description JWT Authorization header using the Bearer scheme.
// @schemes http https

// main is the entry point for the 1View portal API server.
func main() {
	// Load application configuration
	config.LoadConfig()
	// Initialize PostgreSQL database
	databases.InitPostgresDB()

	// Optionally, initialize Redis if enabled in the config
	if config.Config.UseRedis {
		databases.CheckRedisConnection()
	}

	// Initialize the Gin router with defined routes
	routes.InitGin()
	router := routes.New()

	// Define HTTP server with proper timeouts
	server := &http.Server{
		Addr:         config.Config.ServerHost + ":" + config.Config.ServerPort,
		WriteTimeout: time.Second * 30,
		ReadTimeout:  time.Second * 30,
		IdleTimeout:  time.Second * 30,
		Handler:      router,
	}

	// Start the server in a separate goroutine
	go func() {
		log.Printf("Server started on http://%s:%s", config.Config.ServerHost, config.Config.ServerPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Gracefully handle server shutdown
	handleGracefulShutdown(server)
}

// handleGracefulShutdown handles the server shutdown on interrupt signals.
func handleGracefulShutdown(server *http.Server) {
	// Wait for interrupt signal for shutdown (e.g., Ctrl+C)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("Shutdown signal received. Shutting down server...")

	// Create a deadline context for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Attempt to gracefully shut down the server
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server gracefully stopped.")
}
```

This script initializes configurations, sets up the PostgreSQL database, optionally connects to Redis if enabled, configures the Gin router, starts the HTTP server, and handles graceful shutdown on receiving interrupt signals.

Feel free to customize further or add more details as needed. This comprehensive README should now provide clear guidance on the project's structure, setup, usage, and contribution process.