package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all the Prometheus metrics
type Metrics struct {
	// HTTP metrics
	HTTPRequestsTotal    *prometheus.CounterVec
	HTTPRequestDuration  *prometheus.HistogramVec
	HTTPRequestsInFlight prometheus.Gauge

	// Database metrics
	DatabaseConnections       prometheus.Gauge
	DatabaseQueryDuration     *prometheus.HistogramVec
	DatabaseQueriesTotal      *prometheus.CounterVec
	DatabaseConnectionsErrors *prometheus.CounterVec

	// NATS metrics
	NATSMessagesPublished  *prometheus.CounterVec
	NATSMessagesConsumed   *prometheus.CounterVec
	NATSConnectionStatus   prometheus.Gauge
	NATSMessageProcessTime *prometheus.HistogramVec

	// Business metrics
	UsersTotal          prometheus.Gauge
	TradingsTotal       prometheus.Gauge
	SubAccountsTotal    prometheus.Gauge
	TransactionsTotal   *prometheus.CounterVec
	TradingLogsTotal    *prometheus.CounterVec
	BalanceUpdatesTotal *prometheus.CounterVec

	// Auth metrics
	AuthRequestsTotal  *prometheus.CounterVec
	ActiveSessions     prometheus.Gauge
	TokenRefreshTotal  *prometheus.CounterVec
	OAuthRequestsTotal *prometheus.CounterVec

	// Error metrics
	ErrorsTotal        *prometheus.CounterVec
	PanicRecoveryTotal *prometheus.CounterVec
}

// NewMetrics creates and registers all Prometheus metrics
func NewMetrics() *Metrics {
	return &Metrics{
		// HTTP metrics
		HTTPRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status_code"},
		),
		HTTPRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "endpoint", "status_code"},
		),
		HTTPRequestsInFlight: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "http_requests_in_flight",
				Help: "Number of HTTP requests currently being served",
			},
		),

		// Database metrics
		DatabaseConnections: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "database_connections_active",
				Help: "Number of active database connections",
			},
		),
		DatabaseQueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "database_query_duration_seconds",
				Help:    "Database query duration in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0},
			},
			[]string{"operation", "table"},
		),
		DatabaseQueriesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "database_queries_total",
				Help: "Total number of database queries",
			},
			[]string{"operation", "table", "status"},
		),
		DatabaseConnectionsErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "database_connection_errors_total",
				Help: "Total number of database connection errors",
			},
			[]string{"error_type"},
		),

		// NATS metrics
		NATSMessagesPublished: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nats_messages_published_total",
				Help: "Total number of NATS messages published",
			},
			[]string{"subject", "status"},
		),
		NATSMessagesConsumed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nats_messages_consumed_total",
				Help: "Total number of NATS messages consumed",
			},
			[]string{"subject", "consumer", "status"},
		),
		NATSConnectionStatus: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "nats_connection_status",
				Help: "NATS connection status (1=connected, 0=disconnected)",
			},
		),
		NATSMessageProcessTime: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "nats_message_process_duration_seconds",
				Help:    "NATS message processing duration in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0},
			},
			[]string{"subject", "consumer"},
		),

		// Business metrics
		UsersTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "users_total",
				Help: "Total number of registered users",
			},
		),
		TradingsTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "tradings_total",
				Help: "Total number of configured trading platforms",
			},
		),
		SubAccountsTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "sub_accounts_total",
				Help: "Total number of sub-accounts",
			},
		),
		TransactionsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "transactions_total",
				Help: "Total number of transactions",
			},
			[]string{"direction", "reason"},
		),
		TradingLogsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "trading_logs_total",
				Help: "Total number of trading log entries",
			},
			[]string{"type", "source"},
		),
		BalanceUpdatesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "balance_updates_total",
				Help: "Total number of balance updates",
			},
			[]string{"direction", "status"},
		),

		// Auth metrics
		AuthRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "auth_requests_total",
				Help: "Total number of authentication requests",
			},
			[]string{"method", "provider", "status"},
		),
		ActiveSessions: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "active_sessions",
				Help: "Number of active user sessions",
			},
		),
		TokenRefreshTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "token_refresh_total",
				Help: "Total number of token refresh attempts",
			},
			[]string{"status"},
		),
		OAuthRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "oauth_requests_total",
				Help: "Total number of OAuth requests",
			},
			[]string{"provider", "action", "status"},
		),

		// Error metrics
		ErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "errors_total",
				Help: "Total number of errors",
			},
			[]string{"component", "error_type"},
		),
		PanicRecoveryTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "panic_recovery_total",
				Help: "Total number of recovered panics",
			},
			[]string{"component"},
		),
	}
}

// HTTPMetricsMiddleware returns a Gin middleware for collecting HTTP metrics
func (m *Metrics) HTTPMetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Increment in-flight requests
		m.HTTPRequestsInFlight.Inc()
		defer m.HTTPRequestsInFlight.Dec()

		// Record start time
		start := time.Now()

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start).Seconds()

		// Get labels
		method := c.Request.Method
		endpoint := c.FullPath()
		if endpoint == "" {
			endpoint = "unknown"
		}
		statusCode := strconv.Itoa(c.Writer.Status())

		// Record metrics
		m.HTTPRequestsTotal.WithLabelValues(method, endpoint, statusCode).Inc()
		m.HTTPRequestDuration.WithLabelValues(method, endpoint, statusCode).Observe(duration)
	}
}

// RecordDatabaseQuery records database query metrics
func (m *Metrics) RecordDatabaseQuery(operation, table string, duration time.Duration, err error) {
	labels := []string{operation, table}

	// Record duration
	m.DatabaseQueryDuration.WithLabelValues(labels...).Observe(duration.Seconds())

	// Record query count with status
	status := "success"
	if err != nil {
		status = "error"
	}
	m.DatabaseQueriesTotal.WithLabelValues(operation, table, status).Inc()
}

// RecordNATSMessage records NATS message metrics
func (m *Metrics) RecordNATSMessage(subject, consumer string, duration time.Duration, published bool, err error) {
	status := "success"
	if err != nil {
		status = "error"
	}

	if published {
		m.NATSMessagesPublished.WithLabelValues(subject, status).Inc()
	} else {
		m.NATSMessagesConsumed.WithLabelValues(subject, consumer, status).Inc()
		m.NATSMessageProcessTime.WithLabelValues(subject, consumer).Observe(duration.Seconds())
	}
}

// SetNATSConnectionStatus sets the NATS connection status
func (m *Metrics) SetNATSConnectionStatus(connected bool) {
	if connected {
		m.NATSConnectionStatus.Set(1)
	} else {
		m.NATSConnectionStatus.Set(0)
	}
}

// RecordAuthRequest records authentication request metrics
func (m *Metrics) RecordAuthRequest(method, provider, status string) {
	m.AuthRequestsTotal.WithLabelValues(method, provider, status).Inc()
}

// RecordTokenRefresh records token refresh metrics
func (m *Metrics) RecordTokenRefresh(status string) {
	m.TokenRefreshTotal.WithLabelValues(status).Inc()
}

// RecordOAuthRequest records OAuth request metrics
func (m *Metrics) RecordOAuthRequest(provider, action, status string) {
	m.OAuthRequestsTotal.WithLabelValues(provider, action, status).Inc()
}

// RecordTransaction records transaction metrics
func (m *Metrics) RecordTransaction(direction, reason string) {
	m.TransactionsTotal.WithLabelValues(direction, reason).Inc()
}

// RecordTradingLog records trading log metrics
func (m *Metrics) RecordTradingLog(logType, source string) {
	m.TradingLogsTotal.WithLabelValues(logType, source).Inc()
}

// RecordBalanceUpdate records balance update metrics
func (m *Metrics) RecordBalanceUpdate(direction, status string) {
	m.BalanceUpdatesTotal.WithLabelValues(direction, status).Inc()
}

// RecordError records error metrics
func (m *Metrics) RecordError(component, errorType string) {
	m.ErrorsTotal.WithLabelValues(component, errorType).Inc()
}

// RecordPanicRecovery records panic recovery metrics
func (m *Metrics) RecordPanicRecovery(component string) {
	m.PanicRecoveryTotal.WithLabelValues(component).Inc()
}

// UpdateBusinessMetrics updates business-related gauge metrics
func (m *Metrics) UpdateBusinessMetrics(users, tradingPlatforms, subAccounts int64) {
	m.UsersTotal.Set(float64(users))
	m.TradingsTotal.Set(float64(tradingPlatforms))
	m.SubAccountsTotal.Set(float64(subAccounts))
}

// SetDatabaseConnections sets the number of active database connections
func (m *Metrics) SetDatabaseConnections(count int) {
	m.DatabaseConnections.Set(float64(count))
}

// SetActiveSessions sets the number of active sessions
func (m *Metrics) SetActiveSessions(count int) {
	m.ActiveSessions.Set(float64(count))
}

// RecordDatabaseConnectionError records database connection errors
func (m *Metrics) RecordDatabaseConnectionError(errorType string) {
	m.DatabaseConnectionsErrors.WithLabelValues(errorType).Inc()
}
