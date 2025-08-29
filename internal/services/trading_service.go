package services

import (
	"context"
	"fmt"

	"tiris-backend/internal/models"
	"tiris-backend/internal/repositories"

	"github.com/google/uuid"
)


// TradingService handles trading business logic
type TradingService struct {
	repos                  *repositories.Repositories
	exchangeBindingService ExchangeBindingService
}

// NewTradingService creates a new trading service
func NewTradingService(repos *repositories.Repositories, exchangeBindingService ExchangeBindingService) *TradingService {
	return &TradingService{
		repos:                  repos,
		exchangeBindingService: exchangeBindingService,
	}
}

// TradingResponse represents trading information in responses
type TradingResponse struct {
	ID              uuid.UUID               `json:"id"`
	UserID          uuid.UUID               `json:"user_id"`
	Name            string                  `json:"name"`
	Type            string                  `json:"type"`
	ExchangeBinding *ExchangeBindingInfo    `json:"exchange_binding,omitempty"`
	Status          string                  `json:"status"`
	Info            map[string]interface{}  `json:"info"`
	CreatedAt       string                  `json:"created_at"`
	UpdatedAt       string                  `json:"updated_at"`
}

// ExchangeBindingInfo represents exchange binding information in responses
type ExchangeBindingInfo struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Exchange     string    `json:"exchange"`
	Type         string    `json:"type"`
	MaskedAPIKey string    `json:"masked_api_key,omitempty"`
}

// CreateTradingRequest represents trading creation request
type CreateTradingRequest struct {
	Name              string    `json:"name" binding:"required,min=1,max=100" example:"My Trading Account"`
	Type              string    `json:"type" binding:"required" example:"real"`
	ExchangeBindingID uuid.UUID `json:"exchange_binding_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// UpdateTradingRequest represents trading update request
type UpdateTradingRequest struct {
	Name              *string    `json:"name,omitempty" binding:"omitempty,min=1,max=100" example:"My Updated Trading Account"`
	ExchangeBindingID *uuid.UUID `json:"exchange_binding_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	Status            *string    `json:"status,omitempty" binding:"omitempty,oneof=active inactive" example:"active"`
}

// CreateTrading creates a new trading configuration
func (s *TradingService) CreateTrading(ctx context.Context, userID uuid.UUID, req *CreateTradingRequest) (*TradingResponse, error) {
	// Validate that the user has access to the exchange binding
	hasAccess, err := s.exchangeBindingService.ValidateExchangeBindingAccess(ctx, userID, req.ExchangeBindingID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate exchange binding access: %w", err)
	}
	if !hasAccess {
		return nil, fmt.Errorf("access denied to exchange binding")
	}

	// Create info map with metadata
	infoMap := map[string]interface{}{
		"created_by":  "api",
		"api_version": "v1",
	}

	// Create trading model
	trading := &models.Trading{
		ID:                uuid.New(),
		UserID:            userID,
		Name:              req.Name,
		Type:              req.Type,
		ExchangeBindingID: req.ExchangeBindingID,
		Status:            "active", // Default to active
		Info:              models.JSON(infoMap),
	}

	// Save to database - let database constraints handle uniqueness validation
	if err := s.repos.Trading.Create(ctx, trading); err != nil {
		// Check for specific constraint violations and provide user-friendly messages
		if constraintMsg := getSpecificConstraintViolation(err); constraintMsg != "" {
			return nil, fmt.Errorf(constraintMsg)
		}
		return nil, fmt.Errorf("failed to create trading: %w", err)
	}

	return s.convertToTradingResponse(ctx, trading)
}

// GetUserTradings retrieves all tradings for a user
func (s *TradingService) GetUserTradings(ctx context.Context, userID uuid.UUID) ([]*TradingResponse, error) {
	tradings, err := s.repos.Trading.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user tradings: %w", err)
	}

	var responses []*TradingResponse
	for _, trading := range tradings {
		resp, err := s.convertToTradingResponse(ctx, trading)
		if err != nil {
			return nil, fmt.Errorf("failed to convert trading response: %w", err)
		}
		responses = append(responses, resp)
	}

	return responses, nil
}

// GetTrading retrieves a specific trading by ID (must belong to user)
func (s *TradingService) GetTrading(ctx context.Context, userID, tradingID uuid.UUID) (*TradingResponse, error) {
	trading, err := s.repos.Trading.GetByID(ctx, tradingID)
	if err != nil {
		return nil, fmt.Errorf("failed to get trading: %w", err)
	}
	if trading == nil {
		return nil, fmt.Errorf("trading not found")
	}

	// Check if trading belongs to the user
	if trading.UserID != userID {
		return nil, fmt.Errorf("trading not found")
	}

	return s.convertToTradingResponse(ctx, trading)
}

// UpdateTrading updates an existing trading
func (s *TradingService) UpdateTrading(ctx context.Context, userID, tradingID uuid.UUID, req *UpdateTradingRequest) (*TradingResponse, error) {
	// Get existing trading
	trading, err := s.repos.Trading.GetByID(ctx, tradingID)
	if err != nil {
		return nil, fmt.Errorf("failed to get trading: %w", err)
	}
	if trading == nil {
		return nil, fmt.Errorf("trading not found")
	}

	// Check if trading belongs to the user
	if trading.UserID != userID {
		return nil, fmt.Errorf("trading not found")
	}

	// Update fields if provided
	if req.Name != nil {
		trading.Name = *req.Name
	}

	if req.ExchangeBindingID != nil {
		// Validate that the user has access to the new exchange binding
		hasAccess, err := s.exchangeBindingService.ValidateExchangeBindingAccess(ctx, userID, *req.ExchangeBindingID)
		if err != nil {
			return nil, fmt.Errorf("failed to validate exchange binding access: %w", err)
		}
		if !hasAccess {
			return nil, fmt.Errorf("access denied to exchange binding")
		}
		trading.ExchangeBindingID = *req.ExchangeBindingID
	}

	if req.Status != nil {
		trading.Status = *req.Status
	}

	// Save updated trading
	if err := s.repos.Trading.Update(ctx, trading); err != nil {
		// Check for specific constraint violations and provide user-friendly messages
		if constraintMsg := getSpecificConstraintViolation(err); constraintMsg != "" {
			return nil, fmt.Errorf(constraintMsg)
		}
		return nil, fmt.Errorf("failed to update trading: %w", err)
	}

	return s.convertToTradingResponse(ctx, trading)
}

// DeleteTrading deletes a trading (soft delete)
func (s *TradingService) DeleteTrading(ctx context.Context, userID, tradingID uuid.UUID) error {
	// Get existing trading to verify ownership
	trading, err := s.repos.Trading.GetByID(ctx, tradingID)
	if err != nil {
		return fmt.Errorf("failed to get trading: %w", err)
	}
	if trading == nil {
		return fmt.Errorf("trading not found")
	}

	// Check if trading belongs to the user
	if trading.UserID != userID {
		return fmt.Errorf("trading not found")
	}

	// Check if trading has sub-accounts
	subAccounts, err := s.repos.SubAccount.GetByTradingID(ctx, tradingID)
	if err != nil {
		return fmt.Errorf("failed to check sub-accounts: %w", err)
	}

	if len(subAccounts) > 0 {
		return fmt.Errorf("cannot delete trading with existing sub-accounts")
	}

	// Soft delete the trading
	if err := s.repos.Trading.Delete(ctx, tradingID); err != nil {
		return fmt.Errorf("failed to delete trading: %w", err)
	}

	return nil
}

// ListTradings lists all tradings with pagination (admin only)
// For now, returns all tradings without pagination since we don't have List method
func (s *TradingService) ListTradings(ctx context.Context, limit, offset int) ([]*TradingResponse, int64, error) {
	// This would need a List method in the repository
	// For now, we'll return an error indicating this is not implemented
	return nil, 0, fmt.Errorf("list tradings not implemented yet")
}

// GetTradingByID retrieves trading by ID (admin only)
func (s *TradingService) GetTradingByID(ctx context.Context, tradingID uuid.UUID) (*TradingResponse, error) {
	trading, err := s.repos.Trading.GetByID(ctx, tradingID)
	if err != nil {
		return nil, fmt.Errorf("failed to get trading: %w", err)
	}
	if trading == nil {
		return nil, fmt.Errorf("trading not found")
	}

	return s.convertToTradingResponse(ctx, trading)
}

// convertToTradingResponse converts a trading model to response format
func (s *TradingService) convertToTradingResponse(ctx context.Context, trading *models.Trading) (*TradingResponse, error) {
	var info map[string]interface{}
	if len(trading.Info) > 0 {
		info = trading.Info
	} else {
		info = make(map[string]interface{})
	}

	resp := &TradingResponse{
		ID:        trading.ID,
		UserID:    trading.UserID,
		Name:      trading.Name,
		Type:      trading.Type,
		Status:    trading.Status,
		Info:      info,
		CreatedAt: trading.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: trading.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	// Include exchange binding information if available (check if ID is not zero)
	if trading.ExchangeBinding.ID != uuid.Nil {
		resp.ExchangeBinding = &ExchangeBindingInfo{
			ID:       trading.ExchangeBinding.ID,
			Name:     trading.ExchangeBinding.Name,
			Exchange: trading.ExchangeBinding.Exchange,
			Type:     trading.ExchangeBinding.Type,
		}

		// Include masked API key for private bindings that have credentials
		if trading.ExchangeBinding.IsPrivate() && trading.ExchangeBinding.HasCredentials() {
			resp.ExchangeBinding.MaskedAPIKey = trading.ExchangeBinding.GetMaskedAPIKey()
		}
	}

	return resp, nil
}