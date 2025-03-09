package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/san-data-systems/common/logger"
	"github.com/sirupsen/logrus"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/san-data-systems/common/clients/minio"
	"github.com/san-data-systems/common/utils"

	"github.com/san-data-systems/common/config"
	"github.com/san-data-systems/common/databases"
	"github.com/san-data-systems/project-management-api/routes"
)

// updateOpenAPISpec reads, updates, and writes back the JSON configuration file.
func updateOpenAPISpec(filePath string, mode string, port string) error {
	// Read JSON file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	// Parse JSON into a flexible structure
	var openAPISpec map[string]interface{}
	if err := json.Unmarshal(data, &openAPISpec); err != nil {
		return fmt.Errorf("error decoding JSON: %w", err)
	}

	// Determine protocol
	protocol := "http"
	if mode == "release" {
		protocol = "https"
	}

	// Define the new server entry
	newServer := map[string]interface{}{
		"url":         fmt.Sprintf("%s://localhost:%s/api/v1", protocol, port),
		"description": "Local server",
	}

	// Handle the `servers` field
	if servers, ok := openAPISpec["servers"].([]interface{}); ok {
		exists := false
		for _, s := range servers {
			if srv, ok := s.(map[string]interface{}); ok {
				if srv["url"] == newServer["url"] {
					logger.LogInfo("Local Server already exists.", nil)
					exists = true
					break
				}
			}
		}
		// Append only if the server does not exist
		if !exists {
			openAPISpec["servers"] = append(servers, newServer)
		}
	} else {
		// Create `servers` if it does not exist
		openAPISpec["servers"] = []interface{}{newServer}
	}

	// Write updated JSON file
	updatedData, err := json.MarshalIndent(openAPISpec, "", "  ")
	if err != nil {
		return fmt.Errorf("error encoding JSON: %w", err)
	}

	if err := os.WriteFile(filePath, updatedData, 0644); err != nil {
		return fmt.Errorf("error writing JSON file: %w", err)
	}

	fmt.Println("Local server added successfully.")
	return nil
}

// main is the entry point for the Project Management API server.
func main() {
	// Load application configuration
	config.LoadConfig()
	// Initialize PostgresQL database
	databases.InitPostgresDB()

	// Optionally, initialize Redis if enabled in the config
	if config.Config.UseRedis {
		databases.CheckRedisConnection()
	}

	// Optionally, initialize Redis if enabled in the config
	if config.Config.UseMinIO {
		client, err := minio.NewMinIOClient(
			config.Config.MinIOEndpoint,
			config.Config.MinIOAccessKey,
			config.Config.MinIOSecretKey,
			config.Config.MinIOSSL,
		)
		if err != nil {
			logger.LogError("Failed to initialize MinIO client.", logrus.Fields{"error": err.Error()})
		}

		// Example usage of the MinIO client
		err = client.CheckConnection(context.Background())
		if err != nil {
			logger.LogError("Failed to check connection.", logrus.Fields{"error": err.Error()})
		}
	}

	token, _ := utils.GenerateJWT("vbhadauriya@redcloudcomputing.com", "vbhadauriya", "vbhadauriya@redcloudcomputing.com")
	fmt.Println(token)

	DecodedJWT, _ := utils.DecodeCustomJWTToken(token)
	fmt.Println("Decoded JWT:", DecodedJWT)

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

	if err := updateOpenAPISpec("./docs/openapi.json", config.Config.Mode, config.Config.ServerPort); err != nil {
		log.Fatalf("Failed to update openapi.json file.: %v", err)
	}

	// Start the server with SSL if Mode is 'release' and TLS_CERT/TLS_KEY exist
	if config.Config.Mode == "release" && config.Config.TLSKey != "" && config.Config.TLSCert != "" {
		// Serve with TLS (SSL)
		go func() {
			log.Printf("Server started on https://%s:%s", config.Config.ServerHost, config.Config.ServerPort)
			log.Printf("OpenAPI Specficiation can be access on https://%s:%s/docs/openapi.json", config.Config.ServerHost, config.Config.ServerPort)
			if err := server.ListenAndServeTLS(config.Config.TLSCert, config.Config.TLSKey); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Server failed to start: %v", err)
			}
		}()
	} else {
		// Serve without SSL
		go func() {
			log.Printf("Server started on http://%s:%s", config.Config.ServerHost, config.Config.ServerPort)
			log.Printf("OpenAPI Specficiation can be access on http://%s:%s/docs/openapi.json", config.Config.ServerHost, config.Config.ServerPort)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Server failed to start: %v", err)
			}
		}()
	}

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
