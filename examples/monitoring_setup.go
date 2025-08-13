package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tiris-backend/internal/config"
	"tiris-backend/pkg/monitoring"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Example of how to integrate the monitoring system into your application
func main() {
	// Load application configuration
	appConfig := config.Load() // Your app config loader

	// Create database connection
	db, err := gorm.Open(postgres.Open(appConfig.Database.DSN), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Create Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     appConfig.Redis.Address,
		Password: appConfig.Redis.Password,
		DB:       appConfig.Redis.DB,
	})

	// Load monitoring configuration
	monitoringConfig := monitoring.LoadMonitoringConfig()

	// Create monitoring manager
	monitoringManager := monitoring.NewManager(monitoringConfig, db, redisClient)

	// Create monitoring integration helper
	integration := monitoring.NewIntegration(monitoringManager)

	// Wrap database and Redis clients with monitoring
	db = integration.WrapDBWithMetrics(db)
	redisClient = integration.WrapRedisWithMetrics(redisClient)

	// Start monitoring system
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := monitoringManager.Start(ctx); err != nil {
		log.Fatal("Failed to start monitoring system:", err)
	}

	// Create Gin router
	router := gin.New()

	// Setup monitoring middleware and endpoints
	integration.SetupGin(router)

	// Add your application routes
	setupAppRoutes(router, db, redisClient, integration)

	// Create HTTP server
	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server:", err)
		}
	}()

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Println("Shutting down...")

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Shutdown HTTP server
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	// Stop monitoring system
	if err := monitoringManager.Stop(shutdownCtx); err != nil {
		log.Printf("Monitoring shutdown error: %v", err)
	}

	log.Println("Server stopped")
}

// Example of setting up application routes with monitoring integration
func setupAppRoutes(router *gin.Engine, db *gorm.DB, redis *redis.Client, integration *monitoring.Integration) {
	// Get logger and metrics from monitoring
	logger := integration.GetLogger()
	
	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// User routes with monitoring
		users := v1.Group("/users")
		{
			users.GET("/profile", integration.WithRequestMonitoring(func(c *gin.Context) {
				// Simulate getting user profile
				userID := c.GetString("user_id")
				
				// Monitor business transaction
				err := integration.WithBusinessTransactionMonitoring("get_user_profile", func() error {
					// Simulate database query
					time.Sleep(50 * time.Millisecond)
					return nil
				})
				
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get profile"})
					return
				}
				
				logger.Info("User profile retrieved for user %s", userID)
				c.JSON(http.StatusOK, gin.H{"message": "Profile retrieved"})
			}))
			
			users.PUT("/profile", integration.WithRequestMonitoring(func(c *gin.Context) {
				// Update user profile with monitoring
				userID := c.GetString("user_id")
				
				// Monitor security event
				integration.MonitorSecurityEvent(
					"profile_update",
					"User profile updated",
					userID,
					c.ClientIP(),
					"info",
					true,
					map[string]interface{}{
						"user_agent": c.GetHeader("User-Agent"),
						"endpoint":   c.Request.URL.Path,
					},
				)
				
				c.JSON(http.StatusOK, gin.H{"message": "Profile updated"})
			}))
		}
		
		// Trading routes with business monitoring
		trading := v1.Group("/trading")
		{
			trading.POST("/trade", integration.WithRequestMonitoring(func(c *gin.Context) {
				userID := c.GetString("user_id")
				
				// Simulate trade execution with monitoring
				integration.MonitorBusinessTransaction(
					"trade_execution",
					"binance",
					"BTC/USD",
					1.5, // amount
					userID,
					250*time.Millisecond, // execution time
					true, // success
					map[string]interface{}{
						"trade_type": "market",
						"side":       "buy",
					},
				)
				
				c.JSON(http.StatusOK, gin.H{"message": "Trade executed"})
			}))
		}
		
		// Authentication routes with security monitoring
		auth := v1.Group("/auth")
		{
			auth.POST("/login", func(c *gin.Context) {
				// Simulate login attempt
				username := c.PostForm("username")
				password := c.PostForm("password")
				
				// Simulate authentication
				success := password == "correct_password"
				
				// Monitor auth attempt
				result := "failed"
				if success {
					result = "success"
				}
				integration.MonitorAuthAttempt("local", result)
				
				// Monitor security event for failed logins
				if !success {
					integration.MonitorSecurityEvent(
						"login_failed",
						"Failed login attempt",
						username,
						c.ClientIP(),
						"warning",
						false,
						map[string]interface{}{
							"username":   username,
							"user_agent": c.GetHeader("User-Agent"),
						},
					)
				}
				
				if success {
					c.JSON(http.StatusOK, gin.H{"message": "Login successful"})
				} else {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
				}
			})
		}
		
		// API key routes with usage monitoring
		apiKeys := v1.Group("/api-keys")
		{
			apiKeys.Use(func(c *gin.Context) {
				// Monitor API key usage
				keyType := "user" // or "admin", "service", etc.
				apiKey := c.GetHeader("X-API-Key")
				
				if apiKey != "" {
					result := "success" // or "invalid", "expired", etc.
					integration.MonitorAPIKeyUsage(keyType, result)
				}
				
				c.Next()
			})
			
			apiKeys.POST("/create", integration.WithRequestMonitoring(func(c *gin.Context) {
				// Create API key with security monitoring
				userID := c.GetString("user_id")
				
				integration.MonitorSecurityEvent(
					"api_key_created",
					"New API key created",
					userID,
					c.ClientIP(),
					"info",
					true,
					map[string]interface{}{
						"permissions": []string{"read", "write"},
					},
				)
				
				c.JSON(http.StatusCreated, gin.H{"message": "API key created"})
			}))
		}
	}
	
	// External API routes (rate limited)
	external := router.Group("/api/v1/external")
	external.Use(func(c *gin.Context) {
		// Simulate rate limiting check
		userID := c.GetString("user_id")
		if userID == "rate_limited_user" {
			integration.MonitorRateLimitHit("api_requests", "user")
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
			c.Abort()
			return
		}
		c.Next()
	})
	{
		external.GET("/status", func(c *gin.Context) {
			metrics := integration.GetMetrics()
			c.JSON(http.StatusOK, gin.H{
				"status":  "ok",
				"metrics": metrics,
			})
		})
	}
	
	// Admin routes with enhanced monitoring
	admin := router.Group("/admin")
	admin.Use(func(c *gin.Context) {
		// Log all admin actions
		userID := c.GetString("user_id")
		
		integration.MonitorSecurityEvent(
			"admin_access",
			"Admin endpoint accessed",
			userID,
			c.ClientIP(),
			"info",
			true,
			map[string]interface{}{
				"endpoint":   c.Request.URL.Path,
				"method":     c.Request.Method,
				"user_agent": c.GetHeader("User-Agent"),
			},
		)
		
		c.Next()
	})
	{
		admin.GET("/metrics", func(c *gin.Context) {
			metrics := integration.GetMetrics()
			c.JSON(http.StatusOK, metrics)
		})
		
		admin.GET("/health", func(c *gin.Context) {
			healthStatus := integration.GetHealthMonitor().GetLastReport()
			c.JSON(http.StatusOK, healthStatus)
		})
		
		admin.GET("/alerts", func(c *gin.Context) {
			alerts := integration.GetAlertManager().GetActiveAlerts()
			c.JSON(http.StatusOK, gin.H{
				"active_alerts": alerts,
				"count":         len(alerts),
			})
		})
	}
}