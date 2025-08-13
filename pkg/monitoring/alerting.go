package monitoring

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AlertSeverity defines the severity level of alerts
type AlertSeverity string

const (
	SeverityInfo     AlertSeverity = "info"
	SeverityWarning  AlertSeverity = "warning"
	SeverityCritical AlertSeverity = "critical"
)

// AlertStatus defines the current status of an alert
type AlertStatus string

const (
	StatusFiring   AlertStatus = "firing"
	StatusResolved AlertStatus = "resolved"
	StatusSilenced AlertStatus = "silenced"
)

// Alert represents a system alert
type Alert struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Severity    AlertSeverity          `json:"severity"`
	Status      AlertStatus            `json:"status"`
	Source      string                 `json:"source"`
	Component   string                 `json:"component,omitempty"`
	Service     string                 `json:"service"`
	Environment string                 `json:"environment"`
	Labels      map[string]string      `json:"labels,omitempty"`
	Annotations map[string]string      `json:"annotations,omitempty"`
	Value       interface{}            `json:"value,omitempty"`
	Threshold   interface{}            `json:"threshold,omitempty"`
	FiredAt     time.Time              `json:"fired_at"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
	LastSeen    time.Time              `json:"last_seen"`
	Count       int                    `json:"count"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// AlertRule defines conditions that trigger alerts
type AlertRule struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Severity     AlertSeverity          `json:"severity"`
	Component    string                 `json:"component,omitempty"`
	Condition    AlertCondition         `json:"condition"`
	Duration     time.Duration          `json:"duration"`
	Annotations  map[string]string      `json:"annotations,omitempty"`
	Labels       map[string]string      `json:"labels,omitempty"`
	Enabled      bool                   `json:"enabled"`
	Cooldown     time.Duration          `json:"cooldown,omitempty"`
	LastTriggered *time.Time            `json:"last_triggered,omitempty"`
}

// AlertCondition defines the condition that triggers an alert
type AlertCondition struct {
	Type      string      `json:"type"`      // "threshold", "change", "absence"
	Metric    string      `json:"metric"`
	Operator  string      `json:"operator"`  // "gt", "lt", "eq", "ne", "gte", "lte"
	Threshold interface{} `json:"threshold"`
	Window    time.Duration `json:"window,omitempty"`
}

// AlertReceiver defines how alerts are sent
type AlertReceiver interface {
	SendAlert(ctx context.Context, alert *Alert) error
	GetName() string
}

// AlertManager manages the alerting system
type AlertManager struct {
	rules     map[string]*AlertRule
	alerts    map[string]*Alert
	receivers []AlertReceiver
	logger    *Logger
	mu        sync.RWMutex
	
	// Configuration
	service     string
	environment string
	enabled     bool
}

// NewAlertManager creates a new alert manager
func NewAlertManager(service, environment string, logger *Logger) *AlertManager {
	return &AlertManager{
		rules:       make(map[string]*AlertRule),
		alerts:      make(map[string]*Alert),
		receivers:   make([]AlertReceiver, 0),
		logger:      logger,
		service:     service,
		environment: environment,
		enabled:     true,
	}
}

// AddRule adds an alert rule
func (am *AlertManager) AddRule(rule *AlertRule) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.rules[rule.Name] = rule
}

// AddReceiver adds an alert receiver
func (am *AlertManager) AddReceiver(receiver AlertReceiver) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.receivers = append(am.receivers, receiver)
}

// CheckThreshold checks if a metric value triggers any alert rules
func (am *AlertManager) CheckThreshold(metric string, value interface{}, labels map[string]string) {
	if !am.enabled {
		return
	}
	
	am.mu.RLock()
	defer am.mu.RUnlock()
	
	for _, rule := range am.rules {
		if !rule.Enabled || rule.Condition.Metric != metric {
			continue
		}
		
		// Check cooldown period
		if rule.LastTriggered != nil && rule.Cooldown > 0 {
			if time.Since(*rule.LastTriggered) < rule.Cooldown {
				continue
			}
		}
		
		if am.evaluateCondition(rule.Condition, value) {
			am.fireAlert(rule, value, labels)
		}
	}
}

// FireAlert manually fires an alert
func (am *AlertManager) FireAlert(name, description string, severity AlertSeverity, component string, details map[string]interface{}) {
	if !am.enabled {
		return
	}
	
	alert := &Alert{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Severity:    severity,
		Status:      StatusFiring,
		Source:      "manual",
		Component:   component,
		Service:     am.service,
		Environment: am.environment,
		FiredAt:     time.Now().UTC(),
		LastSeen:    time.Now().UTC(),
		Count:       1,
		Details:     details,
	}
	
	am.processAlert(alert)
}

// FireSecurityAlert fires a security-related alert
func (am *AlertManager) FireSecurityAlert(eventType, description, userID, ipAddress string, details map[string]interface{}) {
	severity := SeverityWarning
	if strings.Contains(strings.ToLower(eventType), "critical") ||
	   strings.Contains(strings.ToLower(eventType), "breach") ||
	   strings.Contains(strings.ToLower(eventType), "attack") {
		severity = SeverityCritical
	}
	
	alertDetails := map[string]interface{}{
		"event_type": eventType,
		"user_id":    userID,
		"ip_address": ipAddress,
		"timestamp":  time.Now().UTC(),
	}
	
	for k, v := range details {
		alertDetails[k] = v
	}
	
	am.FireAlert(
		fmt.Sprintf("Security Event: %s", eventType),
		description,
		severity,
		"security",
		alertDetails,
	)
}

// FirePerformanceAlert fires a performance-related alert
func (am *AlertManager) FirePerformanceAlert(metric string, value, threshold interface{}, component string) {
	severity := SeverityWarning
	
	// Determine severity based on how much the threshold is exceeded
	if valueFloat, ok := value.(float64); ok {
		if thresholdFloat, ok := threshold.(float64); ok {
			if valueFloat > thresholdFloat*1.5 {
				severity = SeverityCritical
			}
		}
	}
	
	am.FireAlert(
		fmt.Sprintf("Performance Alert: %s", metric),
		fmt.Sprintf("%s exceeded threshold: %v > %v", metric, value, threshold),
		severity,
		component,
		map[string]interface{}{
			"metric":    metric,
			"value":     value,
			"threshold": threshold,
		},
	)
}

// FireBusinessAlert fires a business-related alert
func (am *AlertManager) FireBusinessAlert(event, description string, details map[string]interface{}) {
	am.FireAlert(
		fmt.Sprintf("Business Alert: %s", event),
		description,
		SeverityWarning,
		"business",
		details,
	)
}

// ResolveAlert resolves an active alert
func (am *AlertManager) ResolveAlert(alertID string) {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	if alert, exists := am.alerts[alertID]; exists {
		if alert.Status == StatusFiring {
			now := time.Now().UTC()
			alert.Status = StatusResolved
			alert.ResolvedAt = &now
			am.notifyReceivers(context.Background(), alert)
			
			am.logger.Info("Alert resolved: %s", alert.Name)
		}
	}
}

// GetActiveAlerts returns all currently active alerts
func (am *AlertManager) GetActiveAlerts() []*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()
	
	var activeAlerts []*Alert
	for _, alert := range am.alerts {
		if alert.Status == StatusFiring {
			activeAlerts = append(activeAlerts, alert)
		}
	}
	
	return activeAlerts
}

// GetAlertHistory returns alert history with optional filtering
func (am *AlertManager) GetAlertHistory(since time.Time, severity AlertSeverity, component string) []*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()
	
	var history []*Alert
	for _, alert := range am.alerts {
		if alert.FiredAt.Before(since) {
			continue
		}
		
		if severity != "" && alert.Severity != severity {
			continue
		}
		
		if component != "" && alert.Component != component {
			continue
		}
		
		history = append(history, alert)
	}
	
	return history
}

// Internal methods

func (am *AlertManager) evaluateCondition(condition AlertCondition, value interface{}) bool {
	switch condition.Type {
	case "threshold":
		return am.evaluateThresholdCondition(condition, value)
	case "change":
		// TODO: Implement change detection
		return false
	case "absence":
		// TODO: Implement absence detection
		return false
	default:
		return false
	}
}

func (am *AlertManager) evaluateThresholdCondition(condition AlertCondition, value interface{}) bool {
	valueFloat, valueOK := value.(float64)
	thresholdFloat, thresholdOK := condition.Threshold.(float64)
	
	if !valueOK || !thresholdOK {
		return false
	}
	
	switch condition.Operator {
	case "gt":
		return valueFloat > thresholdFloat
	case "lt":
		return valueFloat < thresholdFloat
	case "gte":
		return valueFloat >= thresholdFloat
	case "lte":
		return valueFloat <= thresholdFloat
	case "eq":
		return valueFloat == thresholdFloat
	case "ne":
		return valueFloat != thresholdFloat
	default:
		return false
	}
}

func (am *AlertManager) fireAlert(rule *AlertRule, value interface{}, labels map[string]string) {
	// Check if similar alert already exists
	alertKey := fmt.Sprintf("%s:%s", rule.Name, am.generateLabelKey(labels))
	
	am.mu.Lock()
	defer am.mu.Unlock()
	
	now := time.Now().UTC()
	rule.LastTriggered = &now
	
	if existingAlert, exists := am.alerts[alertKey]; exists {
		// Update existing alert
		existingAlert.Count++
		existingAlert.LastSeen = now
		existingAlert.Value = value
		
		// Don't re-notify if alert is still firing
		if existingAlert.Status == StatusFiring {
			return
		}
		
		existingAlert.Status = StatusFiring
		existingAlert.ResolvedAt = nil
	} else {
		// Create new alert
		alert := &Alert{
			ID:          uuid.New().String(),
			Name:        rule.Name,
			Description: rule.Description,
			Severity:    rule.Severity,
			Status:      StatusFiring,
			Source:      "rule",
			Component:   rule.Component,
			Service:     am.service,
			Environment: am.environment,
			Labels:      labels,
			Annotations: rule.Annotations,
			Value:       value,
			Threshold:   rule.Condition.Threshold,
			FiredAt:     now,
			LastSeen:    now,
			Count:       1,
		}
		
		am.alerts[alertKey] = alert
	}
	
	// Notify receivers
	am.notifyReceivers(context.Background(), am.alerts[alertKey])
}

func (am *AlertManager) processAlert(alert *Alert) {
	alertKey := fmt.Sprintf("%s:%s", alert.Name, am.generateLabelKey(alert.Labels))
	
	am.mu.Lock()
	am.alerts[alertKey] = alert
	am.mu.Unlock()
	
	am.notifyReceivers(context.Background(), alert)
	
	am.logger.LogSecurity("alert_fired", string(alert.Severity), "", "", map[string]interface{}{
		"alert_name":    alert.Name,
		"alert_id":      alert.ID,
		"component":     alert.Component,
		"description":   alert.Description,
	})
}

func (am *AlertManager) notifyReceivers(ctx context.Context, alert *Alert) {
	for _, receiver := range am.receivers {
		go func(r AlertReceiver) {
			if err := r.SendAlert(ctx, alert); err != nil {
				am.logger.Error("Failed to send alert to %s: %v", r.GetName(), err)
			}
		}(receiver)
	}
}

func (am *AlertManager) generateLabelKey(labels map[string]string) string {
	if len(labels) == 0 {
		return "default"
	}
	
	var parts []string
	for k, v := range labels {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	
	return strings.Join(parts, ",")
}

// Webhook Alert Receiver
type WebhookReceiver struct {
	name    string
	url     string
	headers map[string]string
	client  *http.Client
}

func NewWebhookReceiver(name, url string, headers map[string]string) *WebhookReceiver {
	return &WebhookReceiver{
		name:    name,
		url:     url,
		headers: headers,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (wr *WebhookReceiver) SendAlert(ctx context.Context, alert *Alert) error {
	payload, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("failed to marshal alert: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", wr.url, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	for k, v := range wr.headers {
		req.Header.Set(k, v)
	}
	
	resp, err := wr.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned error status: %d", resp.StatusCode)
	}
	
	return nil
}

func (wr *WebhookReceiver) GetName() string {
	return wr.name
}

// Slack Alert Receiver
type SlackReceiver struct {
	name       string
	webhookURL string
	channel    string
	client     *http.Client
}

type SlackMessage struct {
	Channel     string            `json:"channel,omitempty"`
	Username    string            `json:"username,omitempty"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	Text        string            `json:"text"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
}

type SlackAttachment struct {
	Color     string       `json:"color,omitempty"`
	Title     string       `json:"title,omitempty"`
	Text      string       `json:"text,omitempty"`
	Fields    []SlackField `json:"fields,omitempty"`
	Timestamp int64        `json:"ts,omitempty"`
}

type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

func NewSlackReceiver(name, webhookURL, channel string) *SlackReceiver {
	return &SlackReceiver{
		name:       name,
		webhookURL: webhookURL,
		channel:    channel,
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

func (sr *SlackReceiver) SendAlert(ctx context.Context, alert *Alert) error {
	color := "warning"
	emoji := ":warning:"
	
	switch alert.Severity {
	case SeverityInfo:
		color = "good"
		emoji = ":information_source:"
	case SeverityCritical:
		color = "danger"
		emoji = ":rotating_light:"
	}
	
	fields := []SlackField{
		{Title: "Service", Value: alert.Service, Short: true},
		{Title: "Environment", Value: alert.Environment, Short: true},
		{Title: "Component", Value: alert.Component, Short: true},
		{Title: "Severity", Value: string(alert.Severity), Short: true},
		{Title: "Status", Value: string(alert.Status), Short: true},
	}
	
	if alert.Value != nil {
		fields = append(fields, SlackField{
			Title: "Value",
			Value: fmt.Sprintf("%v", alert.Value),
			Short: true,
		})
	}
	
	if alert.Threshold != nil {
		fields = append(fields, SlackField{
			Title: "Threshold",
			Value: fmt.Sprintf("%v", alert.Threshold),
			Short: true,
		})
	}
	
	message := SlackMessage{
		Channel:   sr.channel,
		Username:  "Tiris Backend",
		IconEmoji: emoji,
		Text:      fmt.Sprintf("%s Alert: %s", strings.Title(string(alert.Severity)), alert.Name),
		Attachments: []SlackAttachment{
			{
				Color:     color,
				Title:     alert.Name,
				Text:      alert.Description,
				Fields:    fields,
				Timestamp: alert.FiredAt.Unix(),
			},
		},
	}
	
	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack message: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", sr.webhookURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := sr.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Slack message: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		return fmt.Errorf("Slack API returned error status: %d", resp.StatusCode)
	}
	
	return nil
}

func (sr *SlackReceiver) GetName() string {
	return sr.name
}

// Default alert rules for common scenarios
func GetDefaultAlertRules() []*AlertRule {
	return []*AlertRule{
		{
			Name:        "HighErrorRate",
			Description: "HTTP error rate is above 5%",
			Severity:    SeverityWarning,
			Component:   "http",
			Condition: AlertCondition{
				Type:      "threshold",
				Metric:    "http_error_rate",
				Operator:  "gt",
				Threshold: 5.0,
			},
			Duration: 5 * time.Minute,
			Enabled:  true,
			Cooldown: 15 * time.Minute,
		},
		{
			Name:        "HighResponseTime",
			Description: "Average response time is above 1 second",
			Severity:    SeverityCritical,
			Component:   "http",
			Condition: AlertCondition{
				Type:      "threshold",
				Metric:    "http_response_time_avg",
				Operator:  "gt",
				Threshold: 1000.0, // milliseconds
			},
			Duration: 2 * time.Minute,
			Enabled:  true,
			Cooldown: 10 * time.Minute,
		},
		{
			Name:        "DatabaseConnectionsHigh",
			Description: "Database connection usage is above 80%",
			Severity:    SeverityWarning,
			Component:   "database",
			Condition: AlertCondition{
				Type:      "threshold",
				Metric:    "database_connections_usage_percent",
				Operator:  "gt",
				Threshold: 80.0,
			},
			Duration: 5 * time.Minute,
			Enabled:  true,
			Cooldown: 15 * time.Minute,
		},
		{
			Name:        "MemoryUsageHigh",
			Description: "Memory usage is above 85%",
			Severity:    SeverityWarning,
			Component:   "system",
			Condition: AlertCondition{
				Type:      "threshold",
				Metric:    "memory_usage_percent",
				Operator:  "gt",
				Threshold: 85.0,
			},
			Duration: 5 * time.Minute,
			Enabled:  true,
			Cooldown: 15 * time.Minute,
		},
		{
			Name:        "SecurityBruteForce",
			Description: "Multiple failed login attempts detected",
			Severity:    SeverityCritical,
			Component:   "security",
			Condition: AlertCondition{
				Type:      "threshold",
				Metric:    "failed_login_attempts",
				Operator:  "gt",
				Threshold: 5.0,
			},
			Duration: 1 * time.Minute,
			Enabled:  true,
			Cooldown: 5 * time.Minute,
		},
	}
}