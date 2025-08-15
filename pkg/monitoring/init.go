package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"tiris-backend/config"

	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gorm.io/gorm"
)

// MonitoringManager manages all monitoring components
type MonitoringManager struct {
	config           *config.MonitoringConfig
	metricsCollector *MetricsCollector
	logger           *Logger
	alertManager     *AlertManager
	healthManager    *HealthManager
	metricsServer    *http.Server
	stopCh           chan struct{}
	wg               sync.WaitGroup
}

// NewMonitoringManager creates a new monitoring manager
func NewMonitoringManager(cfg *config.MonitoringConfig) (*MonitoringManager, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid monitoring config: %w", err)
	}

	mm := &MonitoringManager{
		config: cfg,
		stopCh: make(chan struct{}),
	}

	// Initialize components
	if err := mm.initializeComponents(); err != nil {
		return nil, fmt.Errorf("failed to initialize monitoring components: %w", err)
	}

	return mm, nil
}

// initializeComponents initializes all monitoring components
func (mm *MonitoringManager) initializeComponents() error {
	// Initialize metrics collector
	if mm.config.MetricsEnabled {
		mm.metricsCollector = NewMetricsCollector()
		if err := mm.metricsCollector.Register(); err != nil {
			return fmt.Errorf("failed to register metrics: %w", err)
		}
	}

	// Initialize logger
	if mm.config.LoggingEnabled {
		loggerConfig := LoggerConfig{
			Level:  mm.config.LogLevel,
			Format: mm.config.LogFormat,
			Output: mm.config.LogOutput,
		}
		
		var err error
		mm.logger, err = NewLogger(loggerConfig)
		if err != nil {
			return fmt.Errorf("failed to create logger: %w", err)
		}
	}

	// Initialize alert manager
	if mm.config.AlertsEnabled {
		alertConfig := AlertManagerConfig{
			Receivers: mm.config.GetEnabledAlertReceivers(),
		}
		
		var err error
		mm.alertManager, err = NewAlertManager(alertConfig)
		if err != nil {
			return fmt.Errorf("failed to create alert manager: %w", err)
		}
	}

	// Initialize health manager
	if mm.config.HealthCheckEnabled {
		mm.healthManager = NewHealthManager()
	}

	return nil
}

// Start starts the monitoring manager and all its components
func (mm *MonitoringManager) Start(ctx context.Context) error {
	// Start metrics server
	if mm.config.MetricsEnabled && mm.metricsCollector != nil {
		if err := mm.startMetricsServer(); err != nil {
			return fmt.Errorf("failed to start metrics server: %w", err)
		}
	}

	// Start health checks
	if mm.config.HealthCheckEnabled && mm.healthManager != nil {
		mm.startHealthChecks(ctx)
	}

	return nil
}

// Stop stops the monitoring manager and all its components
func (mm *MonitoringManager) Stop(ctx context.Context) error {
	close(mm.stopCh)

	// Stop metrics server
	if mm.metricsServer != nil {
		if err := mm.metricsServer.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown metrics server: %w", err)
		}
	}

	// Wait for all goroutines to finish
	done := make(chan struct{})
	go func() {
		mm.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// startMetricsServer starts the Prometheus metrics HTTP server
func (mm *MonitoringManager) startMetricsServer() error {
	mux := http.NewServeMux()
	mux.Handle(mm.config.MetricsPath, promhttp.Handler())

	// Add health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "healthy"}`))
	})

	mm.metricsServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", mm.config.MetricsPort),
		Handler: mux,
	}

	mm.wg.Add(1)
	go func() {
		defer mm.wg.Done()
		if err := mm.metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			mm.LogError("Metrics server error", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()

	return nil
}

// startHealthChecks starts periodic health checks
func (mm *MonitoringManager) startHealthChecks(ctx context.Context) {
	mm.wg.Add(1)
	go func() {
		defer mm.wg.Done()
		ticker := time.NewTicker(mm.config.HealthCheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				mm.performHealthChecks(ctx)
			case <-mm.stopCh:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

// performHealthChecks performs all configured health checks
func (mm *MonitoringManager) performHealthChecks(ctx context.Context) {
	if mm.healthManager == nil {
		return
	}

	// Create context with timeout
	checkCtx, cancel := context.WithTimeout(ctx, mm.config.HealthCheckTimeout)
	defer cancel()

	// Run all health checks
	results := mm.healthManager.CheckAll(checkCtx)

	// Log unhealthy services
	for _, result := range results {
		if result.Status == StatusUnhealthy {
			mm.LogError("Health check failed", map[string]interface{}{
				"service": result.Service,
				"error":   result.Error,
			})

			// Fire alert if configured
			if mm.alertManager != nil {
				mm.alertManager.FireHealthAlert(result.Service, result.Error, map[string]interface{}{
					"check_type": result.Service,
					"timestamp":  result.LastCheck,
				})
			}
		}
	}

	// Record metrics
	if mm.metricsCollector != nil {
		for _, result := range results {
			healthy := 0.0
			if result.Status == StatusHealthy {
				healthy = 1.0
			}
			mm.metricsCollector.RecordHealthStatus(result.Service, healthy)
		}
	}
}

// SetupDatabaseMonitoring sets up database monitoring with GORM hooks
func (mm *MonitoringManager) SetupDatabaseMonitoring(db *gorm.DB) error {
	if !mm.config.DatabaseMonitoringEnabled || mm.metricsCollector == nil {
		return nil
	}

	// Add database health checker
	if mm.healthManager != nil {
		sqlDB, err := db.DB()
		if err != nil {
			return fmt.Errorf("failed to get sql.DB from GORM: %w", err)
		}
		mm.healthManager.AddChecker(NewDatabaseHealthChecker(sqlDB))
	}

	// Add GORM callbacks for monitoring
	err := db.Callback().Query().Before("gorm:query").Register("monitoring:before_query", mm.beforeDBQuery)
	if err != nil {
		return fmt.Errorf("failed to register before query callback: %w", err)
	}

	err = db.Callback().Query().After("gorm:query").Register("monitoring:after_query", mm.afterDBQuery)
	if err != nil {
		return fmt.Errorf("failed to register after query callback: %w", err)
	}

	return nil
}

// SetupRedisMonitoring sets up Redis monitoring
func (mm *MonitoringManager) SetupRedisMonitoring(client *redis.Client) error {
	if !mm.config.RedisMonitoringEnabled {
		return nil
	}

	// Add Redis health checker
	if mm.healthManager != nil {
		mm.healthManager.AddChecker(NewRedisHealthChecker(client))
	}

	// Add Redis hook for command monitoring
	client.AddHook(&redisMonitoringHook{
		metricsCollector: mm.metricsCollector,
		slowThreshold:    mm.config.RedisSlowCommandThreshold,
	})

	return nil
}

// beforeDBQuery is called before database queries
func (mm *MonitoringManager) beforeDBQuery(db *gorm.DB) {
	db.Set("monitoring:start_time", time.Now())
}

// afterDBQuery is called after database queries
func (mm *MonitoringManager) afterDBQuery(db *gorm.DB) {
	startTime, exists := db.Get("monitoring:start_time")
	if !exists {
		return
	}

	start := startTime.(time.Time)
	duration := time.Since(start)

	// Record metrics
	if mm.metricsCollector != nil {
		operation := "unknown"
		if db.Statement != nil {
			switch {
			case db.Statement.SQL.String() != "":
				if db.Statement.SQL.Len() > 6 {
					operation = db.Statement.SQL.String()[:6]
				}
			}
		}
		
		mm.metricsCollector.RecordDatabaseQuery(operation, duration, db.Error == nil)
	}

	// Log slow queries
	if duration > mm.config.SlowQueryThreshold {
		mm.LogWarn("Slow database query detected", map[string]interface{}{
			"duration_ms": duration.Milliseconds(),
			"sql":         db.Statement.SQL.String(),
			"error":       db.Error,
		})
	}
}

// redisMonitoringHook implements redis.Hook for monitoring Redis commands
type redisMonitoringHook struct {
	metricsCollector *MetricsCollector
	slowThreshold    time.Duration
}

func (h *redisMonitoringHook) BeforeProcess(ctx context.Context, cmd redis.Cmder) (context.Context, error) {
	return context.WithValue(ctx, "monitoring:start_time", time.Now()), nil
}

func (h *redisMonitoringHook) AfterProcess(ctx context.Context, cmd redis.Cmder) error {
	startTime, ok := ctx.Value("monitoring:start_time").(time.Time)
	if !ok {
		return nil
	}

	duration := time.Since(startTime)
	
	if h.metricsCollector != nil {
		h.metricsCollector.RecordRedisCommand(cmd.Name(), duration, cmd.Err() == nil)
	}

	return nil
}

func (h *redisMonitoringHook) BeforeProcessPipeline(ctx context.Context, cmds []redis.Cmder) (context.Context, error) {
	return context.WithValue(ctx, "monitoring:start_time", time.Now()), nil
}

func (h *redisMonitoringHook) AfterProcessPipeline(ctx context.Context, cmds []redis.Cmder) error {
	startTime, ok := ctx.Value("monitoring:start_time").(time.Time)
	if !ok {
		return nil
	}

	duration := time.Since(startTime)
	
	if h.metricsCollector != nil {
		hasError := false
		for _, cmd := range cmds {
			if cmd.Err() != nil {
				hasError = true
				break
			}
		}
		h.metricsCollector.RecordRedisCommand("pipeline", duration, !hasError)
	}

	return nil
}

// Convenience methods for logging
func (mm *MonitoringManager) LogInfo(message string, fields map[string]interface{}) {
	if mm.logger != nil {
		mm.logger.LogInfo(message, fields)
	}
}

func (mm *MonitoringManager) LogWarn(message string, fields map[string]interface{}) {
	if mm.logger != nil {
		mm.logger.LogWarn(message, fields)
	}
}

func (mm *MonitoringManager) LogError(message string, fields map[string]interface{}) {
	if mm.logger != nil {
		mm.logger.LogError(message, fields)
	}
}

// GetMetricsCollector returns the metrics collector
func (mm *MonitoringManager) GetMetricsCollector() *MetricsCollector {
	return mm.metricsCollector
}

// GetLogger returns the logger
func (mm *MonitoringManager) GetLogger() *Logger {
	return mm.logger
}

// GetAlertManager returns the alert manager
func (mm *MonitoringManager) GetAlertManager() *AlertManager {
	return mm.alertManager
}

// GetHealthManager returns the health manager
func (mm *MonitoringManager) GetHealthManager() *HealthManager {
	return mm.healthManager
}