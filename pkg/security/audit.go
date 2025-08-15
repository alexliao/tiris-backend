package security

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AuditLevel defines the severity level of audit events
type AuditLevel string

const (
	AuditLevelInfo     AuditLevel = "info"
	AuditLevelWarn     AuditLevel = "warn"
	AuditLevelError    AuditLevel = "error"
	AuditLevelCritical AuditLevel = "critical"
)

// AuditAction defines the type of action being audited
type AuditAction string

const (
	// Authentication actions
	ActionLogin          AuditAction = "auth.login"
	ActionLogout         AuditAction = "auth.logout"
	ActionLoginFailed    AuditAction = "auth.login_failed"
	ActionTokenRefresh   AuditAction = "auth.token_refresh"
	ActionPasswordChange AuditAction = "auth.password_change"
	ActionPasswordReset  AuditAction = "auth.password_reset"

	// User management actions
	ActionUserCreate AuditAction = "user.create"
	ActionUserUpdate AuditAction = "user.update"
	ActionUserDelete AuditAction = "user.delete"
	ActionUserView   AuditAction = "user.view"

	// Exchange management actions
	ActionExchangeCreate AuditAction = "exchange.create"
	ActionExchangeUpdate AuditAction = "exchange.update"
	ActionExchangeDelete AuditAction = "exchange.delete"
	ActionExchangeView   AuditAction = "exchange.view"

	// API key actions
	ActionAPIKeyCreate AuditAction = "apikey.create"
	ActionAPIKeyUpdate AuditAction = "apikey.update"
	ActionAPIKeyDelete AuditAction = "apikey.delete"
	ActionAPIKeyUsed   AuditAction = "apikey.used"

	// Transaction actions
	ActionTransactionCreate AuditAction = "transaction.create"
	ActionTransactionView   AuditAction = "transaction.view"
	ActionTransactionExport AuditAction = "transaction.export"

	// System actions
	ActionSystemAccess  AuditAction = "system.access"
	ActionSystemError   AuditAction = "system.error"
	ActionConfigChange  AuditAction = "system.config_change"
	ActionDataExport    AuditAction = "system.data_export"
	ActionDataImport    AuditAction = "system.data_import"
	ActionRateLimitHit  AuditAction = "system.rate_limit_hit"
	ActionSecurityAlert AuditAction = "system.security_alert"
)

// AuditEvent represents a security audit event
type AuditEvent struct {
	ID        uuid.UUID              `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	Timestamp time.Time              `json:"timestamp" gorm:"default:now();index"`
	Level     AuditLevel             `json:"level" gorm:"type:varchar(20);index"`
	Action    AuditAction            `json:"action" gorm:"type:varchar(50);index"`
	UserID    *uuid.UUID             `json:"user_id,omitempty" gorm:"type:uuid;index"`
	SessionID *string                `json:"session_id,omitempty" gorm:"type:varchar(255);index"`
	IPAddress string                 `json:"ip_address" gorm:"type:inet;index"`
	UserAgent string                 `json:"user_agent" gorm:"type:text"`
	Resource  string                 `json:"resource" gorm:"type:varchar(255);index"`
	Details   map[string]interface{} `json:"details" gorm:"type:jsonb;default:'{}'"`
	Success   bool                   `json:"success" gorm:"index"`
	Error     *string                `json:"error,omitempty" gorm:"type:text"`
	Duration  *time.Duration         `json:"duration,omitempty" gorm:"type:bigint"`

	CreatedAt time.Time `json:"created_at" gorm:"default:now()"`
}

// AuditLogger handles security audit logging
type AuditLogger struct {
	db *gorm.DB
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(db *gorm.DB) *AuditLogger {
	return &AuditLogger{db: db}
}

// LogEvent logs an audit event to the database
func (al *AuditLogger) LogEvent(ctx context.Context, event *AuditEvent) error {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	
	if event.Details == nil {
		event.Details = make(map[string]interface{})
	}

	return al.db.WithContext(ctx).Create(event).Error
}

// LogHTTPRequest logs an HTTP request audit event
func (al *AuditLogger) LogHTTPRequest(ctx context.Context, r *http.Request, action AuditAction, userID *uuid.UUID, success bool, duration time.Duration, err error) error {
	event := &AuditEvent{
		Level:     al.getLevelForAction(action, success),
		Action:    action,
		UserID:    userID,
		IPAddress: al.getClientIP(r),
		UserAgent: r.UserAgent(),
		Resource:  fmt.Sprintf("%s %s", r.Method, r.URL.Path),
		Success:   success,
		Duration:  &duration,
		Details: map[string]interface{}{
			"method":      r.Method,
			"path":        r.URL.Path,
			"query":       r.URL.RawQuery,
			"remote_addr": r.RemoteAddr,
			"referer":     r.Referer(),
		},
	}

	// Add session ID if available
	if sessionID := r.Header.Get("X-Session-ID"); sessionID != "" {
		event.SessionID = &sessionID
	}

	if err != nil {
		errStr := err.Error()
		event.Error = &errStr
		event.Details["error_type"] = fmt.Sprintf("%T", err)
	}

	return al.LogEvent(ctx, event)
}

// LogSecurityEvent logs a security-related event
func (al *AuditLogger) LogSecurityEvent(ctx context.Context, action AuditAction, userID *uuid.UUID, ipAddress string, details map[string]interface{}, err error) error {
	level := AuditLevelInfo
	success := true
	
	if err != nil {
		level = AuditLevelError
		success = false
	}

	// Escalate certain actions to higher levels
	switch action {
	case ActionLoginFailed, ActionRateLimitHit:
		level = AuditLevelWarn
	case ActionSecurityAlert:
		level = AuditLevelCritical
	}

	event := &AuditEvent{
		Level:     level,
		Action:    action,
		UserID:    userID,
		IPAddress: ipAddress,
		Success:   success,
		Details:   details,
	}

	if err != nil {
		errStr := err.Error()
		event.Error = &errStr
	}

	return al.LogEvent(ctx, event)
}

// LogDataAccess logs data access events
func (al *AuditLogger) LogDataAccess(ctx context.Context, userID *uuid.UUID, action AuditAction, resourceType, resourceID string, ipAddress string, success bool) error {
	details := map[string]interface{}{
		"resource_type": resourceType,
		"resource_id":   resourceID,
		"access_type":   string(action),
	}

	event := &AuditEvent{
		Level:     al.getLevelForAction(action, success),
		Action:    action,
		UserID:    userID,
		IPAddress: ipAddress,
		Resource:  fmt.Sprintf("%s:%s", resourceType, resourceID),
		Success:   success,
		Details:   details,
	}

	return al.LogEvent(ctx, event)
}

// GetAuditEvents retrieves audit events with filtering
func (al *AuditLogger) GetAuditEvents(ctx context.Context, filters AuditFilters) ([]AuditEvent, error) {
	query := al.db.WithContext(ctx).Model(&AuditEvent{})
	
	if filters.UserID != nil {
		query = query.Where("user_id = ?", *filters.UserID)
	}
	
	if filters.Action != "" {
		query = query.Where("action = ?", filters.Action)
	}
	
	if filters.Level != "" {
		query = query.Where("level = ?", filters.Level)
	}
	
	if filters.IPAddress != "" {
		query = query.Where("ip_address = ?", filters.IPAddress)
	}
	
	if !filters.StartTime.IsZero() {
		query = query.Where("timestamp >= ?", filters.StartTime)
	}
	
	if !filters.EndTime.IsZero() {
		query = query.Where("timestamp <= ?", filters.EndTime)
	}
	
	if filters.Success != nil {
		query = query.Where("success = ?", *filters.Success)
	}

	var events []AuditEvent
	err := query.Order("timestamp DESC").Limit(filters.Limit).Offset(filters.Offset).Find(&events).Error
	
	return events, err
}

// AuditFilters defines filters for querying audit events
type AuditFilters struct {
	UserID    *uuid.UUID  `json:"user_id,omitempty"`
	Action    AuditAction `json:"action,omitempty"`
	Level     AuditLevel  `json:"level,omitempty"`
	IPAddress string      `json:"ip_address,omitempty"`
	StartTime time.Time   `json:"start_time,omitempty"`
	EndTime   time.Time   `json:"end_time,omitempty"`
	Success   *bool       `json:"success,omitempty"`
	Limit     int         `json:"limit,omitempty"`
	Offset    int         `json:"offset,omitempty"`
}

// GetSecurityAlerts retrieves recent security alerts
func (al *AuditLogger) GetSecurityAlerts(ctx context.Context, since time.Time, limit int) ([]AuditEvent, error) {
	var events []AuditEvent
	err := al.db.WithContext(ctx).
		Where("level IN (?, ?) AND timestamp >= ?", AuditLevelWarn, AuditLevelCritical, since).
		Or("action IN (?, ?, ?)", ActionLoginFailed, ActionRateLimitHit, ActionSecurityAlert).
		Order("timestamp DESC").
		Limit(limit).
		Find(&events).Error
	
	return events, err
}

// GetSuspiciousActivity identifies potentially suspicious patterns
func (al *AuditLogger) GetSuspiciousActivity(ctx context.Context, timeWindow time.Duration) ([]SuspiciousActivity, error) {
	since := time.Now().Add(-timeWindow)
	
	var results []SuspiciousActivity
	
	// Multiple failed logins from same IP
	var failedLogins []struct {
		IPAddress string `json:"ip_address"`
		Count     int    `json:"count"`
	}
	
	err := al.db.WithContext(ctx).
		Model(&AuditEvent{}).
		Select("ip_address, COUNT(*) as count").
		Where("action = ? AND success = false AND timestamp >= ?", ActionLoginFailed, since).
		Group("ip_address").
		Having("COUNT(*) >= ?", 5).
		Find(&failedLogins).Error
	
	if err != nil {
		return nil, err
	}
	
	for _, login := range failedLogins {
		results = append(results, SuspiciousActivity{
			Type:        "multiple_failed_logins",
			IPAddress:   login.IPAddress,
			Count:       login.Count,
			Severity:    "high",
			Description: fmt.Sprintf("%d failed login attempts from IP %s", login.Count, login.IPAddress),
		})
	}
	
	// Rate limit violations
	var rateLimitHits []struct {
		IPAddress string `json:"ip_address"`
		Count     int    `json:"count"`
	}
	
	err = al.db.WithContext(ctx).
		Model(&AuditEvent{}).
		Select("ip_address, COUNT(*) as count").
		Where("action = ? AND timestamp >= ?", ActionRateLimitHit, since).
		Group("ip_address").
		Having("COUNT(*) >= ?", 3).
		Find(&rateLimitHits).Error
	
	if err != nil {
		return nil, err
	}
	
	for _, hit := range rateLimitHits {
		results = append(results, SuspiciousActivity{
			Type:        "excessive_rate_limiting",
			IPAddress:   hit.IPAddress,
			Count:       hit.Count,
			Severity:    "medium",
			Description: fmt.Sprintf("%d rate limit violations from IP %s", hit.Count, hit.IPAddress),
		})
	}
	
	return results, nil
}

// SuspiciousActivity represents detected suspicious activity
type SuspiciousActivity struct {
	Type        string `json:"type"`
	IPAddress   string `json:"ip_address"`
	UserID      string `json:"user_id,omitempty"`
	Count       int    `json:"count"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
}

// CleanupOldEvents removes audit events older than the specified retention period
func (al *AuditLogger) CleanupOldEvents(ctx context.Context, retentionPeriod time.Duration) error {
	cutoff := time.Now().Add(-retentionPeriod)
	
	result := al.db.WithContext(ctx).
		Where("timestamp < ?", cutoff).
		Delete(&AuditEvent{})
	
	if result.Error != nil {
		return result.Error
	}
	
	return nil
}

// Helper methods
func (al *AuditLogger) getLevelForAction(action AuditAction, success bool) AuditLevel {
	if !success {
		switch action {
		case ActionLogin, ActionPasswordChange, ActionPasswordReset:
			return AuditLevelWarn
		case ActionSystemAccess, ActionConfigChange:
			return AuditLevelError
		default:
			return AuditLevelInfo
		}
	}
	
	switch action {
	case ActionPasswordChange, ActionPasswordReset, ActionConfigChange, ActionDataExport:
		return AuditLevelWarn
	case ActionSecurityAlert:
		return AuditLevelCritical
	default:
		return AuditLevelInfo
	}
}

func (al *AuditLogger) getClientIP(r *http.Request) string {
	// Check for forwarded IP in headers (for proxy/load balancer scenarios)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		if ip := net.ParseIP(forwarded); ip != nil {
			return forwarded
		}
	}
	
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		if ip := net.ParseIP(realIP); ip != nil {
			return realIP
		}
	}
	
	// Fall back to remote address
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	
	return ip
}