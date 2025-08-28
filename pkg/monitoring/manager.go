package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// Manager provides a centralized monitoring system
type Manager struct {
	config   *MonitoringConfig
	metrics  *MetricsCollector
	logger   *Logger
	alerts   *AlertManager
	health   *HealthMonitor
	
	// Dependencies
	db    *gorm.DB
	redis *redis.Client
	
	// State
	mu      sync.RWMutex
	started bool
	
	// HTTP servers for monitoring endpoints
	metricsServer *http.Server
	healthServer  *http.Server
}

// NewManager creates a new monitoring manager
func NewManager(config *MonitoringConfig, db *gorm.DB, redis *redis.Client) *Manager {
	if config == nil {
		config = LoadMonitoringConfig()
	}
	
	// Validate configuration
	config.Validate()
	
	// Create logger
	logger := NewLogger(config.Logging)
	
	// Create components
	metrics := NewMetricsCollector()
	alerts := NewAlertManager(config.Service, config.Environment, logger)
	health := NewHealthMonitor(config.Service, config.Version, logger, alerts)
	
	manager := &Manager{
		config:  config,
		metrics: metrics,
		logger:  logger,
		alerts:  alerts,
		health:  health,
		db:      db,
		redis:   redis,
	}
	
	// Configure alert rules and receivers
	manager.setupAlerting()
	
	// Configure health checks
	manager.setupHealthChecks()
	
	return manager
}

// Start starts all monitoring services
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.started {
		return fmt.Errorf("monitoring manager already started")
	}
	
	m.logger.Info("Starting monitoring system")
	
	// Start metrics collection
	if m.config.Metrics.Enabled {
		if err := m.startMetricsServer(); err != nil {
			return fmt.Errorf("failed to start metrics server: %w", err)
		}
		m.metrics.StartMetricsCollection(ctx)
		m.logger.Info("Metrics collection started on port %d", m.config.Metrics.Port)
	}
	
	// Start health monitoring
	if m.config.Health.Enabled {
		if err := m.startHealthServer(); err != nil {
			return fmt.Errorf("failed to start health server: %w", err)
		}
		go m.health.StartMonitoring(ctx)
		m.logger.Info("Health monitoring started on port %d", m.config.Health.Port)
	}
	
	// Start background tasks
	go m.startBackgroundTasks(ctx)
	
	m.started = true
	m.logger.Info("Monitoring system started successfully")
	
	return nil
}

// Stop stops all monitoring services
func (m *Manager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if !m.started {
		return nil
	}
	
	m.logger.Info("Stopping monitoring system")
	
	// Stop servers
	if m.metricsServer != nil {
		if err := m.metricsServer.Shutdown(ctx); err != nil {
			m.logger.Error("Error shutting down metrics server: %v", err)
		}
	}
	
	if m.healthServer != nil {
		if err := m.healthServer.Shutdown(ctx); err != nil {
			m.logger.Error("Error shutting down health server: %v", err)
		}
	}
	
	m.started = false
	m.logger.Info("Monitoring system stopped")
	
	return nil
}

// GetGinMiddleware returns Gin middleware for HTTP monitoring
func (m *Manager) GetGinMiddleware() []gin.HandlerFunc {
	var middleware []gin.HandlerFunc
	
	// Add metrics middleware
	if m.config.Metrics.Enabled {
		middleware = append(middleware, m.metrics.HTTPMiddleware())
	}
	
	// Add logging middleware
	middleware = append(middleware, m.logger.GinMiddleware())
	
	return middleware
}

// GetMetricsCollector returns the metrics collector
func (m *Manager) GetMetricsCollector() *MetricsCollector {
	return m.metrics
}

// GetLogger returns the logger
func (m *Manager) GetLogger() *Logger {
	return m.logger
}

// GetAlertManager returns the alert manager
func (m *Manager) GetAlertManager() *AlertManager {
	return m.alerts
}

// GetHealthMonitor returns the health monitor
func (m *Manager) GetHealthMonitor() *HealthMonitor {
	return m.health
}

// Internal methods

func (m *Manager) setupAlerting() {
	// Add configured receivers
	for _, receiverConfig := range m.config.Alerting.Receivers {
		var receiver AlertReceiver
		
		switch receiverConfig.Type {
		case "webhook":
			receiver = NewWebhookReceiver(
				receiverConfig.Name,
				receiverConfig.URL,
				receiverConfig.Headers,
			)
		case "slack":
			receiver = NewSlackReceiver(
				receiverConfig.Name,
				receiverConfig.URL,
				receiverConfig.Channel,
			)
		default:
			m.logger.Warn("Unknown receiver type: %s", receiverConfig.Type)
			continue
		}
		
		m.alerts.AddReceiver(receiver)
	}
	
	// Add configured alert rules
	for _, ruleConfig := range m.config.Alerting.Rules {
		if !ruleConfig.Enabled {
			continue
		}
		
		duration, err := time.ParseDuration(ruleConfig.Duration)
		if err != nil {
			m.logger.Error("Invalid duration for rule %s: %v", ruleConfig.Name, err)
			continue
		}
		
		cooldown, err := time.ParseDuration(ruleConfig.Cooldown)
		if err != nil {
			m.logger.Error("Invalid cooldown for rule %s: %v", ruleConfig.Name, err)
			continue
		}
		
		var severity AlertSeverity
		switch ruleConfig.Severity {
		case "info":
			severity = SeverityInfo
		case "warning":
			severity = SeverityWarning
		case "critical":
			severity = SeverityCritical
		default:
			severity = SeverityWarning
		}
		
		var conditionWindow time.Duration
		if ruleConfig.Condition.Window != "" {
			conditionWindow, _ = time.ParseDuration(ruleConfig.Condition.Window)
		}
		
		rule := &AlertRule{
			Name:        ruleConfig.Name,
			Description: ruleConfig.Description,
			Severity:    severity,
			Component:   ruleConfig.Component,
			Condition: AlertCondition{
				Type:      ruleConfig.Condition.Type,
				Metric:    ruleConfig.Condition.Metric,
				Operator:  ruleConfig.Condition.Operator,
				Threshold: ruleConfig.Condition.Threshold,
				Window:    conditionWindow,
			},
			Duration:    duration,
			Cooldown:    cooldown,
			Enabled:     true,
			Labels:      ruleConfig.Labels,
			Annotations: ruleConfig.Annotations,
		}
		
		m.alerts.AddRule(rule)
	}
}

func (m *Manager) setupHealthChecks() {
	// Database health check
	if m.db != nil {
		dbChecker := NewDatabaseHealthChecker("database", m.db, true)
		m.health.AddChecker(dbChecker)
	}
	
	// Redis health check
	if m.redis != nil {
		redisChecker := NewRedisHealthChecker("redis", m.redis, true)
		m.health.AddChecker(redisChecker)
	}
	
	// Memory health check
	memoryChecker := NewMemoryHealthChecker("memory", 80.0, 90.0, false)
	m.health.AddChecker(memoryChecker)
	
	// Set health check configuration
	m.health.SetInterval(m.config.Health.Interval)
	m.health.SetTimeout(m.config.Health.Timeout)
}

func (m *Manager) startMetricsServer() error {
	mux := http.NewServeMux()
	mux.Handle(m.config.Metrics.Path, m.metrics.GetMetricsHandler())
	
	// Add debug endpoints
	mux.HandleFunc("/debug/metrics", m.handleDebugMetrics)
	mux.HandleFunc("/debug/config", m.handleDebugConfig)
	
	m.metricsServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", m.config.Metrics.Port),
		Handler: mux,
	}
	
	go func() {
		if err := m.metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			m.logger.Error("Metrics server failed: %v", err)
		}
	}()
	
	return nil
}

func (m *Manager) startHealthServer() error {
	mux := http.NewServeMux()
	mux.HandleFunc(m.config.Health.Path, m.health.HealthHandler)
	mux.HandleFunc(m.config.Health.Liveness, m.health.LivenessHandler)
	mux.HandleFunc(m.config.Health.Readiness, m.health.ReadinessHandler)
	
	// Add status endpoint
	mux.HandleFunc("/status", m.handleStatus)
	
	m.healthServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", m.config.Health.Port),
		Handler: mux,
	}
	
	go func() {
		if err := m.healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			m.logger.Error("Health server failed: %v", err)
		}
	}()
	
	return nil
}

func (m *Manager) startBackgroundTasks(ctx context.Context) {
	ticker := time.NewTicker(m.config.MetricsInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.collectApplicationMetrics()
		}
	}
}

func (m *Manager) collectApplicationMetrics() {
	// This would collect application-specific metrics
	// For now, we'll collect basic counts from the database
	
	if m.db == nil {
		return
	}
	
	var counts struct {
		Users        int64
		Tradings int64
		Transactions int64
		APIKeys      int64
	}
	
	// These table names would need to be adjusted based on your actual schema
	m.db.Table("users").Count(&counts.Users)
	m.db.Table("tradings").Count(&counts.Tradings)
	m.db.Table("transactions").Count(&counts.Transactions)
	m.db.Table("api_keys").Count(&counts.APIKeys)
	
	// Update metrics
	m.metrics.UpdateApplicationStats(
		int(counts.Users),
		int(counts.Tradings),
		int(counts.Transactions),
		int(counts.APIKeys),
		0, // sessions - would need to be tracked separately
	)
	
	// Update database connection metrics if available
	if sqlDB, err := m.db.DB(); err == nil {
		stats := sqlDB.Stats()
		m.metrics.UpdateDBConnectionStats(
			stats.OpenConnections,
			stats.Idle,
			stats.InUse,
			int(stats.WaitCount),
		)
	}
}

// HTTP Handlers

func (m *Manager) handleDebugMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	data := map[string]interface{}{
		"config": map[string]interface{}{
			"enabled":   m.config.Metrics.Enabled,
			"port":      m.config.Metrics.Port,
			"path":      m.config.Metrics.Path,
			"namespace": m.config.Metrics.Namespace,
		},
		"started": m.started,
	}
	
	json.NewEncoder(w).Encode(data)
}

func (m *Manager) handleDebugConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// Return sanitized config (without sensitive data)
	sanitizedConfig := *m.config
	
	// Remove sensitive receiver information
	for i := range sanitizedConfig.Alerting.Receivers {
		sanitizedConfig.Alerting.Receivers[i].URL = "[REDACTED]"
		sanitizedConfig.Alerting.Receivers[i].Headers = map[string]string{"[REDACTED]": "[REDACTED]"}
	}
	
	json.NewEncoder(w).Encode(sanitizedConfig)
}

func (m *Manager) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	status := map[string]interface{}{
		"service":     m.config.Service,
		"version":     m.config.Version,
		"environment": m.config.Environment,
		"started":     m.started,
		"components": map[string]bool{
			"metrics":  m.config.Metrics.Enabled,
			"logging":  true, // Always enabled
			"alerting": m.config.Alerting.Enabled,
			"health":   m.config.Health.Enabled,
		},
	}
	
	json.NewEncoder(w).Encode(status)
}

// IsStarted returns true if the monitoring system is started
func (m *Manager) IsStarted() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.started
}