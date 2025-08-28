package monitoring

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsCollector provides comprehensive application metrics
type MetricsCollector struct {
	// HTTP Metrics
	httpRequestsTotal     *prometheus.CounterVec
	httpRequestDuration   *prometheus.HistogramVec
	httpRequestSize       *prometheus.HistogramVec
	httpResponseSize      *prometheus.HistogramVec
	httpRequestsInFlight  *prometheus.GaugeVec

	// Database Metrics
	dbConnectionsOpen     prometheus.Gauge
	dbConnectionsIdle     prometheus.Gauge
	dbConnectionsInUse    prometheus.Gauge
	dbConnectionsWaiting  prometheus.Gauge
	dbQueryDuration       *prometheus.HistogramVec
	dbTransactionsTotal   *prometheus.CounterVec

	// Redis Metrics
	redisConnectionsOpen  prometheus.Gauge
	redisCommandsTotal    *prometheus.CounterVec
	redisCommandDuration  *prometheus.HistogramVec
	redisKeyspaceSize     *prometheus.GaugeVec

	// Application Metrics
	usersTotal            prometheus.Gauge
	tradingsTotal prometheus.Gauge
	transactionsTotal     prometheus.Gauge
	apiKeysTotal          prometheus.Gauge
	activeSessionsTotal   prometheus.Gauge

	// Security Metrics
	authAttemptsTotal     *prometheus.CounterVec
	rateLimitHitsTotal    *prometheus.CounterVec
	securityEventsTotal   *prometheus.CounterVec
	apiKeyUsageTotal      *prometheus.CounterVec

	// Business Metrics
	tradingVolumeTotal    *prometheus.CounterVec
	tradingFeesTotal      *prometheus.CounterVec
	accountBalances       *prometheus.GaugeVec
	tradingHealthStatus   *prometheus.GaugeVec

	// System Metrics
	goroutinesActive      prometheus.Gauge
	memoryUsage          prometheus.Gauge
	gcDuration           prometheus.Histogram
	uptime               prometheus.Gauge
}

// NewMetricsCollector creates a new metrics collector with all instruments
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		// HTTP Metrics
		httpRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status", "user_type"},
		),
		httpRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "Duration of HTTP requests in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path", "status"},
		),
		httpRequestSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_size_bytes",
				Help:    "Size of HTTP requests in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 6),
			},
			[]string{"method", "path"},
		),
		httpResponseSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_response_size_bytes",
				Help:    "Size of HTTP responses in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 6),
			},
			[]string{"method", "path", "status"},
		),
		httpRequestsInFlight: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "http_requests_in_flight",
				Help: "Number of HTTP requests currently being processed",
			},
			[]string{"method", "path"},
		),

		// Database Metrics
		dbConnectionsOpen: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "database_connections_open",
				Help: "Number of established connections both in use and idle",
			},
		),
		dbConnectionsIdle: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "database_connections_idle",
				Help: "Number of idle connections",
			},
		),
		dbConnectionsInUse: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "database_connections_in_use",
				Help: "Number of connections currently in use",
			},
		),
		dbConnectionsWaiting: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "database_connections_waiting",
				Help: "Number of connections waiting for a free connection",
			},
		),
		dbQueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "database_query_duration_seconds",
				Help:    "Duration of database queries in seconds",
				Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"operation", "table"},
		),
		dbTransactionsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "database_transactions_total",
				Help: "Total number of database transactions",
			},
			[]string{"operation", "result"},
		),

		// Redis Metrics
		redisConnectionsOpen: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "redis_connections_open",
				Help: "Number of Redis connections currently open",
			},
		),
		redisCommandsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "redis_commands_total",
				Help: "Total number of Redis commands executed",
			},
			[]string{"command", "result"},
		),
		redisCommandDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "redis_command_duration_seconds",
				Help:    "Duration of Redis commands in seconds",
				Buckets: []float64{.0001, .0005, .001, .005, .01, .025, .05, .1, .25, .5},
			},
			[]string{"command"},
		),
		redisKeyspaceSize: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "redis_keyspace_size",
				Help: "Number of keys in Redis keyspace",
			},
			[]string{"database"},
		),

		// Application Metrics
		usersTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "users_total",
				Help: "Total number of registered users",
			},
		),
		tradingsTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "tradings_total",
				Help: "Total number of configured tradings",
			},
		),
		transactionsTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "transactions_total",
				Help: "Total number of transactions processed",
			},
		),
		apiKeysTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "api_keys_total",
				Help: "Total number of active API keys",
			},
		),
		activeSessionsTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "active_sessions_total",
				Help: "Total number of active user sessions",
			},
		),

		// Security Metrics
		authAttemptsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "auth_attempts_total",
				Help: "Total number of authentication attempts",
			},
			[]string{"provider", "result"},
		),
		rateLimitHitsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "rate_limit_hits_total",
				Help: "Total number of rate limit violations",
			},
			[]string{"rule", "identifier_type"},
		),
		securityEventsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "security_events_total",
				Help: "Total number of security events",
			},
			[]string{"event_type", "severity"},
		),
		apiKeyUsageTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_key_usage_total",
				Help: "Total number of API key usages",
			},
			[]string{"key_type", "result"},
		),

		// Business Metrics
		tradingVolumeTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "trading_volume_total",
				Help: "Total trading volume processed",
			},
			[]string{"trading", "symbol", "direction"},
		),
		tradingFeesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "trading_fees_total",
				Help: "Total trading fees collected",
			},
			[]string{"trading", "fee_type"},
		),
		accountBalances: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "account_balances",
				Help: "Current account balances by symbol",
			},
			[]string{"trading", "account", "symbol"},
		),
		tradingHealthStatus: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "trading_health_status",
				Help: "Health status of trading connections (1=healthy, 0=unhealthy)",
			},
			[]string{"trading", "endpoint"},
		),

		// System Metrics
		goroutinesActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "goroutines_active",
				Help: "Number of active goroutines",
			},
		),
		memoryUsage: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "memory_usage_bytes",
				Help: "Current memory usage in bytes",
			},
		),
		gcDuration: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "gc_duration_seconds",
				Help:    "Duration of garbage collection cycles",
				Buckets: []float64{.0001, .0005, .001, .005, .01, .025, .05, .1, .25, .5},
			},
		),
		uptime: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "uptime_seconds",
				Help: "Application uptime in seconds",
			},
		),
	}
}

// HTTP Middleware for automatic metrics collection
func (mc *MetricsCollector) HTTPMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()
		method := c.Request.Method
		
		// Track request size
		if c.Request.ContentLength > 0 {
			mc.httpRequestSize.WithLabelValues(method, path).Observe(float64(c.Request.ContentLength))
		}

		// Track in-flight requests
		mc.httpRequestsInFlight.WithLabelValues(method, path).Inc()
		defer mc.httpRequestsInFlight.WithLabelValues(method, path).Dec()

		// Process request
		c.Next()

		// Collect metrics
		status := strconv.Itoa(c.Writer.Status())
		duration := time.Since(start).Seconds()
		
		// Determine user type
		userType := "anonymous"
		if _, exists := c.Get("user_id"); exists {
			userType = "authenticated"
		} else if _, exists := c.Get("api_key"); exists {
			userType = "api_key"
		}

		// Record metrics
		mc.httpRequestsTotal.WithLabelValues(method, path, status, userType).Inc()
		mc.httpRequestDuration.WithLabelValues(method, path, status).Observe(duration)
		mc.httpResponseSize.WithLabelValues(method, path, status).Observe(float64(c.Writer.Size()))
	}
}

// Database metrics methods
func (mc *MetricsCollector) RecordDBQuery(operation, table string, duration time.Duration, err error) {
	result := "success"
	if err != nil {
		result = "error"
	}
	
	mc.dbQueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
	mc.dbTransactionsTotal.WithLabelValues(operation, result).Inc()
}

func (mc *MetricsCollector) UpdateDBConnectionStats(open, idle, inUse, waiting int) {
	mc.dbConnectionsOpen.Set(float64(open))
	mc.dbConnectionsIdle.Set(float64(idle))
	mc.dbConnectionsInUse.Set(float64(inUse))
	mc.dbConnectionsWaiting.Set(float64(waiting))
}

// Redis metrics methods
func (mc *MetricsCollector) RecordRedisCommand(command string, duration time.Duration, err error) {
	result := "success"
	if err != nil {
		result = "error"
	}
	
	mc.redisCommandsTotal.WithLabelValues(command, result).Inc()
	mc.redisCommandDuration.WithLabelValues(command).Observe(duration.Seconds())
}

func (mc *MetricsCollector) UpdateRedisStats(connections int, keyspaceSize map[string]int) {
	mc.redisConnectionsOpen.Set(float64(connections))
	
	for db, size := range keyspaceSize {
		mc.redisKeyspaceSize.WithLabelValues(db).Set(float64(size))
	}
}

// Application metrics methods
func (mc *MetricsCollector) UpdateApplicationStats(users, tradings, transactions, apiKeys, sessions int) {
	mc.usersTotal.Set(float64(users))
	mc.tradingsTotal.Set(float64(tradings))
	mc.transactionsTotal.Set(float64(transactions))
	mc.apiKeysTotal.Set(float64(apiKeys))
	mc.activeSessionsTotal.Set(float64(sessions))
}

// Security metrics methods
func (mc *MetricsCollector) RecordAuthAttempt(provider, result string) {
	mc.authAttemptsTotal.WithLabelValues(provider, result).Inc()
}

func (mc *MetricsCollector) RecordRateLimitHit(rule, identifierType string) {
	mc.rateLimitHitsTotal.WithLabelValues(rule, identifierType).Inc()
}

func (mc *MetricsCollector) RecordSecurityEvent(eventType, severity string) {
	mc.securityEventsTotal.WithLabelValues(eventType, severity).Inc()
}

func (mc *MetricsCollector) RecordAPIKeyUsage(keyType, result string) {
	mc.apiKeyUsageTotal.WithLabelValues(keyType, result).Inc()
}

// Business metrics methods
func (mc *MetricsCollector) RecordTradingVolume(trading, symbol, direction string, volume float64) {
	mc.tradingVolumeTotal.WithLabelValues(trading, symbol, direction).Add(volume)
}

func (mc *MetricsCollector) RecordTradingFee(trading, feeType string, fee float64) {
	mc.tradingFeesTotal.WithLabelValues(trading, feeType).Add(fee)
}

func (mc *MetricsCollector) UpdateAccountBalance(trading, account, symbol string, balance float64) {
	mc.accountBalances.WithLabelValues(trading, account, symbol).Set(balance)
}

func (mc *MetricsCollector) UpdateTradingHealth(trading, endpoint string, healthy bool) {
	status := 0.0
	if healthy {
		status = 1.0
	}
	mc.tradingHealthStatus.WithLabelValues(trading, endpoint).Set(status)
}

// System metrics methods
func (mc *MetricsCollector) UpdateSystemStats(goroutines int, memoryBytes uint64, uptime time.Duration) {
	mc.goroutinesActive.Set(float64(goroutines))
	mc.memoryUsage.Set(float64(memoryBytes))
	mc.uptime.Set(uptime.Seconds())
}

func (mc *MetricsCollector) RecordGCDuration(duration time.Duration) {
	mc.gcDuration.Observe(duration.Seconds())
}

// GetMetricsHandler returns the Prometheus metrics HTTP handler
func (mc *MetricsCollector) GetMetricsHandler() http.Handler {
	return promhttp.Handler()
}

// StartMetricsCollection starts background collection of system metrics
func (mc *MetricsCollector) StartMetricsCollection(ctx context.Context) {
	go mc.collectSystemMetrics(ctx)
}

func (mc *MetricsCollector) collectSystemMetrics(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	
	startTime := time.Now()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// This would collect actual system metrics
			// For now, we'll just update uptime
			mc.uptime.Set(time.Since(startTime).Seconds())
		}
	}
}