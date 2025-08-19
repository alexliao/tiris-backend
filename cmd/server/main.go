package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tiris-backend/internal/api"
	"tiris-backend/internal/config"
	"tiris-backend/internal/database"
	"tiris-backend/internal/metrics"
	"tiris-backend/internal/nats"
	"tiris-backend/internal/repositories"

	"github.com/gin-gonic/gin"
)

// @title Tiris Backend API
// @version 1.0
// @description A Go-based microservice for quantitative trading data management, providing RESTful APIs for user management, exchange integration, and trading operations.
// @termsOfService https://tiris.ai/terms

// @contact.name Tiris API Support
// @contact.url https://tiris.ai/support
// @contact.email support@tiris.ai

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter JWT Bearer token in the format: Bearer {token}

// @x-extension-openapi {"info":{"x-logo":{"url":"https://tiris.ai/logo.png"}}}

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	db, err := database.Initialize(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close(db)

	// Initialize repositories
	repos := repositories.NewRepositories(db.DB)

	// Initialize NATS manager only if enabled
	var natsManager *nats.Manager
	if cfg.NATS.Enabled {
		var err error
		natsManager, err = nats.NewManager(cfg.NATS, repos)
		if err != nil {
			log.Fatalf("Failed to initialize NATS: %v", err)
		}
		defer natsManager.Stop()

		// Start NATS event consumers
		if err := natsManager.Start(); err != nil {
			log.Fatalf("Failed to start NATS consumers: %v", err)
		}
	}

	// Set Gin mode
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize API server
	apiServer := api.NewServer(cfg, repos, db, natsManager)
	router := apiServer.SetupRoutes()

	// Start metrics updater
	metricsUpdater := metrics.NewMetricsUpdater(apiServer.GetMetrics(), repos, 30*time.Second)
	metricsUpdater.Start()
	defer metricsUpdater.Stop()

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeout) * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on port %s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
