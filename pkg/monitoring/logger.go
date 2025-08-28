package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// LogLevel defines the severity level of log messages
type LogLevel int

const (
	LevelTrace LogLevel = iota
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

var logLevelNames = map[LogLevel]string{
	LevelTrace: "TRACE",
	LevelDebug: "DEBUG",
	LevelInfo:  "INFO",
	LevelWarn:  "WARN",
	LevelError: "ERROR",
	LevelFatal: "FATAL",
}

var logLevelValues = map[string]LogLevel{
	"TRACE": LevelTrace,
	"DEBUG": LevelDebug,
	"INFO":  LevelInfo,
	"WARN":  LevelWarn,
	"ERROR": LevelError,
	"FATAL": LevelFatal,
}

// LogFormat defines the output format for logs
type LogFormat string

const (
	FormatJSON LogFormat = "json"
	FormatText LogFormat = "text"
)

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp   time.Time              `json:"timestamp"`
	Level       string                 `json:"level"`
	Message     string                 `json:"message"`
	Service     string                 `json:"service"`
	Version     string                 `json:"version"`
	RequestID   string                 `json:"request_id,omitempty"`
	UserID      string                 `json:"user_id,omitempty"`
	SessionID   string                 `json:"session_id,omitempty"`
	TraceID     string                 `json:"trace_id,omitempty"`
	SpanID      string                 `json:"span_id,omitempty"`
	Component   string                 `json:"component,omitempty"`
	Operation   string                 `json:"operation,omitempty"`
	Duration    *float64               `json:"duration_ms,omitempty"`
	Error       string                 `json:"error,omitempty"`
	StackTrace  string                 `json:"stack_trace,omitempty"`
	Fields      map[string]interface{} `json:"fields,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	
	// HTTP-specific fields
	Method     string `json:"method,omitempty"`
	Path       string `json:"path,omitempty"`
	StatusCode int    `json:"status_code,omitempty"`
	UserAgent  string `json:"user_agent,omitempty"`
	RemoteIP   string `json:"remote_ip,omitempty"`
	Referer    string `json:"referer,omitempty"`
	
	// Database-specific fields
	Query      string  `json:"query,omitempty"`
	Table      string  `json:"table,omitempty"`
	RowsAffected *int64 `json:"rows_affected,omitempty"`
	
	// Business-specific fields
	TradingPlatform string  `json:"trading_platform,omitempty"`
	Symbol     string  `json:"symbol,omitempty"`
	Amount     *float64 `json:"amount,omitempty"`
}

// Logger provides structured logging with multiple outputs and formats
type Logger struct {
	level      LogLevel
	format     LogFormat
	service    string
	version    string
	outputs    []io.Writer
	mu         sync.RWMutex
	hooks      []LogHook
	fields     map[string]interface{}
}

// LogHook defines a function that can modify log entries before output
type LogHook func(*LogEntry)

// LoggerConfig holds logger configuration
type LoggerConfig struct {
	Level     string      `json:"level"`
	Format    string      `json:"format"`
	Service   string      `json:"service"`
	Version   string      `json:"version"`
	Outputs   []string    `json:"outputs"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// NewLogger creates a new logger with the given configuration
func NewLogger(config LoggerConfig) *Logger {
	level := LevelInfo
	if l, exists := logLevelValues[strings.ToUpper(config.Level)]; exists {
		level = l
	}

	format := FormatJSON
	if config.Format == "text" {
		format = FormatText
	}

	logger := &Logger{
		level:   level,
		format:  format,
		service: config.Service,
		version: config.Version,
		fields:  make(map[string]interface{}),
	}

	// Copy default fields
	for k, v := range config.Fields {
		logger.fields[k] = v
	}

	// Set up outputs
	if len(config.Outputs) == 0 {
		logger.outputs = []io.Writer{os.Stdout}
	} else {
		for _, output := range config.Outputs {
			switch output {
			case "stdout":
				logger.outputs = append(logger.outputs, os.Stdout)
			case "stderr":
				logger.outputs = append(logger.outputs, os.Stderr)
			default:
				// Could add file outputs here
				logger.outputs = append(logger.outputs, os.Stdout)
			}
		}
	}

	return logger
}

// WithField returns a new logger with the given field added
func (l *Logger) WithField(key string, value interface{}) *Logger {
	l.mu.RLock()
	newFields := make(map[string]interface{})
	for k, v := range l.fields {
		newFields[k] = v
	}
	l.mu.RUnlock()
	
	newFields[key] = value
	
	return &Logger{
		level:   l.level,
		format:  l.format,
		service: l.service,
		version: l.version,
		outputs: l.outputs,
		hooks:   l.hooks,
		fields:  newFields,
	}
}

// WithFields returns a new logger with the given fields added
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	l.mu.RLock()
	newFields := make(map[string]interface{})
	for k, v := range l.fields {
		newFields[k] = v
	}
	l.mu.RUnlock()
	
	for k, v := range fields {
		newFields[k] = v
	}
	
	return &Logger{
		level:   l.level,
		format:  l.format,
		service: l.service,
		version: l.version,
		outputs: l.outputs,
		hooks:   l.hooks,
		fields:  newFields,
	}
}

// WithContext returns a logger with context information
func (l *Logger) WithContext(ctx context.Context) *Logger {
	fields := make(map[string]interface{})
	
	// Extract request ID from context
	if requestID := ctx.Value("request_id"); requestID != nil {
		fields["request_id"] = requestID
	}
	
	// Extract user ID from context
	if userID := ctx.Value("user_id"); userID != nil {
		fields["user_id"] = userID
	}
	
	// Extract trace information from context
	if traceID := ctx.Value("trace_id"); traceID != nil {
		fields["trace_id"] = traceID
	}
	
	if spanID := ctx.Value("span_id"); spanID != nil {
		fields["span_id"] = spanID
	}
	
	return l.WithFields(fields)
}

// AddHook adds a log hook that will be called for each log entry
func (l *Logger) AddHook(hook LogHook) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.hooks = append(l.hooks, hook)
}

// Log methods
func (l *Logger) Trace(message string, args ...interface{}) {
	l.log(LevelTrace, fmt.Sprintf(message, args...), nil)
}

func (l *Logger) Debug(message string, args ...interface{}) {
	l.log(LevelDebug, fmt.Sprintf(message, args...), nil)
}

func (l *Logger) Info(message string, args ...interface{}) {
	l.log(LevelInfo, fmt.Sprintf(message, args...), nil)
}

func (l *Logger) Warn(message string, args ...interface{}) {
	l.log(LevelWarn, fmt.Sprintf(message, args...), nil)
}

func (l *Logger) Error(message string, args ...interface{}) {
	l.log(LevelError, fmt.Sprintf(message, args...), nil)
}

func (l *Logger) Fatal(message string, args ...interface{}) {
	l.log(LevelFatal, fmt.Sprintf(message, args...), nil)
	os.Exit(1)
}

// Log with error
func (l *Logger) ErrorWithError(err error, message string, args ...interface{}) {
	entry := &LogEntry{Error: err.Error()}
	if l.level <= LevelError {
		entry.StackTrace = getStackTrace()
	}
	l.log(LevelError, fmt.Sprintf(message, args...), entry)
}

// Structured logging methods
func (l *Logger) LogHTTP(method, path string, statusCode int, duration time.Duration, userID, requestID string) {
	if l.level > LevelInfo {
		return
	}
	
	durationMs := float64(duration.Nanoseconds()) / 1e6
	entry := &LogEntry{
		Method:     method,
		Path:       path,
		StatusCode: statusCode,
		Duration:   &durationMs,
		RequestID:  requestID,
		UserID:     userID,
		Component:  "http",
	}
	
	l.log(LevelInfo, fmt.Sprintf("HTTP %s %s %d", method, path, statusCode), entry)
}

func (l *Logger) LogDatabase(operation, table string, duration time.Duration, rowsAffected int64, err error) {
	level := LevelDebug
	if err != nil {
		level = LevelError
	}
	
	durationMs := float64(duration.Nanoseconds()) / 1e6
	entry := &LogEntry{
		Operation:    operation,
		Table:        table,
		Duration:     &durationMs,
		RowsAffected: &rowsAffected,
		Component:    "database",
	}
	
	if err != nil {
		entry.Error = err.Error()
	}
	
	message := fmt.Sprintf("DB %s %s", operation, table)
	if err != nil {
		message += fmt.Sprintf(" failed: %v", err)
	}
	
	l.log(level, message, entry)
}

func (l *Logger) LogSecurity(eventType, severity, userID, ipAddress string, details map[string]interface{}) {
	level := LevelWarn
	if severity == "critical" {
		level = LevelError
	}
	
	entry := &LogEntry{
		Operation: eventType,
		UserID:    userID,
		RemoteIP:  ipAddress,
		Component: "security",
		Fields:    details,
		Tags:      []string{"security", severity},
	}
	
	l.log(level, fmt.Sprintf("Security event: %s", eventType), entry)
}

func (l *Logger) LogBusiness(operation, tradingPlatform, symbol string, amount float64, userID string, details map[string]interface{}) {
	entry := &LogEntry{
		Operation: operation,
		TradingPlatform: tradingPlatform,
		Symbol:    symbol,
		Amount:    &amount,
		UserID:    userID,
		Component: "business",
		Fields:    details,
		Tags:      []string{"business", "trading"},
	}
	
	l.log(LevelInfo, fmt.Sprintf("Business: %s %s %.8f %s", operation, tradingPlatform, amount, symbol), entry)
}

// Core logging method
func (l *Logger) log(level LogLevel, message string, extraEntry *LogEntry) {
	if level < l.level {
		return
	}
	
	entry := &LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     logLevelNames[level],
		Message:   message,
		Service:   l.service,
		Version:   l.version,
	}
	
	// Merge extra entry fields
	if extraEntry != nil {
		if extraEntry.RequestID != "" {
			entry.RequestID = extraEntry.RequestID
		}
		if extraEntry.UserID != "" {
			entry.UserID = extraEntry.UserID
		}
		if extraEntry.SessionID != "" {
			entry.SessionID = extraEntry.SessionID
		}
		if extraEntry.TraceID != "" {
			entry.TraceID = extraEntry.TraceID
		}
		if extraEntry.SpanID != "" {
			entry.SpanID = extraEntry.SpanID
		}
		if extraEntry.Component != "" {
			entry.Component = extraEntry.Component
		}
		if extraEntry.Operation != "" {
			entry.Operation = extraEntry.Operation
		}
		if extraEntry.Duration != nil {
			entry.Duration = extraEntry.Duration
		}
		if extraEntry.Error != "" {
			entry.Error = extraEntry.Error
		}
		if extraEntry.StackTrace != "" {
			entry.StackTrace = extraEntry.StackTrace
		}
		if extraEntry.Fields != nil {
			entry.Fields = extraEntry.Fields
		}
		if extraEntry.Tags != nil {
			entry.Tags = extraEntry.Tags
		}
		if extraEntry.Method != "" {
			entry.Method = extraEntry.Method
		}
		if extraEntry.Path != "" {
			entry.Path = extraEntry.Path
		}
		if extraEntry.StatusCode != 0 {
			entry.StatusCode = extraEntry.StatusCode
		}
		if extraEntry.UserAgent != "" {
			entry.UserAgent = extraEntry.UserAgent
		}
		if extraEntry.RemoteIP != "" {
			entry.RemoteIP = extraEntry.RemoteIP
		}
		if extraEntry.Referer != "" {
			entry.Referer = extraEntry.Referer
		}
		if extraEntry.Query != "" {
			entry.Query = extraEntry.Query
		}
		if extraEntry.Table != "" {
			entry.Table = extraEntry.Table
		}
		if extraEntry.RowsAffected != nil {
			entry.RowsAffected = extraEntry.RowsAffected
		}
		if extraEntry.TradingPlatform != "" {
			entry.TradingPlatform = extraEntry.TradingPlatform
		}
		if extraEntry.Symbol != "" {
			entry.Symbol = extraEntry.Symbol
		}
		if extraEntry.Amount != nil {
			entry.Amount = extraEntry.Amount
		}
	}
	
	// Add logger fields
	l.mu.RLock()
	if len(l.fields) > 0 {
		if entry.Fields == nil {
			entry.Fields = make(map[string]interface{})
		}
		for k, v := range l.fields {
			entry.Fields[k] = v
		}
	}
	l.mu.RUnlock()
	
	// Apply hooks
	l.mu.RLock()
	for _, hook := range l.hooks {
		hook(entry)
	}
	l.mu.RUnlock()
	
	// Format and output
	var output string
	if l.format == FormatJSON {
		if jsonBytes, err := json.Marshal(entry); err == nil {
			output = string(jsonBytes) + "\n"
		} else {
			output = fmt.Sprintf("{\"level\":\"ERROR\",\"message\":\"Failed to marshal log entry: %v\"}\n", err)
		}
	} else {
		output = l.formatText(entry)
	}
	
	// Write to all outputs
	for _, writer := range l.outputs {
		writer.Write([]byte(output))
	}
}

func (l *Logger) formatText(entry *LogEntry) string {
	var parts []string
	
	parts = append(parts, entry.Timestamp.Format("2006-01-02T15:04:05.000Z"))
	parts = append(parts, fmt.Sprintf("[%s]", entry.Level))
	
	if entry.Component != "" {
		parts = append(parts, fmt.Sprintf("<%s>", entry.Component))
	}
	
	if entry.RequestID != "" {
		parts = append(parts, fmt.Sprintf("req=%s", entry.RequestID[:8]))
	}
	
	if entry.UserID != "" {
		parts = append(parts, fmt.Sprintf("user=%s", entry.UserID[:8]))
	}
	
	parts = append(parts, entry.Message)
	
	if entry.Duration != nil {
		parts = append(parts, fmt.Sprintf("duration=%.2fms", *entry.Duration))
	}
	
	if entry.Error != "" {
		parts = append(parts, fmt.Sprintf("error=%s", entry.Error))
	}
	
	return strings.Join(parts, " ") + "\n"
}

// Gin middleware for request logging
func (l *Logger) GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		
		// Generate request ID if not present
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
			c.Header("X-Request-ID", requestID)
		}
		
		// Set in context for other middleware
		c.Set("request_id", requestID)
		
		// Process request
		c.Next()
		
		// Log request completion
		duration := time.Since(start)
		statusCode := c.Writer.Status()
		
		var userID string
		if uid, exists := c.Get("user_id"); exists {
			if id, ok := uid.(uuid.UUID); ok {
				userID = id.String()
			}
		}
		
		if raw != "" {
			path = path + "?" + raw
		}
		
		l.LogHTTP(c.Request.Method, path, statusCode, duration, userID, requestID)
		
		// Log errors if any
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				l.ErrorWithError(err.Err, "Request error")
			}
		}
	}
}

// Helper functions
func getStackTrace() string {
	buf := make([]byte, 1024)
	for {
		n := runtime.Stack(buf, false)
		if n < len(buf) {
			return string(buf[:n])
		}
		buf = make([]byte, 2*len(buf))
	}
}