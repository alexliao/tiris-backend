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
	ExchangeID uuid.UUID              `json:"exchange_id"`
	Name       string                 `json:"name"`
	Symbol     string                 `json:"symbol"`
	Balance    float64                `json:"balance"`
	Info       map[string]interface{} `json:"info"`
	CreatedAt  string                 `json:"created_at"`
	UpdatedAt  string                 `json:"updated_at"`
}

// CreateSubAccountRequest represents sub-account creation request
type CreateSubAccountRequest struct {
	ExchangeID uuid.UUID `json:"exchange_id" binding:"required"`
	Name       string    `json:"name" binding:"required,min=1,max=100"`
	Symbol     string    `json:"symbol" binding:"required,min=1,max=20"`
}

// UpdateSubAccountRequest represents sub-account update request
type UpdateSubAccountRequest struct {
	Name    *string  `json:"name,omitempty" binding:"omitempty,min=1,max=100"`
	Symbol  *string  `json:"symbol,omitempty" binding:"omitempty,min=1,max=20"`
	Balance *float64 `json:"balance,omitempty" binding:"omitempty,min=0"`
}

// UpdateBalanceRequest represents balance update request
type UpdateBalanceRequest struct {
	Amount    float64                `json:"amount" binding:"required"`
	Direction string                 `json:"direction" binding:"required,oneof=credit debit"`
	Reason    string                 `json:"reason" binding:"required,min=1,max=255"`
	Info      map[string]interface{} `json:"info,omitempty"`
}

// CreateSubAccount creates a new sub-account
func (s *SubAccountService) CreateSubAccount(ctx context.Context, userID uuid.UUID, req *CreateSubAccountRequest) (*SubAccountResponse, error) {
	// Verify user owns the exchange
	exchange, err := s.repos.Exchange.GetByID(ctx, req.ExchangeID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify exchange: %w", err)
	}
	if exchange == nil || exchange.UserID != userID {
		return nil, fmt.Errorf("exchange not found")
	}

	// Check if sub-account name is unique for this user+exchange combination
	existingSubAccounts, err := s.repos.SubAccount.GetByUserID(ctx, userID, &req.ExchangeID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing sub-accounts: %w", err)
	}

	for _, subAccount := range existingSubAccounts {
		if subAccount.Name == req.Name {
			return nil, fmt.Errorf("sub-account name already exists for this exchange")
		}
	}

	// Create info map with metadata
	infoMap := map[string]interface{}{
		"created_by":    "api",
		"api_version":   "v1",
		"exchange_type": exchange.Type,
	}

	// Create sub-account model
	subAccount := &models.SubAccount{
		ID:         uuid.New(),
		UserID:     userID,
		ExchangeID: req.ExchangeID,
		Name:       req.Name,
		Symbol:     req.Symbol,
		Balance:    0.0, // Start with zero balance
		Info:       models.JSON(infoMap),
	}

	// Save to database
	if err := s.repos.SubAccount.Create(ctx, subAccount); err != nil {
		return nil, fmt.Errorf("failed to create sub-account: %w", err)
	}

	return s.convertToSubAccountResponse(subAccount), nil
}

// GetUserSubAccounts retrieves all sub-accounts for a user
func (s *SubAccountService) GetUserSubAccounts(ctx context.Context, userID uuid.UUID, exchangeID *uuid.UUID) ([]*SubAccountResponse, error) {
	// If exchangeID is provided, verify user owns it
	if exchangeID != nil {
		exchange, err := s.repos.Exchange.GetByID(ctx, *exchangeID)
		if err != nil {
			return nil, fmt.Errorf("failed to verify exchange: %w", err)
		}
		if exchange == nil || exchange.UserID != userID {
			return nil, fmt.Errorf("exchange not found")
		}
	}

	subAccounts, err := s.repos.SubAccount.GetByUserID(ctx, userID, exchangeID)
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

	// Update fields if provided
	if req.Name != nil {
		// Check if new name is unique for this user+exchange
		userSubAccounts, err := s.repos.SubAccount.GetByUserID(ctx, userID, &subAccount.ExchangeID)
		if err != nil {
			return nil, fmt.Errorf("failed to check sub-account names: %w", err)
		}

		for _, userSubAccount := range userSubAccounts {
			if userSubAccount.ID != subAccountID && userSubAccount.Name == *req.Name {
				return nil, fmt.Errorf("sub-account name already exists for this exchange")
			}
		}

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
		ExchangeID: subAccount.ExchangeID,
		Name:       subAccount.Name,
		Symbol:     subAccount.Symbol,
		Balance:    subAccount.Balance,
		Info:       info,
		CreatedAt:  subAccount.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:  subAccount.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
