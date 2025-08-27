package services

import (
	"context"
	"fmt"

	"tiris-backend/internal/models"
	"tiris-backend/internal/repositories"

	"github.com/google/uuid"
)

// ExchangeService handles exchange business logic
type ExchangeService struct {
	repos *repositories.Repositories
}

// NewExchangeService creates a new exchange service
func NewExchangeService(repos *repositories.Repositories) *ExchangeService {
	return &ExchangeService{
		repos: repos,
	}
}

// ExchangeResponse represents exchange information in responses
type ExchangeResponse struct {
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

// CreateExchangeRequest represents exchange creation request
type CreateExchangeRequest struct {
	Name      string `json:"name" binding:"required,min=1,max=100" example:"My Exchange Account"`
	Type      string `json:"type" binding:"required" example:"binance"`
	APIKey    string `json:"api_key" binding:"required,min=1" example:"your_api_key_here"`
	APISecret string `json:"api_secret" binding:"required,min=1" example:"your_api_secret_here"`
}

// UpdateExchangeRequest represents exchange update request
type UpdateExchangeRequest struct {
	Name      *string `json:"name,omitempty" binding:"omitempty,min=1,max=100" example:"My Updated Exchange Account"`
	APIKey    *string `json:"api_key,omitempty" binding:"omitempty,min=1" example:"updated_api_key_12345"`
	APISecret *string `json:"api_secret,omitempty" binding:"omitempty,min=1" example:"updated_api_secret_67890"`
	Status    *string `json:"status,omitempty" binding:"omitempty,oneof=active inactive" example:"active"`
}

// CreateExchange creates a new exchange configuration
func (s *ExchangeService) CreateExchange(ctx context.Context, userID uuid.UUID, req *CreateExchangeRequest) (*ExchangeResponse, error) {

	// Create info map with metadata
	infoMap := map[string]interface{}{
		"created_by":  "api",
		"api_version": "v1",
	}

	// Create exchange model
	exchange := &models.Exchange{
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
	if err := s.repos.Exchange.Create(ctx, exchange); err != nil {
		// Check for specific constraint violations and provide user-friendly messages
		if constraintMsg := getSpecificConstraintViolation(err); constraintMsg != "" {
			return nil, fmt.Errorf(constraintMsg)
		}
		return nil, fmt.Errorf("failed to create exchange: %w", err)
	}

	return s.convertToExchangeResponse(exchange), nil
}

// GetUserExchanges retrieves all exchanges for a user
func (s *ExchangeService) GetUserExchanges(ctx context.Context, userID uuid.UUID) ([]*ExchangeResponse, error) {
	exchanges, err := s.repos.Exchange.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user exchanges: %w", err)
	}

	var responses []*ExchangeResponse
	for _, exchange := range exchanges {
		responses = append(responses, s.convertToExchangeResponse(exchange))
	}

	return responses, nil
}

// GetExchange retrieves a specific exchange by ID (must belong to user)
func (s *ExchangeService) GetExchange(ctx context.Context, userID, exchangeID uuid.UUID) (*ExchangeResponse, error) {
	exchange, err := s.repos.Exchange.GetByID(ctx, exchangeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get exchange: %w", err)
	}
	if exchange == nil {
		return nil, fmt.Errorf("exchange not found")
	}

	// Check if exchange belongs to the user
	if exchange.UserID != userID {
		return nil, fmt.Errorf("exchange not found")
	}

	return s.convertToExchangeResponse(exchange), nil
}

// UpdateExchange updates an existing exchange
func (s *ExchangeService) UpdateExchange(ctx context.Context, userID, exchangeID uuid.UUID, req *UpdateExchangeRequest) (*ExchangeResponse, error) {
	// Get existing exchange
	exchange, err := s.repos.Exchange.GetByID(ctx, exchangeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get exchange: %w", err)
	}
	if exchange == nil {
		return nil, fmt.Errorf("exchange not found")
	}

	// Check if exchange belongs to the user
	if exchange.UserID != userID {
		return nil, fmt.Errorf("exchange not found")
	}

	// Update fields if provided - let database constraints handle uniqueness validation
	if req.Name != nil {
		exchange.Name = *req.Name
	}

	if req.APIKey != nil {
		exchange.APIKey = *req.APIKey
	}

	if req.APISecret != nil {
		exchange.APISecret = *req.APISecret
	}

	if req.Status != nil {
		exchange.Status = *req.Status
	}

	// Save updated exchange
	if err := s.repos.Exchange.Update(ctx, exchange); err != nil {
		// Check for specific constraint violations and provide user-friendly messages
		if constraintMsg := getSpecificConstraintViolation(err); constraintMsg != "" {
			return nil, fmt.Errorf(constraintMsg)
		}
		return nil, fmt.Errorf("failed to update exchange: %w", err)
	}

	return s.convertToExchangeResponse(exchange), nil
}

// DeleteExchange deletes an exchange (soft delete)
func (s *ExchangeService) DeleteExchange(ctx context.Context, userID, exchangeID uuid.UUID) error {
	// Get existing exchange to verify ownership
	exchange, err := s.repos.Exchange.GetByID(ctx, exchangeID)
	if err != nil {
		return fmt.Errorf("failed to get exchange: %w", err)
	}
	if exchange == nil {
		return fmt.Errorf("exchange not found")
	}

	// Check if exchange belongs to the user
	if exchange.UserID != userID {
		return fmt.Errorf("exchange not found")
	}

	// Check if exchange has sub-accounts
	subAccounts, err := s.repos.SubAccount.GetByExchangeID(ctx, exchangeID)
	if err != nil {
		return fmt.Errorf("failed to check sub-accounts: %w", err)
	}

	if len(subAccounts) > 0 {
		return fmt.Errorf("cannot delete exchange with existing sub-accounts")
	}

	// Soft delete the exchange
	if err := s.repos.Exchange.Delete(ctx, exchangeID); err != nil {
		return fmt.Errorf("failed to delete exchange: %w", err)
	}

	return nil
}

// ListExchanges lists all exchanges with pagination (admin only)
// For now, returns all exchanges without pagination since we don't have List method
func (s *ExchangeService) ListExchanges(ctx context.Context, limit, offset int) ([]*ExchangeResponse, int64, error) {
	// This would need a List method in the repository
	// For now, we'll return an error indicating this is not implemented
	return nil, 0, fmt.Errorf("list exchanges not implemented yet")
}

// GetExchangeByID retrieves exchange by ID (admin only)
func (s *ExchangeService) GetExchangeByID(ctx context.Context, exchangeID uuid.UUID) (*ExchangeResponse, error) {
	exchange, err := s.repos.Exchange.GetByID(ctx, exchangeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get exchange: %w", err)
	}
	if exchange == nil {
		return nil, fmt.Errorf("exchange not found")
	}

	return s.convertToExchangeResponse(exchange), nil
}

// convertToExchangeResponse converts an exchange model to response format
func (s *ExchangeService) convertToExchangeResponse(exchange *models.Exchange) *ExchangeResponse {
	var info map[string]interface{}
	if len(exchange.Info) > 0 {
		info = exchange.Info
	} else {
		info = make(map[string]interface{})
	}

	// Mask API key for security (show only first 4 and last 4 characters)
	maskedAPIKey := exchange.APIKey
	if len(maskedAPIKey) > 8 {
		maskedAPIKey = maskedAPIKey[:4] + "****" + maskedAPIKey[len(maskedAPIKey)-4:]
	}

	return &ExchangeResponse{
		ID:        exchange.ID,
		UserID:    exchange.UserID,
		Name:      exchange.Name,
		Type:      exchange.Type,
		APIKey:    maskedAPIKey,
		Status:    exchange.Status,
		Info:      info,
		CreatedAt: exchange.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: exchange.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
