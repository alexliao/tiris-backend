package services

import (
	"context"
	"fmt"
	"time"

	"tiris-backend/internal/models"
	"tiris-backend/internal/repositories"

	"github.com/google/uuid"
)

// TransactionService handles transaction query business logic
type TransactionService struct {
	repos *repositories.Repositories
}

// NewTransactionService creates a new transaction service
func NewTransactionService(repos *repositories.Repositories) *TransactionService {
	return &TransactionService{
		repos: repos,
	}
}

// TransactionResponse represents transaction information in responses
type TransactionResponse struct {
	ID             uuid.UUID              `json:"id"`
	UserID         uuid.UUID              `json:"user_id"`
	TradingID      uuid.UUID              `json:"trading_id"`
	SubAccountID   uuid.UUID              `json:"sub_account_id"`
	Timestamp      string                 `json:"timestamp"`
	Direction      string                 `json:"direction"`
	Reason         string                 `json:"reason"`
	Amount         float64                `json:"amount"`
	ClosingBalance float64                `json:"closing_balance"`
	Price          *float64               `json:"price,omitempty"`
	QuoteSymbol    *string                `json:"quote_symbol,omitempty"`
	Info           map[string]interface{} `json:"info"`
}

// TransactionQueryRequest represents transaction query parameters
type TransactionQueryRequest struct {
	Direction *string    `form:"direction" binding:"omitempty,oneof=debit credit"`
	Reason    *string    `form:"reason"`
	StartDate *time.Time `form:"start_date" time_format:"2006-01-02T15:04:05Z07:00"`
	EndDate   *time.Time `form:"end_date" time_format:"2006-01-02T15:04:05Z07:00"`
	MinAmount *float64   `form:"min_amount" binding:"omitempty,min=0"`
	MaxAmount *float64   `form:"max_amount" binding:"omitempty,min=0"`
	Limit     int        `form:"limit" binding:"omitempty,min=1,max=1000"`
	Offset    int        `form:"offset" binding:"omitempty,min=0"`
}

// TransactionQueryResponse represents paginated transaction results
type TransactionQueryResponse struct {
	Transactions []*TransactionResponse `json:"transactions"`
	Total        int64                  `json:"total"`
	Limit        int                    `json:"limit"`
	Offset       int                    `json:"offset"`
	HasMore      bool                   `json:"has_more"`
}

// GetUserTransactions retrieves transactions for a user with filtering
func (s *TransactionService) GetUserTransactions(ctx context.Context, userID uuid.UUID, req *TransactionQueryRequest) (*TransactionQueryResponse, error) {
	// Set default pagination
	if req.Limit == 0 {
		req.Limit = 100
	}

	// Validate date range
	if req.StartDate != nil && req.EndDate != nil && req.StartDate.After(*req.EndDate) {
		return nil, fmt.Errorf("start date cannot be after end date")
	}

	// Validate amount range
	if req.MinAmount != nil && req.MaxAmount != nil && *req.MinAmount > *req.MaxAmount {
		return nil, fmt.Errorf("min amount cannot be greater than max amount")
	}

	// Create filters
	filters := repositories.TransactionFilters{
		Direction: req.Direction,
		Reason:    req.Reason,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		MinAmount: req.MinAmount,
		MaxAmount: req.MaxAmount,
		Limit:     req.Limit,
		Offset:    req.Offset,
	}

	// Query transactions
	transactions, total, err := s.repos.Transaction.GetByUserID(ctx, userID, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get user transactions: %w", err)
	}

	// Convert to response format
	var responses []*TransactionResponse
	for _, transaction := range transactions {
		responses = append(responses, s.convertToTransactionResponse(transaction))
	}

	return &TransactionQueryResponse{
		Transactions: responses,
		Total:        total,
		Limit:        req.Limit,
		Offset:       req.Offset,
		HasMore:      int64(req.Offset+req.Limit) < total,
	}, nil
}

// GetSubAccountTransactions retrieves transactions for a specific sub-account
func (s *TransactionService) GetSubAccountTransactions(ctx context.Context, userID, subAccountID uuid.UUID, req *TransactionQueryRequest) (*TransactionQueryResponse, error) {
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

	// Validate filters (same as GetUserTransactions)
	if req.StartDate != nil && req.EndDate != nil && req.StartDate.After(*req.EndDate) {
		return nil, fmt.Errorf("start date cannot be after end date")
	}
	if req.MinAmount != nil && req.MaxAmount != nil && *req.MinAmount > *req.MaxAmount {
		return nil, fmt.Errorf("min amount cannot be greater than max amount")
	}

	// Create filters
	filters := repositories.TransactionFilters{
		Direction: req.Direction,
		Reason:    req.Reason,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		MinAmount: req.MinAmount,
		MaxAmount: req.MaxAmount,
		Limit:     req.Limit,
		Offset:    req.Offset,
	}

	// Query transactions
	transactions, total, err := s.repos.Transaction.GetBySubAccountID(ctx, subAccountID, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get sub-account transactions: %w", err)
	}

	// Convert to response format
	var responses []*TransactionResponse
	for _, transaction := range transactions {
		responses = append(responses, s.convertToTransactionResponse(transaction))
	}

	return &TransactionQueryResponse{
		Transactions: responses,
		Total:        total,
		Limit:        req.Limit,
		Offset:       req.Offset,
		HasMore:      int64(req.Offset+req.Limit) < total,
	}, nil
}

// GetExchangeTransactions retrieves transactions for a specific exchange
func (s *TransactionService) GetExchangeTransactions(ctx context.Context, userID, exchangeID uuid.UUID, req *TransactionQueryRequest) (*TransactionQueryResponse, error) {
	// Verify user owns the exchange
	exchange, err := s.repos.Trading.GetByID(ctx, exchangeID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify exchange: %w", err)
	}
	if exchange == nil || exchange.UserID != userID {
		return nil, fmt.Errorf("trading platform not found")
	}

	// Set default pagination
	if req.Limit == 0 {
		req.Limit = 100
	}

	// Validate filters (same as GetUserTransactions)
	if req.StartDate != nil && req.EndDate != nil && req.StartDate.After(*req.EndDate) {
		return nil, fmt.Errorf("start date cannot be after end date")
	}
	if req.MinAmount != nil && req.MaxAmount != nil && *req.MinAmount > *req.MaxAmount {
		return nil, fmt.Errorf("min amount cannot be greater than max amount")
	}

	// Create filters
	filters := repositories.TransactionFilters{
		Direction: req.Direction,
		Reason:    req.Reason,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		MinAmount: req.MinAmount,
		MaxAmount: req.MaxAmount,
		Limit:     req.Limit,
		Offset:    req.Offset,
	}

	// Query transactions
	transactions, total, err := s.repos.Transaction.GetByTradingID(ctx, exchangeID, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get exchange transactions: %w", err)
	}

	// Convert to response format
	var responses []*TransactionResponse
	for _, transaction := range transactions {
		responses = append(responses, s.convertToTransactionResponse(transaction))
	}

	return &TransactionQueryResponse{
		Transactions: responses,
		Total:        total,
		Limit:        req.Limit,
		Offset:       req.Offset,
		HasMore:      int64(req.Offset+req.Limit) < total,
	}, nil
}

// GetTransaction retrieves a specific transaction by ID
func (s *TransactionService) GetTransaction(ctx context.Context, userID, transactionID uuid.UUID) (*TransactionResponse, error) {
	transaction, err := s.repos.Transaction.GetByID(ctx, transactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}
	if transaction == nil {
		return nil, fmt.Errorf("transaction not found")
	}

	// Check if transaction belongs to the user
	if transaction.UserID != userID {
		return nil, fmt.Errorf("transaction not found")
	}

	return s.convertToTransactionResponse(transaction), nil
}

// GetTransactionsByTimeRange retrieves transactions within a specific time range
func (s *TransactionService) GetTransactionsByTimeRange(ctx context.Context, userID uuid.UUID, startTime, endTime time.Time, req *TransactionQueryRequest) (*TransactionQueryResponse, error) {
	// Validate time range
	if startTime.After(endTime) {
		return nil, fmt.Errorf("start time cannot be after end time")
	}

	// Set default pagination
	if req.Limit == 0 {
		req.Limit = 100
	}

	// Validate amount range
	if req.MinAmount != nil && req.MaxAmount != nil && *req.MinAmount > *req.MaxAmount {
		return nil, fmt.Errorf("min amount cannot be greater than max amount")
	}

	// Create filters (override date filters with provided time range)
	filters := repositories.TransactionFilters{
		Direction: req.Direction,
		Reason:    req.Reason,
		StartDate: &startTime,
		EndDate:   &endTime,
		MinAmount: req.MinAmount,
		MaxAmount: req.MaxAmount,
		Limit:     req.Limit,
		Offset:    req.Offset,
	}

	// Query transactions by time range
	transactions, _, err := s.repos.Transaction.GetByTimeRange(ctx, startTime, endTime, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions by time range: %w", err)
	}

	// Filter to only include user's transactions
	var userTransactions []*models.Transaction
	for _, transaction := range transactions {
		if transaction.UserID == userID {
			userTransactions = append(userTransactions, transaction)
		}
	}

	// Convert to response format
	var responses []*TransactionResponse
	for _, transaction := range userTransactions {
		responses = append(responses, s.convertToTransactionResponse(transaction))
	}

	// Recalculate total for user's transactions only
	userTotal := int64(len(userTransactions))

	return &TransactionQueryResponse{
		Transactions: responses,
		Total:        userTotal,
		Limit:        req.Limit,
		Offset:       req.Offset,
		HasMore:      int64(req.Offset+req.Limit) < userTotal,
	}, nil
}

// ListAllTransactions lists all transactions with pagination (admin only)
func (s *TransactionService) ListAllTransactions(ctx context.Context, req *TransactionQueryRequest) (*TransactionQueryResponse, error) {
	// Set default pagination
	if req.Limit == 0 {
		req.Limit = 100
	}

	// Validate filters
	if req.StartDate != nil && req.EndDate != nil && req.StartDate.After(*req.EndDate) {
		return nil, fmt.Errorf("start date cannot be after end date")
	}
	if req.MinAmount != nil && req.MaxAmount != nil && *req.MinAmount > *req.MaxAmount {
		return nil, fmt.Errorf("min amount cannot be greater than max amount")
	}

	// For admin queries, we'll use a time range approach to get all transactions
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
	filters := repositories.TransactionFilters{
		Direction: req.Direction,
		Reason:    req.Reason,
		StartDate: &startTime,
		EndDate:   &endTime,
		MinAmount: req.MinAmount,
		MaxAmount: req.MaxAmount,
		Limit:     req.Limit,
		Offset:    req.Offset,
	}

	// Query all transactions in time range
	transactions, total, err := s.repos.Transaction.GetByTimeRange(ctx, startTime, endTime, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list all transactions: %w", err)
	}

	// Convert to response format
	var responses []*TransactionResponse
	for _, transaction := range transactions {
		responses = append(responses, s.convertToTransactionResponse(transaction))
	}

	return &TransactionQueryResponse{
		Transactions: responses,
		Total:        total,
		Limit:        req.Limit,
		Offset:       req.Offset,
		HasMore:      int64(req.Offset+req.Limit) < total,
	}, nil
}

// GetTransactionByID retrieves transaction by ID (admin only)
func (s *TransactionService) GetTransactionByID(ctx context.Context, transactionID uuid.UUID) (*TransactionResponse, error) {
	transaction, err := s.repos.Transaction.GetByID(ctx, transactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}
	if transaction == nil {
		return nil, fmt.Errorf("transaction not found")
	}

	return s.convertToTransactionResponse(transaction), nil
}

// convertToTransactionResponse converts a transaction model to response format
func (s *TransactionService) convertToTransactionResponse(transaction *models.Transaction) *TransactionResponse {
	var info map[string]interface{}
	if len(transaction.Info) > 0 {
		info = transaction.Info
	} else {
		info = make(map[string]interface{})
	}

	return &TransactionResponse{
		ID:             transaction.ID,
		UserID:         transaction.UserID,
		TradingID:      transaction.TradingID,
		SubAccountID:   transaction.SubAccountID,
		Timestamp:      transaction.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
		Direction:      transaction.Direction,
		Reason:         transaction.Reason,
		Amount:         transaction.Amount,
		ClosingBalance: transaction.ClosingBalance,
		Price:          transaction.Price,
		QuoteSymbol:    transaction.QuoteSymbol,
		Info:           info,
	}
}
