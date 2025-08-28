package services

import (
	"context"
	"fmt"

	"tiris-backend/internal/models"
	"tiris-backend/internal/repositories"

	"github.com/google/uuid"
)

// SubAccountService handles sub-account business logic
type SubAccountService struct {
	repos *repositories.Repositories
}

// NewSubAccountService creates a new sub-account service
func NewSubAccountService(repos *repositories.Repositories) *SubAccountService {
	return &SubAccountService{
		repos: repos,
	}
}

// SubAccountResponse represents sub-account information in responses
type SubAccountResponse struct {
	ID         uuid.UUID              `json:"id"`
	UserID     uuid.UUID              `json:"user_id"`
	TradingID  uuid.UUID              `json:"trading_id"`
	Name       string                 `json:"name"`
	Symbol     string                 `json:"symbol"`
	Balance    float64                `json:"balance"`
	Info       map[string]interface{} `json:"info"`
	CreatedAt  string                 `json:"created_at"`
	UpdatedAt  string                 `json:"updated_at"`
}

// CreateSubAccountRequest represents sub-account creation request
type CreateSubAccountRequest struct {
	TradingID  uuid.UUID `json:"trading_id" binding:"required" example:"453f0347-3959-49de-8e3f-1cf7c8e0827c"`
	Name       string    `json:"name" binding:"required,min=1,max=100" example:"BTC Trading Account"`
	Symbol     string    `json:"symbol" binding:"required,min=1,max=20" example:"BTC/USDT"`
}

// UpdateSubAccountRequest represents sub-account update request
type UpdateSubAccountRequest struct {
	Name    *string  `json:"name,omitempty" binding:"omitempty,min=1,max=100" example:"ETH Trading Account"`
	Symbol  *string  `json:"symbol,omitempty" binding:"omitempty,min=1,max=20" example:"ETH/USD"`
	Balance *float64 `json:"balance,omitempty" binding:"omitempty,min=0" example:"1250.75"`
}

// UpdateBalanceRequest represents balance update request
type UpdateBalanceRequest struct {
	Amount    float64                `json:"amount" binding:"required" example:"500.25"`
	Direction string                 `json:"direction" binding:"required,oneof=credit debit" example:"credit"`
	Reason    string                 `json:"reason" binding:"required,min=1,max=255" example:"Successful BTC/USDT trade profit"`
	Info      map[string]interface{} `json:"info,omitempty"`
}

// CreateSubAccount creates a new sub-account
func (s *SubAccountService) CreateSubAccount(ctx context.Context, userID uuid.UUID, req *CreateSubAccountRequest) (*SubAccountResponse, error) {
	// Verify user owns the trading
	trading, err := s.repos.Trading.GetByID(ctx, req.TradingID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify trading: %w", err)
	}
	if trading == nil || trading.UserID != userID {
		return nil, fmt.Errorf("trading not found")
	}

	// Create info map with metadata
	infoMap := map[string]interface{}{
		"created_by":    "api",
		"api_version":   "v1",
		"trading_type": trading.Type,
	}

	// Create sub-account model
	subAccount := &models.SubAccount{
		ID:         uuid.New(),
		UserID:     userID,
		TradingID: req.TradingID,
		Name:       req.Name,
		Symbol:     req.Symbol,
		Balance:    0.0, // Start with zero balance
		Info:       models.JSON(infoMap),
	}

	// Save to database - let database constraints handle uniqueness validation
	if err := s.repos.SubAccount.Create(ctx, subAccount); err != nil {
		// Check for specific constraint violations and provide user-friendly messages
		if constraintMsg := getSpecificConstraintViolation(err); constraintMsg != "" {
			return nil, fmt.Errorf(constraintMsg)
		}
		return nil, fmt.Errorf("failed to create sub-account: %w", err)
	}

	return s.convertToSubAccountResponse(subAccount), nil
}

// GetUserSubAccounts retrieves all sub-accounts for a user
func (s *SubAccountService) GetUserSubAccounts(ctx context.Context, userID uuid.UUID, tradingID *uuid.UUID) ([]*SubAccountResponse, error) {
	// If tradingID is provided, verify user owns it
	if tradingID != nil {
		trading, err := s.repos.Trading.GetByID(ctx, *tradingID)
		if err != nil {
			return nil, fmt.Errorf("failed to verify trading: %w", err)
		}
		if trading == nil || trading.UserID != userID {
			return nil, fmt.Errorf("trading not found")
		}
	}

	subAccounts, err := s.repos.SubAccount.GetByUserID(ctx, userID, tradingID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user sub-accounts: %w", err)
	}

	var responses []*SubAccountResponse
	for _, subAccount := range subAccounts {
		responses = append(responses, s.convertToSubAccountResponse(subAccount))
	}

	return responses, nil
}

// GetSubAccount retrieves a specific sub-account by ID (must belong to user)
func (s *SubAccountService) GetSubAccount(ctx context.Context, userID, subAccountID uuid.UUID) (*SubAccountResponse, error) {
	subAccount, err := s.repos.SubAccount.GetByID(ctx, subAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sub-account: %w", err)
	}
	if subAccount == nil {
		return nil, fmt.Errorf("sub-account not found")
	}

	// Check if sub-account belongs to the user
	if subAccount.UserID != userID {
		return nil, fmt.Errorf("sub-account not found")
	}

	return s.convertToSubAccountResponse(subAccount), nil
}

// UpdateSubAccount updates an existing sub-account
func (s *SubAccountService) UpdateSubAccount(ctx context.Context, userID, subAccountID uuid.UUID, req *UpdateSubAccountRequest) (*SubAccountResponse, error) {
	// Get existing sub-account
	subAccount, err := s.repos.SubAccount.GetByID(ctx, subAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sub-account: %w", err)
	}
	if subAccount == nil {
		return nil, fmt.Errorf("sub-account not found")
	}

	// Check if sub-account belongs to the user
	if subAccount.UserID != userID {
		return nil, fmt.Errorf("sub-account not found")
	}

	// Update fields if provided - let database constraints handle uniqueness validation
	if req.Name != nil {
		subAccount.Name = *req.Name
	}

	if req.Symbol != nil {
		subAccount.Symbol = *req.Symbol
	}

	// Note: Direct balance updates should use UpdateBalance method for proper logging
	if req.Balance != nil {
		subAccount.Balance = *req.Balance
	}

	// Save updated sub-account
	if err := s.repos.SubAccount.Update(ctx, subAccount); err != nil {
		// Check for specific constraint violations and provide user-friendly messages
		if constraintMsg := getSpecificConstraintViolation(err); constraintMsg != "" {
			return nil, fmt.Errorf(constraintMsg)
		}
		return nil, fmt.Errorf("failed to update sub-account: %w", err)
	}

	return s.convertToSubAccountResponse(subAccount), nil
}

// UpdateBalance updates sub-account balance with proper logging
func (s *SubAccountService) UpdateBalance(ctx context.Context, userID, subAccountID uuid.UUID, req *UpdateBalanceRequest) (*SubAccountResponse, error) {
	// Verify sub-account belongs to user
	subAccount, err := s.repos.SubAccount.GetByID(ctx, subAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sub-account: %w", err)
	}
	if subAccount == nil {
		return nil, fmt.Errorf("sub-account not found")
	}

	if subAccount.UserID != userID {
		return nil, fmt.Errorf("sub-account not found")
	}

	// Calculate new balance
	var newBalance float64
	switch req.Direction {
	case "credit":
		newBalance = subAccount.Balance + req.Amount
	case "debit":
		newBalance = subAccount.Balance - req.Amount
		if newBalance < 0 {
			return nil, fmt.Errorf("insufficient balance")
		}
	default:
		return nil, fmt.Errorf("invalid direction")
	}

	// Use repository method for atomic balance update with logging
	_, err = s.repos.SubAccount.UpdateBalance(ctx, subAccountID, newBalance, req.Amount, req.Direction, req.Reason, req.Info)
	if err != nil {
		return nil, fmt.Errorf("failed to update balance: %w", err)
	}

	// Return updated sub-account
	return s.GetSubAccount(ctx, userID, subAccountID)
}

// DeleteSubAccount deletes a sub-account (soft delete)
func (s *SubAccountService) DeleteSubAccount(ctx context.Context, userID, subAccountID uuid.UUID) error {
	// Get existing sub-account to verify ownership
	subAccount, err := s.repos.SubAccount.GetByID(ctx, subAccountID)
	if err != nil {
		return fmt.Errorf("failed to get sub-account: %w", err)
	}
	if subAccount == nil {
		return fmt.Errorf("sub-account not found")
	}

	// Check if sub-account belongs to the user
	if subAccount.UserID != userID {
		return fmt.Errorf("sub-account not found")
	}

	// Check if sub-account has balance
	if subAccount.Balance > 0 {
		return fmt.Errorf("cannot delete sub-account with positive balance")
	}

	// Soft delete the sub-account
	if err := s.repos.SubAccount.Delete(ctx, subAccountID); err != nil {
		return fmt.Errorf("failed to delete sub-account: %w", err)
	}

	return nil
}

// GetSubAccountsBySymbol retrieves sub-accounts by symbol
func (s *SubAccountService) GetSubAccountsBySymbol(ctx context.Context, userID uuid.UUID, symbol string) ([]*SubAccountResponse, error) {
	subAccounts, err := s.repos.SubAccount.GetBySymbol(ctx, userID, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get sub-accounts by symbol: %w", err)
	}

	var responses []*SubAccountResponse
	for _, subAccount := range subAccounts {
		responses = append(responses, s.convertToSubAccountResponse(subAccount))
	}

	return responses, nil
}

// convertToSubAccountResponse converts a sub-account model to response format
func (s *SubAccountService) convertToSubAccountResponse(subAccount *models.SubAccount) *SubAccountResponse {
	var info map[string]interface{}
	if len(subAccount.Info) > 0 {
		info = subAccount.Info
	} else {
		info = make(map[string]interface{})
	}

	return &SubAccountResponse{
		ID:         subAccount.ID,
		UserID:     subAccount.UserID,
		TradingID: subAccount.TradingID,
		Name:       subAccount.Name,
		Symbol:     subAccount.Symbol,
		Balance:    subAccount.Balance,
		Info:       info,
		CreatedAt:  subAccount.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:  subAccount.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
