package api

import (
	"context"
	"net/http"
	"time"

	"tiris-backend/internal/database"
	"tiris-backend/internal/nats"

	"github.com/gin-gonic/gin"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	db          *database.DB
	natsManager *nats.Manager
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(db *database.DB, natsManager *nats.Manager) *HealthHandler {
	return &HealthHandler{
		db:          db,
		natsManager: natsManager,
	}
}

// HealthStatus represents the health status of a component
type HealthStatus struct {
	Status  string                 `json:"status"`
	Message string                 `json:"message,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// HealthResponse represents the overall health response
type HealthResponse struct {
	Status       string                  `json:"status"`
	Timestamp    string                  `json:"timestamp"`
	Version      string                  `json:"version"`
	Dependencies map[string]HealthStatus `json:"dependencies"`
}

// LivenessProbe handles Kubernetes liveness probe
// @Summary Liveness probe
// @Description Kubernetes liveness probe - checks if the application is running
// @Tags Health
// @Produce json
// @Success 200 {object} SuccessResponse
// @Router /health/live [get]
func (h *HealthHandler) LivenessProbe(c *gin.Context) {
	response := SuccessResponse{
		Success: true,
		Data: gin.H{
			"status":    "healthy",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"message":   "Service is alive",
		},
		Metadata: ResponseMetadata{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			TraceID:   getTraceID(c),
		},
	}

	c.JSON(http.StatusOK, response)
}

// ReadinessProbe handles Kubernetes readiness probe
// @Summary Readiness probe
// @Description Kubernetes readiness probe - checks if the application is ready to serve traffic
// @Tags Health
// @Produce json
// @Success 200 {object} HealthResponse
// @Failure 503 {object} ErrorResponse
// @Router /health/ready [get]
func (h *HealthHandler) ReadinessProbe(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	dependencies := make(map[string]HealthStatus)
	overallHealthy := true

	// Check database connectivity
	dbStatus := h.checkDatabase(ctx)
	dependencies["database"] = dbStatus
	if dbStatus.Status != "healthy" {
		overallHealthy = false
	}

	// Check NATS connectivity
	natsStatus := h.checkNATS(ctx)
	dependencies["nats"] = natsStatus
	if natsStatus.Status != "healthy" && natsStatus.Status != "disabled" {
		overallHealthy = false
	}

	// Determine overall status
	status := "healthy"
	httpStatus := http.StatusOK
	if !overallHealthy {
		status = "unhealthy"
		httpStatus = http.StatusServiceUnavailable
	}

	healthResponse := HealthResponse{
		Status:       status,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		Version:      "1.0.0", // This could be injected from build info
		Dependencies: dependencies,
	}

	if overallHealthy {
		c.JSON(httpStatus, SuccessResponse{
			Success: true,
			Data:    healthResponse,
			Metadata: ResponseMetadata{
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				TraceID:   getTraceID(c),
			},
		})
	} else {
		c.JSON(httpStatus, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "SERVICE_UNAVAILABLE",
				Message: "One or more dependencies are unhealthy",
			},
			Metadata: ResponseMetadata{
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				TraceID:   getTraceID(c),
			},
		})
	}
}

// HealthCheck provides detailed health information
// @Summary Detailed health check
// @Description Provides detailed health information about the service and its dependencies
// @Tags Health
// @Produce json
// @Success 200 {object} HealthResponse
// @Failure 503 {object} ErrorResponse
// @Router /health [get]
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	dependencies := make(map[string]HealthStatus)
	overallHealthy := true

	// Check database with detailed info
	dbStatus := h.checkDatabaseDetailed(ctx)
	dependencies["database"] = dbStatus
	if dbStatus.Status != "healthy" {
		overallHealthy = false
	}

	// Check NATS with detailed info
	natsStatus := h.checkNATSDetailed(ctx)
	dependencies["nats"] = natsStatus
	if natsStatus.Status != "healthy" && natsStatus.Status != "disabled" {
		overallHealthy = false
	}

	// Add system info
	systemStatus := h.getSystemInfo()
	dependencies["system"] = systemStatus

	// Determine overall status
	status := "healthy"
	httpStatus := http.StatusOK
	if !overallHealthy {
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	}

	healthResponse := HealthResponse{
		Status:       status,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		Version:      "1.0.0",
		Dependencies: dependencies,
	}

	if overallHealthy {
		c.JSON(httpStatus, SuccessResponse{
			Success: true,
			Data:    healthResponse,
			Metadata: ResponseMetadata{
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				TraceID:   getTraceID(c),
			},
		})
	} else {
		c.JSON(httpStatus, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "SERVICE_DEGRADED",
				Message: "Service is running but some dependencies are unhealthy",
			},
			Metadata: ResponseMetadata{
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				TraceID:   getTraceID(c),
			},
		})
	}
}

// checkDatabase performs a basic database connectivity check
func (h *HealthHandler) checkDatabase(ctx context.Context) HealthStatus {
	if h.db == nil {
		return HealthStatus{
			Status:  "unhealthy",
			Message: "Database not initialized",
		}
	}

	// Simple ping to check connectivity
	sqlDB, err := h.db.DB.DB()
	if err != nil {
		return HealthStatus{
			Status:  "unhealthy",
			Message: "Failed to get database connection",
		}
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		return HealthStatus{
			Status:  "unhealthy",
			Message: "Database ping failed: " + err.Error(),
		}
	}

	return HealthStatus{
		Status:  "healthy",
		Message: "Database is accessible",
	}
}

// checkDatabaseDetailed performs a detailed database health check
func (h *HealthHandler) checkDatabaseDetailed(ctx context.Context) HealthStatus {
	if h.db == nil {
		return HealthStatus{
			Status:  "unhealthy",
			Message: "Database not initialized",
		}
	}

	sqlDB, err := h.db.DB.DB()
	if err != nil {
		return HealthStatus{
			Status:  "unhealthy",
			Message: "Failed to get database connection",
		}
	}

	// Check connectivity
	if err := sqlDB.PingContext(ctx); err != nil {
		return HealthStatus{
			Status:  "unhealthy",
			Message: "Database ping failed: " + err.Error(),
		}
	}

	// Get connection pool stats
	stats := sqlDB.Stats()
	details := map[string]interface{}{
		"open_connections":     stats.OpenConnections,
		"in_use":              stats.InUse,
		"idle":                stats.Idle,
		"max_open_connections": stats.MaxOpenConnections,
		"max_idle_closed":     stats.MaxIdleClosed,
		"max_lifetime_closed":  stats.MaxLifetimeClosed,
		"wait_count":          stats.WaitCount,
		"wait_duration":       stats.WaitDuration.String(),
	}

	return HealthStatus{
		Status:  "healthy",
		Message: "Database is accessible with connection pool statistics",
		Details: details,
	}
}

// checkNATS performs a basic NATS connectivity check
func (h *HealthHandler) checkNATS(ctx context.Context) HealthStatus {
	if h.natsManager == nil {
		return HealthStatus{
			Status:  "disabled",
			Message: "NATS is disabled",
		}
	}

	if !h.natsManager.IsConnected() {
		return HealthStatus{
			Status:  "unhealthy",
			Message: "NATS connection is not established",
		}
	}

	return HealthStatus{
		Status:  "healthy",
		Message: "NATS is connected",
	}
}

// checkNATSDetailed performs a detailed NATS health check
func (h *HealthHandler) checkNATSDetailed(ctx context.Context) HealthStatus {
	if h.natsManager == nil {
		return HealthStatus{
			Status:  "disabled",
			Message: "NATS is disabled",
		}
	}

	if !h.natsManager.IsConnected() {
		return HealthStatus{
			Status:  "unhealthy",
			Message: "NATS connection is not established",
		}
	}

	// Get detailed connection info
	details := h.natsManager.GetConnectionStats()
	if details == nil {
		return HealthStatus{
			Status:  "unhealthy",
			Message: "Unable to retrieve connection statistics",
		}
	}

	return HealthStatus{
		Status:  "healthy",
		Message: "NATS is connected with detailed statistics",
		Details: details,
	}
}

// getSystemInfo provides basic system information
func (h *HealthHandler) getSystemInfo() HealthStatus {
	details := map[string]interface{}{
		"uptime":      time.Since(startTime).String(),
		"start_time":  startTime.Format(time.RFC3339),
		"go_version":  "1.23",  // This could be injected from build info
		"environment": "development", // This should come from config
	}

	return HealthStatus{
		Status:  "healthy",
		Message: "System information",
		Details: details,
	}
}

// startTime tracks when the service started
var startTime = time.Now()