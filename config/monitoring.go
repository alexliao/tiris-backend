package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// MonitoringConfig holds monitoring system configuration
type MonitoringConfig struct {
	// Metrics configuration
	MetricsEnabled bool   `json:"metrics_enabled"`
	MetricsPort    int    `json:"metrics_port"`
	MetricsPath    string `json:"metrics_path"`

	// Logging configuration
	LoggingEnabled bool   `json:"logging_enabled"`
	LogLevel       string `json:"log_level"`
	LogFormat      string `json:"log_format"` // "json" or "text"
	LogOutput      string `json:"log_output"` // "stdout", "file", or file path

	// Alert configuration
	AlertsEnabled bool                  `json:"alerts_enabled"`
	AlertReceivers []AlertReceiverConfig `json:"alert_receivers"`
	
	// Health check configuration
	HealthCheckEnabled  bool          `json:"health_check_enabled"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	HealthCheckTimeout  time.Duration `json:"health_check_timeout"`
	
	// Database monitoring
	DatabaseMonitoringEnabled bool          `json:"database_monitoring_enabled"`
	SlowQueryThreshold        time.Duration `json:"slow_query_threshold"`
	
	// Redis monitoring
	RedisMonitoringEnabled bool          `json:"redis_monitoring_enabled"`
	RedisSlowCommandThreshold time.Duration `json:"redis_slow_command_threshold"`
	
	// Security monitoring
	SecurityMonitoringEnabled bool `json:"security_monitoring_enabled"`
	SecurityAlertCooldown     time.Duration `json:"security_alert_cooldown"`
}

// AlertReceiverConfig holds alert receiver configuration
type AlertReceiverConfig struct {
	Type     string            `json:"type"`     // "webhook", "slack", "email"
	Name     string            `json:"name"`
	URL      string            `json:"url,omitempty"`
	Token    string            `json:"token,omitempty"`
	Channel  string            `json:"channel,omitempty"`
	Username string            `json:"username,omitempty"`
	Password string            `json:"password,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
	Enabled  bool              `json:"enabled"`
}

// LoadMonitoringConfig loads monitoring configuration from environment variables
func LoadMonitoringConfig() (*MonitoringConfig, error) {
	config := &MonitoringConfig{
		// Default values
		MetricsEnabled: true,
		MetricsPort:    9090,
		MetricsPath:    "/metrics",
		
		LoggingEnabled: true,
		LogLevel:       "info",
		LogFormat:      "json",
		LogOutput:      "stdout",
		
		AlertsEnabled: false,
		
		HealthCheckEnabled:  true,
		HealthCheckInterval: 30 * time.Second,
		HealthCheckTimeout:  5 * time.Second,
		
		DatabaseMonitoringEnabled: true,
		SlowQueryThreshold:        500 * time.Millisecond,
		
		RedisMonitoringEnabled: true,
		RedisSlowCommandThreshold: 100 * time.Millisecond,
		
		SecurityMonitoringEnabled: true,
		SecurityAlertCooldown:     5 * time.Minute,
	}

	// Load from environment variables
	if val := os.Getenv("MONITORING_METRICS_ENABLED"); val != "" {
		config.MetricsEnabled = val == "true"
	}
	
	if val := os.Getenv("MONITORING_METRICS_PORT"); val != "" {
		if port, err := strconv.Atoi(val); err == nil {
			config.MetricsPort = port
		}
	}
	
	if val := os.Getenv("MONITORING_METRICS_PATH"); val != "" {
		config.MetricsPath = val
	}
	
	if val := os.Getenv("MONITORING_LOGGING_ENABLED"); val != "" {
		config.LoggingEnabled = val == "true"
	}
	
	if val := os.Getenv("MONITORING_LOG_LEVEL"); val != "" {
		config.LogLevel = strings.ToLower(val)
	}
	
	if val := os.Getenv("MONITORING_LOG_FORMAT"); val != "" {
		config.LogFormat = strings.ToLower(val)
	}
	
	if val := os.Getenv("MONITORING_LOG_OUTPUT"); val != "" {
		config.LogOutput = val
	}
	
	if val := os.Getenv("MONITORING_ALERTS_ENABLED"); val != "" {
		config.AlertsEnabled = val == "true"
	}
	
	if val := os.Getenv("MONITORING_HEALTH_CHECK_ENABLED"); val != "" {
		config.HealthCheckEnabled = val == "true"
	}
	
	if val := os.Getenv("MONITORING_HEALTH_CHECK_INTERVAL"); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			config.HealthCheckInterval = duration
		}
	}
	
	if val := os.Getenv("MONITORING_HEALTH_CHECK_TIMEOUT"); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			config.HealthCheckTimeout = duration
		}
	}
	
	if val := os.Getenv("MONITORING_DATABASE_ENABLED"); val != "" {
		config.DatabaseMonitoringEnabled = val == "true"
	}
	
	if val := os.Getenv("MONITORING_SLOW_QUERY_THRESHOLD"); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			config.SlowQueryThreshold = duration
		}
	}
	
	if val := os.Getenv("MONITORING_REDIS_ENABLED"); val != "" {
		config.RedisMonitoringEnabled = val == "true"
	}
	
	if val := os.Getenv("MONITORING_REDIS_SLOW_COMMAND_THRESHOLD"); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			config.RedisSlowCommandThreshold = duration
		}
	}
	
	if val := os.Getenv("MONITORING_SECURITY_ENABLED"); val != "" {
		config.SecurityMonitoringEnabled = val == "true"
	}
	
	if val := os.Getenv("MONITORING_SECURITY_ALERT_COOLDOWN"); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			config.SecurityAlertCooldown = duration
		}
	}

	// Load alert receivers from environment
	config.AlertReceivers = loadAlertReceiversFromEnv()

	return config, nil
}

// loadAlertReceiversFromEnv loads alert receiver configurations from environment variables
func loadAlertReceiversFromEnv() []AlertReceiverConfig {
	var receivers []AlertReceiverConfig

	// Slack receiver
	if slackURL := os.Getenv("MONITORING_SLACK_WEBHOOK_URL"); slackURL != "" {
		receiver := AlertReceiverConfig{
			Type:    "slack",
			Name:    "slack-alerts",
			URL:     slackURL,
			Channel: os.Getenv("MONITORING_SLACK_CHANNEL"),
			Enabled: os.Getenv("MONITORING_SLACK_ENABLED") == "true",
		}
		if receiver.Channel == "" {
			receiver.Channel = "#alerts"
		}
		receivers = append(receivers, receiver)
	}

	// Generic webhook receiver
	if webhookURL := os.Getenv("MONITORING_WEBHOOK_URL"); webhookURL != "" {
		headers := make(map[string]string)
		if token := os.Getenv("MONITORING_WEBHOOK_TOKEN"); token != "" {
			headers["Authorization"] = fmt.Sprintf("Bearer %s", token)
		}
		if contentType := os.Getenv("MONITORING_WEBHOOK_CONTENT_TYPE"); contentType != "" {
			headers["Content-Type"] = contentType
		} else {
			headers["Content-Type"] = "application/json"
		}

		receiver := AlertReceiverConfig{
			Type:    "webhook",
			Name:    "webhook-alerts",
			URL:     webhookURL,
			Headers: headers,
			Enabled: os.Getenv("MONITORING_WEBHOOK_ENABLED") == "true",
		}
		receivers = append(receivers, receiver)
	}

	return receivers
}

// Validate validates the monitoring configuration
func (c *MonitoringConfig) Validate() error {
	if c.MetricsPort <= 0 || c.MetricsPort > 65535 {
		return fmt.Errorf("invalid metrics port: %d", c.MetricsPort)
	}

	if c.LogLevel != "" {
		validLevels := []string{"debug", "info", "warn", "error"}
		valid := false
		for _, level := range validLevels {
			if c.LogLevel == level {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid log level: %s (must be one of: %s)", c.LogLevel, strings.Join(validLevels, ", "))
		}
	}

	if c.LogFormat != "" && c.LogFormat != "json" && c.LogFormat != "text" {
		return fmt.Errorf("invalid log format: %s (must be 'json' or 'text')", c.LogFormat)
	}

	if c.HealthCheckInterval <= 0 {
		return fmt.Errorf("invalid health check interval: %v", c.HealthCheckInterval)
	}

	if c.HealthCheckTimeout <= 0 {
		return fmt.Errorf("invalid health check timeout: %v", c.HealthCheckTimeout)
	}

	// Validate alert receivers
	for i, receiver := range c.AlertReceivers {
		if receiver.Enabled {
			if receiver.Name == "" {
				return fmt.Errorf("alert receiver %d: name is required", i)
			}
			if receiver.Type == "" {
				return fmt.Errorf("alert receiver %d: type is required", i)
			}
			if receiver.Type == "webhook" || receiver.Type == "slack" {
				if receiver.URL == "" {
					return fmt.Errorf("alert receiver %d: URL is required for type %s", i, receiver.Type)
				}
			}
		}
	}

	return nil
}

// GetEnabledAlertReceivers returns only the enabled alert receivers
func (c *MonitoringConfig) GetEnabledAlertReceivers() []AlertReceiverConfig {
	var enabled []AlertReceiverConfig
	for _, receiver := range c.AlertReceivers {
		if receiver.Enabled {
			enabled = append(enabled, receiver)
		}
	}
	return enabled
}