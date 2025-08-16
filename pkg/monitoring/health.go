package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	StatusHealthy   HealthStatus = "healthy"
	StatusDegraded  HealthStatus = "degraded"
	StatusUnhealthy HealthStatus = "unhealthy"
	StatusUnknown   HealthStatus = "unknown"
)

// HealthCheck represents a single health check
type HealthCheck struct {
	Name        string                 `json:"name"`
	Status      HealthStatus           `json:"status"`
	Message     string                 `json:"message,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Duration    time.Duration          `json:"duration"`
	LastChecked time.Time              `json:"last_checked"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Critical    bool                   `json:"critical"`
}

// HealthReport represents the overall health of the system
type HealthReport struct {
	Status      HealthStatus             `json:"status"`
	Timestamp   time.Time                `json:"timestamp"`
	Version     string                   `json:"version,omitempty"`
	Service     string                   `json:"service,omitempty"`
	Uptime      time.Duration            `json:"uptime"`
	Checks      map[string]*HealthCheck  `json:"checks"`
	Summary     map[string]int           `json:"summary"`
	Details     map[string]interface{}   `json:"details,omitempty"`
}

// HealthChecker defines a health check function
type HealthChecker interface {
	Check(ctx context.Context) *HealthCheck
	Name() string
	IsCritical() bool
}

// HealthMonitor manages and executes health checks
type HealthMonitor struct {
	checkers    []HealthChecker
	lastReport  *HealthReport
	mu          sync.RWMutex
	startTime   time.Time
	service     string
	version     string
	interval    time.Duration
	timeout     time.Duration
	logger      *Logger
	alertMgr    *AlertManager
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(service, version string, logger *Logger, alertMgr *AlertManager) *HealthMonitor {
	return &HealthMonitor{
		checkers:    make([]HealthChecker, 0),
		startTime:   time.Now(),
		service:     service,
		version:     version,
		interval:    30 * time.Second,
		timeout:     10 * time.Second,
		logger:      logger,
		alertMgr:    alertMgr,
	}
}

// AddChecker adds a health checker
func (hm *HealthMonitor) AddChecker(checker HealthChecker) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.checkers = append(hm.checkers, checker)
}

// SetInterval sets the health check interval
func (hm *HealthMonitor) SetInterval(interval time.Duration) {
	hm.interval = interval
}

// SetTimeout sets the health check timeout
func (hm *HealthMonitor) SetTimeout(timeout time.Duration) {
	hm.timeout = timeout
}

// StartMonitoring starts the health monitoring loop
func (hm *HealthMonitor) StartMonitoring(ctx context.Context) {
	ticker := time.NewTicker(hm.interval)
	defer ticker.Stop()
	
	// Initial health check
	hm.RunHealthChecks(ctx)
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hm.RunHealthChecks(ctx)
		}
	}
}

// RunHealthChecks executes all health checks
func (hm *HealthMonitor) RunHealthChecks(ctx context.Context) *HealthReport {
	checkCtx, cancel := context.WithTimeout(ctx, hm.timeout)
	defer cancel()
	
	report := &HealthReport{
		Timestamp: time.Now().UTC(),
		Service:   hm.service,
		Version:   hm.version,
		Uptime:    time.Since(hm.startTime),
		Checks:    make(map[string]*HealthCheck),
		Summary:   make(map[string]int),
	}
	
	hm.mu.RLock()
	checkers := make([]HealthChecker, len(hm.checkers))
	copy(checkers, hm.checkers)
	hm.mu.RUnlock()
	
	// Run all checks concurrently
	var wg sync.WaitGroup
	checkResults := make(chan *HealthCheck, len(checkers))
	
	for _, checker := range checkers {
		wg.Add(1)
		go func(c HealthChecker) {
			defer wg.Done()
			check := c.Check(checkCtx)
			checkResults <- check
		}(checker)
	}
	
	// Wait for all checks to complete
	wg.Wait()
	close(checkResults)
	
	// Collect results
	overallStatus := StatusHealthy
	for check := range checkResults {
		report.Checks[check.Name] = check
		report.Summary[string(check.Status)]++
		
		// Determine overall status
		if check.Critical && check.Status == StatusUnhealthy {
			overallStatus = StatusUnhealthy
		} else if overallStatus == StatusHealthy && check.Status == StatusDegraded {
			overallStatus = StatusDegraded
		}
		
		// Fire alerts for unhealthy critical components
		if check.Critical && check.Status == StatusUnhealthy && hm.alertMgr != nil {
			hm.alertMgr.FireAlert(
				fmt.Sprintf("Health Check Failed: %s", check.Name),
				fmt.Sprintf("Critical health check '%s' is unhealthy: %s", check.Name, check.Message),
				SeverityCritical,
				"health",
				map[string]interface{}{
					"check_name": check.Name,
					"error":      check.Error,
					"duration":   check.Duration.String(),
				},
			)
		}
		
		// Log health check results
		if hm.logger != nil {
			if check.Status == StatusUnhealthy {
				hm.logger.Error("Health check failed: %s - %s", check.Name, check.Message)
			} else if check.Status == StatusDegraded {
				hm.logger.Warn("Health check degraded: %s - %s", check.Name, check.Message)
			} else {
				hm.logger.Debug("Health check passed: %s", check.Name)
			}
		}
	}
	
	report.Status = overallStatus
	
	hm.mu.Lock()
	hm.lastReport = report
	hm.mu.Unlock()
	
	return report
}

// GetLastReport returns the last health report
func (hm *HealthMonitor) GetLastReport() *HealthReport {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	return hm.lastReport
}

// IsHealthy returns true if the system is healthy
func (hm *HealthMonitor) IsHealthy() bool {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	
	if hm.lastReport == nil {
		return false
	}
	
	return hm.lastReport.Status == StatusHealthy || hm.lastReport.Status == StatusDegraded
}

// HTTP Handlers
func (hm *HealthMonitor) LivenessHandler(w http.ResponseWriter, r *http.Request) {
	// Liveness probe should only check if the application is running
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"alive","timestamp":"` + time.Now().UTC().Format(time.RFC3339) + `"}`))
}

func (hm *HealthMonitor) ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	report := hm.GetLastReport()
	
	w.Header().Set("Content-Type", "application/json")
	
	if report == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status":"unknown","message":"No health checks completed"}`))
		return
	}
	
	// Check if any critical components are unhealthy
	ready := true
	for _, check := range report.Checks {
		if check.Critical && check.Status == StatusUnhealthy {
			ready = false
			break
		}
	}
	
	if ready {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	
	responseBytes, _ := json.Marshal(map[string]interface{}{
		"status":    report.Status,
		"timestamp": report.Timestamp,
		"ready":     ready,
		"checks":    report.Checks,
	})
	w.Write(responseBytes)
}

func (hm *HealthMonitor) HealthHandler(w http.ResponseWriter, r *http.Request) {
	report := hm.GetLastReport()
	
	w.Header().Set("Content-Type", "application/json")
	
	if report == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status":"unknown","message":"No health checks completed"}`))
		return
	}
	
	if report.Status == StatusHealthy || report.Status == StatusDegraded {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	
	responseBytes, _ := json.Marshal(report)
	w.Write(responseBytes)
}

// Built-in health checkers

// DatabaseHealthChecker checks database connectivity
type DatabaseHealthChecker struct {
	name     string
	db       *gorm.DB
	critical bool
}

func NewDatabaseHealthChecker(name string, db *gorm.DB, critical bool) *DatabaseHealthChecker {
	return &DatabaseHealthChecker{
		name:     name,
		db:       db,
		critical: critical,
	}
}

func (dhc *DatabaseHealthChecker) Name() string {
	return dhc.name
}

func (dhc *DatabaseHealthChecker) IsCritical() bool {
	return dhc.critical
}

func (dhc *DatabaseHealthChecker) Check(ctx context.Context) *HealthCheck {
	start := time.Now()
	check := &HealthCheck{
		Name:        dhc.name,
		Critical:    dhc.critical,
		LastChecked: start,
	}
	
	// Test basic connectivity
	sqlDB, err := dhc.db.DB()
	if err != nil {
		check.Status = StatusUnhealthy
		check.Error = err.Error()
		check.Message = "Failed to get database connection"
		check.Duration = time.Since(start)
		return check
	}
	
	// Test ping
	err = sqlDB.PingContext(ctx)
	if err != nil {
		check.Status = StatusUnhealthy
		check.Error = err.Error()
		check.Message = "Database ping failed"
		check.Duration = time.Since(start)
		return check
	}
	
	// Get connection stats
	stats := sqlDB.Stats()
	check.Details = map[string]interface{}{
		"open_connections": stats.OpenConnections,
		"in_use":          stats.InUse,
		"idle":            stats.Idle,
		"wait_count":      stats.WaitCount,
		"wait_duration":   stats.WaitDuration.String(),
		"max_idle_closed": stats.MaxIdleClosed,
		"max_lifetime_closed": stats.MaxLifetimeClosed,
	}
	
	// Determine status based on connection usage
	maxConnections := stats.MaxOpenConnections
	if maxConnections == 0 {
		maxConnections = 10 // Default assumption
	}
	
	usagePercent := float64(stats.OpenConnections) / float64(maxConnections) * 100
	
	if usagePercent > 90 {
		check.Status = StatusDegraded
		check.Message = fmt.Sprintf("High connection usage: %.1f%%", usagePercent)
	} else if usagePercent > 80 {
		check.Status = StatusDegraded
		check.Message = fmt.Sprintf("Elevated connection usage: %.1f%%", usagePercent)
	} else {
		check.Status = StatusHealthy
		check.Message = "Database is healthy"
	}
	
	check.Duration = time.Since(start)
	return check
}

// RedisHealthChecker checks Redis connectivity
type RedisHealthChecker struct {
	name     string
	client   *redis.Client
	critical bool
}

func NewRedisHealthChecker(name string, client *redis.Client, critical bool) *RedisHealthChecker {
	return &RedisHealthChecker{
		name:     name,
		client:   client,
		critical: critical,
	}
}

func (rhc *RedisHealthChecker) Name() string {
	return rhc.name
}

func (rhc *RedisHealthChecker) IsCritical() bool {
	return rhc.critical
}

func (rhc *RedisHealthChecker) Check(ctx context.Context) *HealthCheck {
	start := time.Now()
	check := &HealthCheck{
		Name:        rhc.name,
		Critical:    rhc.critical,
		LastChecked: start,
	}
	
	// Test ping
	pong, err := rhc.client.Ping(ctx).Result()
	if err != nil {
		check.Status = StatusUnhealthy
		check.Error = err.Error()
		check.Message = "Redis ping failed"
		check.Duration = time.Since(start)
		return check
	}
	
	if pong != "PONG" {
		check.Status = StatusUnhealthy
		check.Message = fmt.Sprintf("Unexpected ping response: %s", pong)
		check.Duration = time.Since(start)
		return check
	}
	
	// Get Redis info
	info, err := rhc.client.Info(ctx, "memory", "clients", "keyspace").Result()
	if err != nil {
		check.Status = StatusDegraded
		check.Message = "Could not retrieve Redis info"
		check.Error = err.Error()
	} else {
		check.Details = map[string]interface{}{
			"info": info,
		}
		check.Status = StatusHealthy
		check.Message = "Redis is healthy"
	}
	
	check.Duration = time.Since(start)
	return check
}

// HTTPHealthChecker checks external HTTP dependencies
type HTTPHealthChecker struct {
	name     string
	url      string
	critical bool
	client   *http.Client
}

func NewHTTPHealthChecker(name, url string, critical bool, timeout time.Duration) *HTTPHealthChecker {
	return &HTTPHealthChecker{
		name:     name,
		url:      url,
		critical: critical,
		client:   &http.Client{Timeout: timeout},
	}
}

func (hhc *HTTPHealthChecker) Name() string {
	return hhc.name
}

func (hhc *HTTPHealthChecker) IsCritical() bool {
	return hhc.critical
}

func (hhc *HTTPHealthChecker) Check(ctx context.Context) *HealthCheck {
	start := time.Now()
	check := &HealthCheck{
		Name:        hhc.name,
		Critical:    hhc.critical,
		LastChecked: start,
	}
	
	req, err := http.NewRequestWithContext(ctx, "GET", hhc.url, nil)
	if err != nil {
		check.Status = StatusUnhealthy
		check.Error = err.Error()
		check.Message = "Failed to create request"
		check.Duration = time.Since(start)
		return check
	}
	
	resp, err := hhc.client.Do(req)
	if err != nil {
		check.Status = StatusUnhealthy
		check.Error = err.Error()
		check.Message = "HTTP request failed"
		check.Duration = time.Since(start)
		return check
	}
	defer resp.Body.Close()
	
	check.Details = map[string]interface{}{
		"status_code": resp.StatusCode,
		"url":         hhc.url,
	}
	
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		check.Status = StatusHealthy
		check.Message = "HTTP endpoint is healthy"
	} else if resp.StatusCode >= 500 {
		check.Status = StatusUnhealthy
		check.Message = fmt.Sprintf("HTTP endpoint returned server error: %d", resp.StatusCode)
	} else {
		check.Status = StatusDegraded
		check.Message = fmt.Sprintf("HTTP endpoint returned client error: %d", resp.StatusCode)
	}
	
	check.Duration = time.Since(start)
	return check
}

// MemoryHealthChecker checks system memory usage
type MemoryHealthChecker struct {
	name            string
	warningPercent  float64
	criticalPercent float64
	critical        bool
}

func NewMemoryHealthChecker(name string, warningPercent, criticalPercent float64, critical bool) *MemoryHealthChecker {
	return &MemoryHealthChecker{
		name:            name,
		warningPercent:  warningPercent,
		criticalPercent: criticalPercent,
		critical:        critical,
	}
}

func (mhc *MemoryHealthChecker) Name() string {
	return mhc.name
}

func (mhc *MemoryHealthChecker) IsCritical() bool {
	return mhc.critical
}

func (mhc *MemoryHealthChecker) Check(ctx context.Context) *HealthCheck {
	start := time.Now()
	check := &HealthCheck{
		Name:        mhc.name,
		Critical:    mhc.critical,
		LastChecked: start,
	}
	
	// This is a simplified memory check
	// In production, you would use actual system calls or libraries
	// to get real memory usage statistics
	
	// Placeholder implementation
	memoryUsagePercent := 45.0 // Would be calculated from actual system stats
	
	check.Details = map[string]interface{}{
		"usage_percent": memoryUsagePercent,
		"warning_threshold": mhc.warningPercent,
		"critical_threshold": mhc.criticalPercent,
	}
	
	if memoryUsagePercent >= mhc.criticalPercent {
		check.Status = StatusUnhealthy
		check.Message = fmt.Sprintf("Memory usage critical: %.1f%%", memoryUsagePercent)
	} else if memoryUsagePercent >= mhc.warningPercent {
		check.Status = StatusDegraded
		check.Message = fmt.Sprintf("Memory usage elevated: %.1f%%", memoryUsagePercent)
	} else {
		check.Status = StatusHealthy
		check.Message = fmt.Sprintf("Memory usage normal: %.1f%%", memoryUsagePercent)
	}
	
	check.Duration = time.Since(start)
	return check
}