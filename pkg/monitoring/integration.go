package monitoring

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// Integration provides easy integration with common frameworks and libraries
type Integration struct {
	manager *Manager
	config  *MonitoringConfig
}

// NewIntegration creates a new monitoring integration helper
func NewIntegration(manager *Manager) *Integration {
	return &Integration{
		manager: manager,
		config:  manager.config,
	}
}

// SetupGin configures Gin with comprehensive monitoring
func (i *Integration) SetupGin(router *gin.Engine) {
	// Add monitoring middleware
	middlewares := i.manager.GetGinMiddleware()
	for _, middleware := range middlewares {
		router.Use(middleware)
	}
	
	// Add monitoring endpoints if enabled
	if i.config.Metrics.Enabled {
		// Metrics endpoint on the main router for internal access
		router.GET("/internal/metrics", gin.WrapH(i.manager.metrics.GetMetricsHandler()))
	}
	
	if i.config.Health.Enabled {
		// Health endpoints on the main router
		router.GET("/internal/health", gin.WrapF(i.manager.health.HealthHandler))
		router.GET("/internal/health/live", gin.WrapF(i.manager.health.LivenessHandler))
		router.GET("/internal/health/ready", gin.WrapF(i.manager.health.ReadinessHandler))
	}
}

// WrapDBWithMetrics wraps GORM database calls with monitoring
func (i *Integration) WrapDBWithMetrics(db *gorm.DB) *gorm.DB {
	// Add GORM callbacks for monitoring database operations
	db.Callback().Create().Before("gorm:create").Register("monitoring:before_create", i.beforeDBOperation)
	db.Callback().Create().After("gorm:create").Register("monitoring:after_create", i.afterDBOperation)
	
	db.Callback().Query().Before("gorm:query").Register("monitoring:before_query", i.beforeDBOperation)
	db.Callback().Query().After("gorm:query").Register("monitoring:after_query", i.afterDBOperation)
	
	db.Callback().Update().Before("gorm:update").Register("monitoring:before_update", i.beforeDBOperation)
	db.Callback().Update().After("gorm:update").Register("monitoring:after_update", i.afterDBOperation)
	
	db.Callback().Delete().Before("gorm:delete").Register("monitoring:before_delete", i.beforeDBOperation)
	db.Callback().Delete().After("gorm:delete").Register("monitoring:after_delete", i.afterDBOperation)
	
	db.Callback().Raw().Before("gorm:raw").Register("monitoring:before_raw", i.beforeDBOperation)
	db.Callback().Raw().After("gorm:raw").Register("monitoring:after_raw", i.afterDBOperation)
	
	return db
}

// WrapRedisWithMetrics wraps Redis client with monitoring
func (i *Integration) WrapRedisWithMetrics(client *redis.Client) *redis.Client {
	// Add Redis hooks for monitoring
	client.AddHook(&redisMonitoringHook{
		metrics: i.manager.metrics,
		logger:  i.manager.logger,
	})
	
	return client
}

// MonitorBusinessTransaction records business transaction metrics and logs
func (i *Integration) MonitorBusinessTransaction(
	operation, exchange, symbol string,
	amount float64,
	userID string,
	duration time.Duration,
	success bool,
	details map[string]interface{},
) {
	// Record metrics
	direction := "unknown"
	if amount > 0 {
		direction = "buy"
	} else if amount < 0 {
		direction = "sell"
		amount = -amount // Make positive for volume tracking
	}
	
	if success {
		i.manager.metrics.RecordTradingVolume(exchange, symbol, direction, amount)
	}
	
	// Log business event
	i.manager.logger.LogBusiness(operation, exchange, symbol, amount, userID, details)
	
	// Check for business alerts
	if amount > 100000 { // Large transaction threshold
		i.manager.alerts.FireBusinessAlert(
			"large_transaction",
			fmt.Sprintf("Large %s transaction: %.2f %s on %s", operation, amount, symbol, exchange),
			map[string]interface{}{
				"operation": operation,
				"exchange":  exchange,
				"symbol":    symbol,
				"amount":    amount,
				"user_id":   userID,
				"duration":  duration.String(),
			},
		)
	}
}

// MonitorSecurityEvent records security-related events
func (i *Integration) MonitorSecurityEvent(
	eventType, description, userID, ipAddress string,
	severity string,
	success bool,
	details map[string]interface{},
) {
	// Record security metrics
	result := "success"
	if !success {
		result = "failure"
	}
	
	i.manager.metrics.RecordSecurityEvent(eventType, severity)
	
	// Log security event
	i.manager.logger.LogSecurity(eventType, severity, userID, ipAddress, details)
	
	// Fire security alert if needed
	if severity == "critical" || !success {
		i.manager.alerts.FireSecurityAlert(eventType, description, userID, ipAddress, details)
	}
}

// MonitorAPIKeyUsage records API key usage metrics
func (i *Integration) MonitorAPIKeyUsage(keyType, result string) {
	i.manager.metrics.RecordAPIKeyUsage(keyType, result)
}

// MonitorAuthAttempt records authentication attempt metrics
func (i *Integration) MonitorAuthAttempt(provider, result string) {
	i.manager.metrics.RecordAuthAttempt(provider, result)
}

// MonitorRateLimitHit records rate limit violations
func (i *Integration) MonitorRateLimitHit(rule, identifierType string) {
	i.manager.metrics.RecordRateLimitHit(rule, identifierType)
}

// GORM callback functions
func (i *Integration) beforeDBOperation(db *gorm.DB) {
	db.InstanceSet("monitoring:start_time", time.Now())
}

func (i *Integration) afterDBOperation(db *gorm.DB) {
	startTime, exists := db.InstanceGet("monitoring:start_time")
	if !exists {
		return
	}
	
	start, ok := startTime.(time.Time)
	if !ok {
		return
	}
	
	duration := time.Since(start)
	
	// Determine operation type
	operation := "unknown"
	table := "unknown"
	
	if db.Statement != nil {
		if db.Statement.Schema != nil {
			table = db.Statement.Schema.Table
		}
		
		// Determine operation from the callback name
		switch {
		case db.Statement.SQL.String() != "":
			if len(db.Statement.SQL.String()) > 6 {
				operation = db.Statement.SQL.String()[:6]
			}
		default:
			operation = "query"
		}
	}
	
	// Record metrics
	i.manager.metrics.RecordDBQuery(operation, table, duration, db.Error)
	
	// Log if it's a slow query or error
	if db.Error != nil || duration > time.Second {
		i.manager.logger.LogDatabase(operation, table, duration, db.Statement.RowsAffected, db.Error)
	}
}

// Redis monitoring hook
type redisMonitoringHook struct {
	metrics *MetricsCollector
	logger  *Logger
}

func (hook *redisMonitoringHook) BeforeProcess(ctx context.Context, cmd redis.Cmder) (context.Context, error) {
	ctx = context.WithValue(ctx, "monitoring:start_time", time.Now())
	return ctx, nil
}

func (hook *redisMonitoringHook) AfterProcess(ctx context.Context, cmd redis.Cmder) error {
	startTime := ctx.Value("monitoring:start_time")
	if startTime == nil {
		return nil
	}
	
	start, ok := startTime.(time.Time)
	if !ok {
		return nil
	}
	
	duration := time.Since(start)
	command := cmd.Name()
	
	// Record metrics
	hook.metrics.RecordRedisCommand(command, duration, cmd.Err())
	
	// Log errors or slow commands
	if cmd.Err() != nil || duration > 100*time.Millisecond {
		hook.logger.Debug("Redis command %s took %v, error: %v", command, duration, cmd.Err())
	}
	
	return nil
}

func (hook *redisMonitoringHook) BeforeProcessPipeline(ctx context.Context, cmds []redis.Cmder) (context.Context, error) {
	ctx = context.WithValue(ctx, "monitoring:start_time", time.Now())
	return ctx, nil
}

func (hook *redisMonitoringHook) AfterProcessPipeline(ctx context.Context, cmds []redis.Cmder) error {
	startTime := ctx.Value("monitoring:start_time")
	if startTime == nil {
		return nil
	}
	
	start, ok := startTime.(time.Time)
	if !ok {
		return nil
	}
	
	duration := time.Since(start)
	
	// Record pipeline metrics
	hook.metrics.RecordRedisCommand("pipeline", duration, nil)
	
	// Check for errors in pipeline
	errorCount := 0
	for _, cmd := range cmds {
		if cmd.Err() != nil {
			errorCount++
		}
	}
	
	if errorCount > 0 || duration > 500*time.Millisecond {
		hook.logger.Debug("Redis pipeline with %d commands took %v, %d errors", 
			len(cmds), duration, errorCount)
	}
	
	return nil
}

// Helper functions for common monitoring patterns

// WithRequestMonitoring wraps a handler with request-specific monitoring
func (i *Integration) WithRequestMonitoring(handler gin.HandlerFunc) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// Add request-specific monitoring context
		start := time.Now()
		
		// Process request
		handler(c)
		
		// Record additional metrics if needed
		duration := time.Since(start)
		if duration > 5*time.Second {
			i.manager.alerts.FirePerformanceAlert(
				"slow_request",
				duration.Milliseconds(),
				5000,
				"http",
			)
		}
	})
}

// WithBusinessTransactionMonitoring wraps business operations with monitoring
func (i *Integration) WithBusinessTransactionMonitoring(
	operation string,
	fn func() error,
) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start)
	
	success := err == nil
	
	// Log the business operation
	details := map[string]interface{}{
		"duration":  duration.String(),
		"success":   success,
		"operation": operation,
	}
	
	if err != nil {
		details["error"] = err.Error()
	}
	
	i.manager.logger.WithFields(details).Info("Business operation: %s", operation)
	
	return err
}

// GetMetrics returns current application metrics snapshot
func (i *Integration) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"monitoring_enabled": i.config.Enabled,
		"service":           i.config.Service,
		"version":           i.config.Version,
		"environment":       i.config.Environment,
		"started":           i.manager.IsStarted(),
		"health_status":     i.manager.health.IsHealthy(),
		"active_alerts":     len(i.manager.alerts.GetActiveAlerts()),
	}
}