package api

import "time"

// SuccessResponse represents a successful API response
type SuccessResponse struct {
	Success  bool             `json:"success"`
	Data     interface{}      `json:"data"`
	Metadata ResponseMetadata `json:"metadata"`
}

// ErrorResponse represents an error API response
type ErrorResponse struct {
	Success  bool             `json:"success"`
	Error    ErrorDetail      `json:"error"`
	Metadata ResponseMetadata `json:"metadata"`
}

// ErrorDetail represents error details
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// ResponseMetadata represents response metadata
type ResponseMetadata struct {
	Timestamp string `json:"timestamp"`
	TraceID   string `json:"trace_id,omitempty"`
}

// PaginationMetadata represents pagination information
type PaginationMetadata struct {
	Total      int64 `json:"total"`
	Limit      int   `json:"limit"`
	Offset     int   `json:"offset"`
	HasMore    bool  `json:"has_more"`
	NextOffset *int  `json:"next_offset,omitempty"`
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	Success    bool                `json:"success"`
	Data       interface{}         `json:"data"`
	Pagination *PaginationMetadata `json:"pagination,omitempty"`
	Metadata   ResponseMetadata    `json:"metadata"`
}

// CreateSuccessResponse creates a success response
func CreateSuccessResponse(data interface{}, traceID string) SuccessResponse {
	return SuccessResponse{
		Success: true,
		Data:    data,
		Metadata: ResponseMetadata{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			TraceID:   traceID,
		},
	}
}

// CreateErrorResponse creates an error response
func CreateErrorResponse(code, message, details, traceID string) ErrorResponse {
	return ErrorResponse{
		Success: false,
		Error: ErrorDetail{
			Code:    code,
			Message: message,
			Details: details,
		},
		Metadata: ResponseMetadata{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			TraceID:   traceID,
		},
	}
}

// CreatePaginatedResponse creates a paginated response
func CreatePaginatedResponse(data interface{}, pagination *PaginationMetadata, traceID string) PaginatedResponse {
	return PaginatedResponse{
		Success:    true,
		Data:       data,
		Pagination: pagination,
		Metadata: ResponseMetadata{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			TraceID:   traceID,
		},
	}
}

// HealthCheckResponse represents health check response
type HealthCheckResponse struct {
	Status    string                 `json:"status"`
	Checks    map[string]string      `json:"checks,omitempty"`
	Timestamp string                 `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}
