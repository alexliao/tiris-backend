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
	repos *repositories.Repositories
}

// NewTradingService creates a new trading service
func NewTradingService(repos *repositories.Repositories) *TradingService {
	return &TradingService{
		repos: repos,
	}
}

// TradingResponse represents trading information in responses
type TradingResponse struct {
	ID        uuid.UUID              `json:"id"`
	UserID    uuid.UUID              `json:"user_id"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	APIKey    string                 `json:"api_key,omitempty"` // Masked in production
	Status    string                 `json:"status"`
	Info      map[string]interface{} `json:"info"`
	CreatedAt string                 `json:"created_at"`
	UpdatedAt string                 `json:"updated_at"`
}

// CreateTradingRequest represents trading creation request
type CreateTradingRequest struct {
	Name      string `json:"name" binding:"required,min=1,max=100" example:"My Trading Account"`
	Type      string `json:"type" binding:"required" example:"binance"`
	APIKey    string `json:"api_key" binding:"required,min=1" example:"your_api_key_here"`
	APISecret string `json:"api_secret" binding:"required,min=1" example:"your_api_secret_here"`
}

// UpdateTradingRequest represents trading update request
type UpdateTradingRequest struct {
	Name      *string `json:"name,omitempty" binding:"omitempty,min=1,max=100" example:"My Updated Trading Account"`
	APIKey    *string `json:"api_key,omitempty" binding:"omitempty,min=1" example:"updated_api_key_12345"`
	APISecret *string `json:"api_secret,omitempty" binding:"omitempty,min=1" example:"updated_api_secret_67890"`
	Status    *string `json:"status,omitempty" binding:"omitempty,oneof=active inactive" example:"active"`
}

// CreateTrading creates a new trading configuration
func (s *TradingService) CreateTrading(ctx context.Context, userID uuid.UUID, req *CreateTradingRequest) (*TradingResponse, error) {

	// Create info map with metadata
	infoMap := map[string]interface{}{
		"created_by":  "api",
		"api_version": "v1",
	}

	// Create trading model
	trading := &models.Trading{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      req.Name,
		Type:      req.Type,
		APIKey:    req.APIKey,
		APISecret: req.APISecret,
		Status:    "active", // Default to active
		Info:      models.JSON(infoMap),
	}

	// Save to database - let database constraints handle uniqueness validation
	if err := s.repos.Trading.Create(ctx, trading); err != nil {
		// Check for specific constraint violations and provide user-friendly messages
		if constraintMsg := getSpecificConstraintViolation(err); constraintMsg != "" {
			return nil, fmt.Errorf(constraintMsg)
		}
		return nil, fmt.Errorf("failed to create trading: %w", err)
	}

	return s.convertToTradingResponse(trading), nil
}

// GetUserTradings retrieves all tradings for a user
func (s *TradingService) GetUserTradings(ctx context.Context, userID uuid.UUID) ([]*TradingResponse, error) {
	tradings, err := s.repos.Trading.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user tradings: %w", err)
	}

	var responses []*TradingResponse
	for _, trading := range tradings {
		responses = append(responses, s.convertToTradingResponse(trading))
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

	return s.convertToTradingResponse(trading), nil
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

	// Update fields if provided - let database constraints handle uniqueness validation
	if req.Name != nil {
		trading.Name = *req.Name
	}

	if req.APIKey != nil {
		trading.APIKey = *req.APIKey
	}

	if req.APISecret != nil {
		trading.APISecret = *req.APISecret
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

	return s.convertToTradingResponse(trading), nil
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

	return s.convertToTradingResponse(trading), nil
}

// convertToTradingResponse converts a trading model to response format
func (s *TradingService) convertToTradingResponse(trading *models.Trading) *TradingResponse {
	var info map[string]interface{}
	if len(trading.Info) > 0 {
		info = trading.Info
	} else {
		info = make(map[string]interface{})
	}

	// Mask API key for security (show only first 4 and last 4 characters)
	maskedAPIKey := trading.APIKey
	if len(maskedAPIKey) > 8 {
		maskedAPIKey = maskedAPIKey[:4] + "****" + maskedAPIKey[len(maskedAPIKey)-4:]
	}

	return &TradingResponse{
		ID:        trading.ID,
		UserID:    trading.UserID,
		Name:      trading.Name,
		Type:      trading.Type,
		APIKey:    maskedAPIKey,
		Status:    trading.Status,
		Info:      info,
		CreatedAt: trading.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: trading.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}