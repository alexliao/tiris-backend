package monitoring

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// MonitoringConfig holds configuration for the monitoring system
type MonitoringConfig struct {
	// General settings
	Enabled         bool          `json:"enabled"`
	Service         string        `json:"service"`
	Version         string        `json:"version"`
	Environment     string        `json:"environment"`
	MetricsInterval time.Duration `json:"metrics_interval"`

	// Metrics settings
	Metrics MetricsConfig `json:"metrics"`

	// Logging settings
	Logging LoggerConfig `json:"logging"`

	// Alerting settings
	Alerting AlertingConfig `json:"alerting"`

	// Health checking settings
	Health HealthConfig `json:"health"`
}

// MetricsConfig configures Prometheus metrics collection
type MetricsConfig struct {
	Enabled    bool   `json:"enabled"`
	Port       int    `json:"port"`
	Path       string `json:"path"`
	Namespace  string `json:"namespace"`
	Subsystem  string `json:"subsystem"`
}

// AlertingConfig configures the alerting system
type AlertingConfig struct {
	Enabled   bool               `json:"enabled"`
	Receivers []ReceiverConfig   `json:"receivers"`
	Rules     []AlertRuleConfig  `json:"rules"`
}

// ReceiverConfig configures alert receivers
type ReceiverConfig struct {
	Name     string                 `json:"name"`
	Type     string                 `json:"type"` // "webhook", "slack"
	URL      string                 `json:"url"`
	Channel  string                 `json:"channel,omitempty"`
	Headers  map[string]string      `json:"headers,omitempty"`
	Settings map[string]interface{} `json:"settings,omitempty"`
}

// AlertRuleConfig configures alert rules
type AlertRuleConfig struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Severity    string                 `json:"severity"`
	Component   string                 `json:"component"`
	Condition   AlertConditionConfig   `json:"condition"`
	Duration    string                 `json:"duration"`
	Cooldown    string                 `json:"cooldown"`
	Enabled     bool                   `json:"enabled"`
	Labels      map[string]string      `json:"labels,omitempty"`
	Annotations map[string]string      `json:"annotations,omitempty"`
}

// AlertConditionConfig configures alert conditions
type AlertConditionConfig struct {
	Type      string      `json:"type"`
	Metric    string      `json:"metric"`
	Operator  string      `json:"operator"`
	Threshold interface{} `json:"threshold"`
	Window    string      `json:"window,omitempty"`
}

// HealthConfig configures health checking
type HealthConfig struct {
	Enabled     bool          `json:"enabled"`
	Port        int           `json:"port"`
	Path        string        `json:"path"`
	Interval    time.Duration `json:"interval"`
	Timeout     time.Duration `json:"timeout"`
	Liveness    string        `json:"liveness"`
	Readiness   string        `json:"readiness"`
}

// LoadMonitoringConfig loads monitoring configuration from environment and defaults
func LoadMonitoringConfig() *MonitoringConfig {
	config := &MonitoringConfig{
		Enabled:         getBoolEnv("MONITORING_ENABLED", true),
		Service:         getEnv("SERVICE_NAME", "tiris-backend"),
		Version:         getEnv("SERVICE_VERSION", "dev"),
		Environment:     getEnv("ENVIRONMENT", "development"),
		MetricsInterval: getDurationEnv("METRICS_INTERVAL", 15*time.Second),

		Metrics: MetricsConfig{
			Enabled:   getBoolEnv("METRICS_ENABLED", true),
			Port:      getIntEnv("METRICS_PORT", 9090),
			Path:      getEnv("METRICS_PATH", "/metrics"),
			Namespace: getEnv("METRICS_NAMESPACE", "tiris"),
			Subsystem: getEnv("METRICS_SUBSYSTEM", "backend"),
		},

		Logging: LoggerConfig{
			Level:   getEnv("LOG_LEVEL", "info"),
			Format:  getEnv("LOG_FORMAT", "json"),
			Service: getEnv("SERVICE_NAME", "tiris-backend"),
			Version: getEnv("SERVICE_VERSION", "dev"),
			Outputs: []string{"stdout"},
			Fields: map[string]interface{}{
				"environment": getEnv("ENVIRONMENT", "development"),
				"datacenter":  getEnv("DATACENTER", "local"),
			},
		},

		Alerting: AlertingConfig{
			Enabled:   getBoolEnv("ALERTING_ENABLED", true),
			Receivers: loadAlertReceivers(),
			Rules:     loadDefaultAlertRules(),
		},

		Health: HealthConfig{
			Enabled:   getBoolEnv("HEALTH_ENABLED", true),
			Port:      getIntEnv("HEALTH_PORT", 8081),
			Path:      getEnv("HEALTH_PATH", "/health"),
			Interval:  getDurationEnv("HEALTH_INTERVAL", 30*time.Second),
			Timeout:   getDurationEnv("HEALTH_TIMEOUT", 10*time.Second),
			Liveness:  getEnv("HEALTH_LIVENESS_PATH", "/live"),
			Readiness: getEnv("HEALTH_READINESS_PATH", "/ready"),
		},
	}

	return config
}

// loadAlertReceivers loads alert receivers from environment
func loadAlertReceivers() []ReceiverConfig {
	var receivers []ReceiverConfig

	// Webhook receiver
	if webhookURL := getEnv("ALERT_WEBHOOK_URL", ""); webhookURL != "" {
		receivers = append(receivers, ReceiverConfig{
			Name: "webhook",
			Type: "webhook",
			URL:  webhookURL,
			Headers: map[string]string{
				"Authorization": getEnv("ALERT_WEBHOOK_TOKEN", ""),
			},
		})
	}

	// Slack receiver
	if slackURL := getEnv("ALERT_SLACK_WEBHOOK_URL", ""); slackURL != "" {
		receivers = append(receivers, ReceiverConfig{
			Name:    "slack",
			Type:    "slack",
			URL:     slackURL,
			Channel: getEnv("ALERT_SLACK_CHANNEL", "#alerts"),
		})
	}

	return receivers
}

// loadDefaultAlertRules loads default alert rules with environment overrides
func loadDefaultAlertRules() []AlertRuleConfig {
	return []AlertRuleConfig{
		{
			Name:        "HighErrorRate",
			Description: "HTTP error rate is above threshold",
			Severity:    getEnv("ALERT_ERROR_RATE_SEVERITY", "warning"),
			Component:   "http",
			Condition: AlertConditionConfig{
				Type:      "threshold",
				Metric:    "http_error_rate",
				Operator:  "gt",
				Threshold: getFloatEnv("ALERT_ERROR_RATE_THRESHOLD", 5.0),
			},
			Duration: getEnv("ALERT_ERROR_RATE_DURATION", "5m"),
			Cooldown: getEnv("ALERT_ERROR_RATE_COOLDOWN", "15m"),
			Enabled:  getBoolEnv("ALERT_ERROR_RATE_ENABLED", true),
		},
		{
			Name:        "HighResponseTime",
			Description: "Average response time is above threshold",
			Severity:    getEnv("ALERT_RESPONSE_TIME_SEVERITY", "critical"),
			Component:   "http",
			Condition: AlertConditionConfig{
				Type:      "threshold",
				Metric:    "http_response_time_avg",
				Operator:  "gt",
				Threshold: getFloatEnv("ALERT_RESPONSE_TIME_THRESHOLD", 1000.0),
			},
			Duration: getEnv("ALERT_RESPONSE_TIME_DURATION", "2m"),
			Cooldown: getEnv("ALERT_RESPONSE_TIME_COOLDOWN", "10m"),
			Enabled:  getBoolEnv("ALERT_RESPONSE_TIME_ENABLED", true),
		},
		{
			Name:        "DatabaseConnectionsHigh",
			Description: "Database connection usage is above threshold",
			Severity:    getEnv("ALERT_DB_CONN_SEVERITY", "warning"),
			Component:   "database",
			Condition: AlertConditionConfig{
				Type:      "threshold",
				Metric:    "database_connections_usage_percent",
				Operator:  "gt",
				Threshold: getFloatEnv("ALERT_DB_CONN_THRESHOLD", 80.0),
			},
			Duration: getEnv("ALERT_DB_CONN_DURATION", "5m"),
			Cooldown: getEnv("ALERT_DB_CONN_COOLDOWN", "15m"),
			Enabled:  getBoolEnv("ALERT_DB_CONN_ENABLED", true),
		},
		{
			Name:        "MemoryUsageHigh",
			Description: "Memory usage is above threshold",
			Severity:    getEnv("ALERT_MEMORY_SEVERITY", "warning"),
			Component:   "system",
			Condition: AlertConditionConfig{
				Type:      "threshold",
				Metric:    "memory_usage_percent",
				Operator:  "gt",
				Threshold: getFloatEnv("ALERT_MEMORY_THRESHOLD", 85.0),
			},
			Duration: getEnv("ALERT_MEMORY_DURATION", "5m"),
			Cooldown: getEnv("ALERT_MEMORY_COOLDOWN", "15m"),
			Enabled:  getBoolEnv("ALERT_MEMORY_ENABLED", true),
		},
		{
			Name:        "SecurityBruteForce",
			Description: "Multiple failed login attempts detected",
			Severity:    getEnv("ALERT_BRUTE_FORCE_SEVERITY", "critical"),
			Component:   "security",
			Condition: AlertConditionConfig{
				Type:      "threshold",
				Metric:    "failed_login_attempts",
				Operator:  "gt",
				Threshold: getFloatEnv("ALERT_BRUTE_FORCE_THRESHOLD", 5.0),
			},
			Duration: getEnv("ALERT_BRUTE_FORCE_DURATION", "1m"),
			Cooldown: getEnv("ALERT_BRUTE_FORCE_COOLDOWN", "5m"),
			Enabled:  getBoolEnv("ALERT_BRUTE_FORCE_ENABLED", true),
		},
	}
}

// Helper functions for environment variable parsing
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getFloatEnv(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// Validate validates the monitoring configuration
func (c *MonitoringConfig) Validate() error {
	// Basic validation
	if c.Service == "" {
		c.Service = "tiris-backend"
	}
	if c.Version == "" {
		c.Version = "dev"
	}
	if c.Environment == "" {
		c.Environment = "development"
	}
	if c.MetricsInterval <= 0 {
		c.MetricsInterval = 15 * time.Second
	}

	// Metrics validation
	if c.Metrics.Port <= 0 || c.Metrics.Port > 65535 {
		c.Metrics.Port = 9090
	}
	if c.Metrics.Path == "" {
		c.Metrics.Path = "/metrics"
	}

	// Health validation
	if c.Health.Port <= 0 || c.Health.Port > 65535 {
		c.Health.Port = 8081
	}
	if c.Health.Path == "" {
		c.Health.Path = "/health"
	}
	if c.Health.Liveness == "" {
		c.Health.Liveness = "/live"
	}
	if c.Health.Readiness == "" {
		c.Health.Readiness = "/ready"
	}
	if c.Health.Interval <= 0 {
		c.Health.Interval = 30 * time.Second
	}
	if c.Health.Timeout <= 0 {
		c.Health.Timeout = 10 * time.Second
	}

	return nil
}

// GetMonitoringTags returns standard tags for monitoring
func (c *MonitoringConfig) GetMonitoringTags() map[string]string {
	return map[string]string{
		"service":     c.Service,
		"version":     c.Version,
		"environment": c.Environment,
	}
}

// IsProductionEnvironment returns true if running in production
func (c *MonitoringConfig) IsProductionEnvironment() bool {
	env := strings.ToLower(c.Environment)
	return env == "production" || env == "prod"
}

// IsTestEnvironment returns true if running in test mode
func (c *MonitoringConfig) IsTestEnvironment() bool {
	env := strings.ToLower(c.Environment)
	return env == "test" || env == "testing"
}