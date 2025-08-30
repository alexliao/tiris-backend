package services

import (
	"context"
	"fmt"
	"time"

	"tiris-backend/internal/models"
	"tiris-backend/internal/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TradingLogService handles trading log business logic
type TradingLogService struct {
	repos     *repositories.Repositories
	db        *gorm.DB
	processor *TradingLogProcessor
}

// NewTradingLogService creates a new trading log service
func NewTradingLogService(repos *repositories.Repositories, db *gorm.DB) *TradingLogService {
	return &TradingLogService{
		repos:     repos,
		db:        db,
		processor: NewTradingLogProcessor(repos),
	}
}

// TradingLogResponse represents trading log information in responses
type TradingLogResponse struct {
	ID            uuid.UUID              `json:"id"`
	UserID        uuid.UUID              `json:"user_id"`
	TradingID     uuid.UUID              `json:"trading_id"`
	SubAccountID  *uuid.UUID             `json:"sub_account_id,omitempty"`
	TransactionID *uuid.UUID             `json:"transaction_id,omitempty"`
	Timestamp     string                 `json:"timestamp"`
	EventTime     *string                `json:"event_time,omitempty"`
	Type          string                 `json:"type"`
	Source        string                 `json:"source"`
	Message       string                 `json:"message"`
	Info          map[string]interface{} `json:"info"`
}

// CreateTradingLogRequest represents trading log creation request
// @Description Request for creating a new trading log entry. The 'info' field structure depends on the 'type' value:
// @Description - For types 'long', 'short', 'stop_loss': Must use TradingLogInfo structure
// @Description - For types 'deposit', 'withdraw': Must use DepositWithdrawInfo structure  
// @Description - For other types: Can use any object structure
type CreateTradingLogRequest struct {
	TradingID     uuid.UUID              `json:"trading_id" binding:"required" example:"453f0347-3959-49de-8e3f-1cf7c8e0827c" description:"ID of the trading where the trading activity occurred"`
	SubAccountID  *uuid.UUID             `json:"sub_account_id,omitempty" example:"b4e006d0-1069-4ef4-b33f-7690af4929f4" description:"Optional sub-account ID (used for some trading log types)"`
	TransactionID *uuid.UUID             `json:"transaction_id,omitempty" example:"1a098613-e738-447d-b921-74c3594df3a5" description:"Optional transaction ID for linking to specific transactions"`
	EventTime     *time.Time             `json:"event_time,omitempty" example:"2024-01-15T10:30:00Z" description:"Logical timestamp when the trading event occurred. If not provided, defaults to NULL. For live trading, this should match current time. For backtesting, this represents the historical time when the event logically occurred."`
	Type          string                 `json:"type" binding:"required,min=1,max=50" example:"long" enums:"long,short,stop_loss,deposit,withdraw,trade_execution,api_call,system_event,error,custom" description:"Type of trading log entry. Business logic types (long, short, stop_loss, deposit, withdraw) require specific 'info' field structures and trigger automatic financial calculations"`
	Source        string                 `json:"source" binding:"required,oneof=manual bot" example:"bot" description:"Source of the trading log entry"`
	Message       string                 `json:"message" binding:"required,min=1" example:"Successfully executed BUY order for 0.5 BTC at $42,500" description:"Human-readable description of the trading activity"`
	Info          map[string]interface{} `json:"info,omitempty" description:"Type-specific structured data. Required structure depends on the 'type' field: long/short/stop_loss: Use TradingLogInfo schema, deposit/withdraw: Use DepositWithdrawInfo schema, other types: Any object structure"`
}

// CreateLongTradingLogExample shows example structure for long trading log requests
// @Description Example request structure for creating a long position trading log
type CreateLongTradingLogExample struct {
	TradingID  uuid.UUID              `json:"trading_id" example:"453f0347-3959-49de-8e3f-1cf7c8e0827c"`
	Type       string                 `json:"type" example:"long"`
	Source     string                 `json:"source" example:"bot"`
	Message    string                 `json:"message" example:"ETH long position opened"`
	Info       TradingLogInfo         `json:"info"`
}

// CreateDepositTradingLogExample shows example structure for deposit trading log requests  
// @Description Example request structure for creating a deposit trading log
type CreateDepositTradingLogExample struct {
	TradingID  uuid.UUID              `json:"trading_id" example:"453f0347-3959-49de-8e3f-1cf7c8e0827c"`
	Type       string                 `json:"type" example:"deposit"`
	Source     string                 `json:"source" example:"api"`
	Message    string                 `json:"message" example:"USDT deposit to account"`
	Info       DepositWithdrawInfo    `json:"info"`
}

// TradingLogQueryRequest represents trading log query parameters
type TradingLogQueryRequest struct {
	Type      *string    `form:"type" example:"trade_execution"`
	Source    *string    `form:"source" binding:"omitempty,oneof=manual bot" example:"bot"`
	StartDate *time.Time `form:"start_date" time_format:"2006-01-02T15:04:05Z07:00" example:"2025-08-01T00:00:00Z"`
	EndDate   *time.Time `form:"end_date" time_format:"2006-01-02T15:04:05Z07:00" example:"2025-08-31T23:59:59Z"`
	Limit     int        `form:"limit" binding:"omitempty,min=1,max=1000" example:"50"`
	Offset    int        `form:"offset" binding:"omitempty,min=0" example:"0"`
}

// TradingLogQueryResponse represents paginated trading log results
type TradingLogQueryResponse struct {
	TradingLogs []*TradingLogResponse `json:"trading_logs"`
	Total       int64                 `json:"total"`
	Limit       int                   `json:"limit"`
	Offset      int                   `json:"offset"`
	HasMore     bool                  `json:"has_more"`
}

// CreateTradingLog creates a new trading log entry with business logic processing
func (s *TradingLogService) CreateTradingLog(ctx context.Context, userID uuid.UUID, req *CreateTradingLogRequest) (*TradingLogResponse, error) {
	// Process the trading log using the processor
	result, err := s.processor.ProcessTradingLog(ctx, s.db, userID, req)
	if err != nil {
		return nil, fmt.Errorf("failed to process trading log: %w", err)
	}

	// Convert the created trading log to response format
	response := s.convertToTradingLogResponse(result.TradingLogRecord)

	// Add processing summary to the response if business logic was applied
	if len(result.CreatedTransactions) > 0 {
		// Add metadata about the processed transactions
		if response.Info == nil {
			response.Info = make(map[string]interface{})
		}
		response.Info["processed_transactions"] = len(result.CreatedTransactions)
		response.Info["updated_accounts"] = len(result.UpdatedSubAccounts)

		// Add transaction IDs for audit trail
		var transactionIDs []string
		for _, tx := range result.CreatedTransactions {
			transactionIDs = append(transactionIDs, tx.ID.String())
		}
		response.Info["transaction_ids"] = transactionIDs
	}

	return response, nil
}

// GetUserTradingLogs retrieves trading logs for a user with filtering
func (s *TradingLogService) GetUserTradingLogs(ctx context.Context, userID uuid.UUID, req *TradingLogQueryRequest) (*TradingLogQueryResponse, error) {
	// Set default pagination
	if req.Limit == 0 {
		req.Limit = 100
	}

	// Validate date range
	if req.StartDate != nil && req.EndDate != nil && req.StartDate.After(*req.EndDate) {
		return nil, fmt.Errorf("start date cannot be after end date")
	}

	// Create filters
	filters := repositories.TradingLogFilters{
		Type:      req.Type,
		Source:    req.Source,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		Limit:     req.Limit,
		Offset:    req.Offset,
	}

	// Query trading logs
	tradingLogs, total, err := s.repos.TradingLog.GetByUserID(ctx, userID, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get user trading logs: %w", err)
	}

	// Convert to response format
	var responses []*TradingLogResponse
	for _, tradingLog := range tradingLogs {
		responses = append(responses, s.convertToTradingLogResponse(tradingLog))
	}

	return &TradingLogQueryResponse{
		TradingLogs: responses,
		Total:       total,
		Limit:       req.Limit,
		Offset:      req.Offset,
		HasMore:     int64(req.Offset+req.Limit) < total,
	}, nil
}

// GetSubAccountTradingLogs retrieves trading logs for a specific sub-account
func (s *TradingLogService) GetSubAccountTradingLogs(ctx context.Context, userID, subAccountID uuid.UUID, req *TradingLogQueryRequest) (*TradingLogQueryResponse, error) {
	// Verify user owns the sub-account
	subAccount, err := s.repos.SubAccount.GetByID(ctx, subAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify sub-account: %w", err)
	}
	if subAccount == nil || subAccount.UserID != userID {
		return nil, fmt.Errorf("sub-account not found")
	}

	// Set default pagination
	if req.Limit == 0 {
		req.Limit = 100
	}

	// Validate date range
	if req.StartDate != nil && req.EndDate != nil && req.StartDate.After(*req.EndDate) {
		return nil, fmt.Errorf("start date cannot be after end date")
	}

	// Create filters
	filters := repositories.TradingLogFilters{
		Type:      req.Type,
		Source:    req.Source,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		Limit:     req.Limit,
		Offset:    req.Offset,
	}

	// Query trading logs
	tradingLogs, total, err := s.repos.TradingLog.GetBySubAccountID(ctx, subAccountID, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get sub-account trading logs: %w", err)
	}

	// Convert to response format
	var responses []*TradingLogResponse
	for _, tradingLog := range tradingLogs {
		responses = append(responses, s.convertToTradingLogResponse(tradingLog))
	}

	return &TradingLogQueryResponse{
		TradingLogs: responses,
		Total:       total,
		Limit:       req.Limit,
		Offset:      req.Offset,
		HasMore:     int64(req.Offset+req.Limit) < total,
	}, nil
}

// GetTradingLogs retrieves trading logs for a specific trading
func (s *TradingLogService) GetTradingLogs(ctx context.Context, userID, tradingID uuid.UUID, req *TradingLogQueryRequest) (*TradingLogQueryResponse, error) {
	// Verify user owns the trading
	trading, err := s.repos.Trading.GetByID(ctx, tradingID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify trading: %w", err)
	}
	if trading == nil || trading.UserID != userID {
		return nil, fmt.Errorf("trading not found")
	}

	// Set default pagination
	if req.Limit == 0 {
		req.Limit = 100
	}

	// Validate date range
	if req.StartDate != nil && req.EndDate != nil && req.StartDate.After(*req.EndDate) {
		return nil, fmt.Errorf("start date cannot be after end date")
	}

	// Create filters
	filters := repositories.TradingLogFilters{
		Type:      req.Type,
		Source:    req.Source,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		Limit:     req.Limit,
		Offset:    req.Offset,
	}

	// Query trading logs
	tradingLogs, total, err := s.repos.TradingLog.GetByTradingID(ctx, tradingID, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get trading logs: %w", err)
	}

	// Convert to response format
	var responses []*TradingLogResponse
	for _, tradingLog := range tradingLogs {
		responses = append(responses, s.convertToTradingLogResponse(tradingLog))
	}

	return &TradingLogQueryResponse{
		TradingLogs: responses,
		Total:       total,
		Limit:       req.Limit,
		Offset:      req.Offset,
		HasMore:     int64(req.Offset+req.Limit) < total,
	}, nil
}

// GetTradingLog retrieves a specific trading log by ID
func (s *TradingLogService) GetTradingLog(ctx context.Context, userID, tradingLogID uuid.UUID) (*TradingLogResponse, error) {
	tradingLog, err := s.repos.TradingLog.GetByID(ctx, tradingLogID)
	if err != nil {
		return nil, fmt.Errorf("failed to get trading log: %w", err)
	}
	if tradingLog == nil {
		return nil, fmt.Errorf("trading log not found")
	}

	// Check if trading log belongs to the user
	if tradingLog.UserID != userID {
		return nil, fmt.Errorf("trading log not found")
	}

	return s.convertToTradingLogResponse(tradingLog), nil
}

// GetTradingLogsByTimeRange retrieves trading logs within a specific time range
func (s *TradingLogService) GetTradingLogsByTimeRange(ctx context.Context, userID uuid.UUID, startTime, endTime time.Time, req *TradingLogQueryRequest) (*TradingLogQueryResponse, error) {
	// Validate time range
	if startTime.After(endTime) {
		return nil, fmt.Errorf("start time cannot be after end time")
	}

	// Set default pagination
	if req.Limit == 0 {
		req.Limit = 100
	}

	// Create filters (override date filters with provided time range)
	filters := repositories.TradingLogFilters{
		Type:      req.Type,
		Source:    req.Source,
		StartDate: &startTime,
		EndDate:   &endTime,
		Limit:     req.Limit,
		Offset:    req.Offset,
	}

	// Query trading logs by time range
	tradingLogs, _, err := s.repos.TradingLog.GetByTimeRange(ctx, startTime, endTime, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get trading logs by time range: %w", err)
	}

	// Filter to only include user's trading logs
	var userTradingLogs []*models.TradingLog
	for _, tradingLog := range tradingLogs {
		if tradingLog.UserID == userID {
			userTradingLogs = append(userTradingLogs, tradingLog)
		}
	}

	// Convert to response format
	var responses []*TradingLogResponse
	for _, tradingLog := range userTradingLogs {
		responses = append(responses, s.convertToTradingLogResponse(tradingLog))
	}

	// Recalculate total for user's trading logs only
	userTotal := int64(len(userTradingLogs))

	return &TradingLogQueryResponse{
		TradingLogs: responses,
		Total:       userTotal,
		Limit:       req.Limit,
		Offset:      req.Offset,
		HasMore:     int64(req.Offset+req.Limit) < userTotal,
	}, nil
}

// DeleteTradingLog deletes a trading log (soft delete)
func (s *TradingLogService) DeleteTradingLog(ctx context.Context, userID, tradingLogID uuid.UUID) error {
	// Get existing trading log to verify ownership
	tradingLog, err := s.repos.TradingLog.GetByID(ctx, tradingLogID)
	if err != nil {
		return fmt.Errorf("failed to get trading log: %w", err)
	}
	if tradingLog == nil {
		return fmt.Errorf("trading log not found")
	}

	// Check if trading log belongs to the user
	if tradingLog.UserID != userID {
		return fmt.Errorf("trading log not found")
	}

	// Only allow deletion of manual logs
	if tradingLog.Source != "manual" {
		return fmt.Errorf("cannot delete bot-generated trading logs")
	}

	// Delete the trading log
	if err := s.repos.TradingLog.Delete(ctx, tradingLogID); err != nil {
		return fmt.Errorf("failed to delete trading log: %w", err)
	}

	return nil
}

// ListAllTradingLogs lists all trading logs with pagination (admin only)
func (s *TradingLogService) ListAllTradingLogs(ctx context.Context, req *TradingLogQueryRequest) (*TradingLogQueryResponse, error) {
	// Set default pagination
	if req.Limit == 0 {
		req.Limit = 100
	}

	// Validate date range
	if req.StartDate != nil && req.EndDate != nil && req.StartDate.After(*req.EndDate) {
		return nil, fmt.Errorf("start date cannot be after end date")
	}

	// For admin queries, we'll use a time range approach to get all trading logs
	// Use a very broad time range if no specific dates provided
	startTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Now().UTC()

	if req.StartDate != nil {
		startTime = *req.StartDate
	}
	if req.EndDate != nil {
		endTime = *req.EndDate
	}

	// Create filters
	filters := repositories.TradingLogFilters{
		Type:      req.Type,
		Source:    req.Source,
		StartDate: &startTime,
		EndDate:   &endTime,
		Limit:     req.Limit,
		Offset:    req.Offset,
	}

	// Query all trading logs in time range
	tradingLogs, total, err := s.repos.TradingLog.GetByTimeRange(ctx, startTime, endTime, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list all trading logs: %w", err)
	}

	// Convert to response format
	var responses []*TradingLogResponse
	for _, tradingLog := range tradingLogs {
		responses = append(responses, s.convertToTradingLogResponse(tradingLog))
	}

	return &TradingLogQueryResponse{
		TradingLogs: responses,
		Total:       total,
		Limit:       req.Limit,
		Offset:      req.Offset,
		HasMore:     int64(req.Offset+req.Limit) < total,
	}, nil
}

// GetTradingLogByID retrieves trading log by ID (admin only)
func (s *TradingLogService) GetTradingLogByID(ctx context.Context, tradingLogID uuid.UUID) (*TradingLogResponse, error) {
	tradingLog, err := s.repos.TradingLog.GetByID(ctx, tradingLogID)
	if err != nil {
		return nil, fmt.Errorf("failed to get trading log: %w", err)
	}
	if tradingLog == nil {
		return nil, fmt.Errorf("trading log not found")
	}

	return s.convertToTradingLogResponse(tradingLog), nil
}

// convertToTradingLogResponse converts a trading log model to response format
func (s *TradingLogService) convertToTradingLogResponse(tradingLog *models.TradingLog) *TradingLogResponse {
	var info map[string]interface{}
	if len(tradingLog.Info) > 0 {
		info = tradingLog.Info
	} else {
		info = make(map[string]interface{})
	}

	var eventTimeStr *string
	if tradingLog.EventTime != nil {
		eventTime := tradingLog.EventTime.Format("2006-01-02T15:04:05Z07:00")
		eventTimeStr = &eventTime
	}

	return &TradingLogResponse{
		ID:            tradingLog.ID,
		UserID:        tradingLog.UserID,
		TradingID:     tradingLog.TradingID,
		SubAccountID:  tradingLog.SubAccountID,
		TransactionID: tradingLog.TransactionID,
		Timestamp:     tradingLog.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
		EventTime:     eventTimeStr,
		Type:          tradingLog.Type,
		Source:        tradingLog.Source,
		Message:       tradingLog.Message,
		Info:          info,
	}
}
