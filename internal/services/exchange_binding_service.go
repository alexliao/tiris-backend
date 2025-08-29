package services

import (
	"context"
	"fmt"
	"strings"

	"tiris-backend/internal/models"
	"tiris-backend/internal/repositories"

	"github.com/google/uuid"
)

// ExchangeBindingService defines the interface for exchange binding operations
type ExchangeBindingService interface {
	CreateExchangeBinding(ctx context.Context, request *models.CreateExchangeBindingRequest) (*models.ExchangeBinding, error)
	GetExchangeBinding(ctx context.Context, id uuid.UUID) (*models.ExchangeBinding, error)
	GetUserExchangeBindings(ctx context.Context, userID uuid.UUID, params models.PaginationParams) ([]*models.ExchangeBinding, *models.PaginationResult, error)
	GetPublicExchangeBindings(ctx context.Context, exchange string) ([]*models.ExchangeBinding, error)
	UpdateExchangeBinding(ctx context.Context, id uuid.UUID, request *models.UpdateExchangeBindingRequest) (*models.ExchangeBinding, error)
	DeleteExchangeBinding(ctx context.Context, id uuid.UUID) error
	ValidateExchangeBindingAccess(ctx context.Context, userID uuid.UUID, bindingID uuid.UUID) (bool, error)
}

// exchangeBindingService implements ExchangeBindingService
type exchangeBindingService struct {
	repo repositories.ExchangeBindingRepository
}

// NewExchangeBindingService creates a new exchange binding service
func NewExchangeBindingService(repo repositories.ExchangeBindingRepository) ExchangeBindingService {
	return &exchangeBindingService{
		repo: repo,
	}
}

// CreateExchangeBinding creates a new exchange binding
func (s *exchangeBindingService) CreateExchangeBinding(ctx context.Context, request *models.CreateExchangeBindingRequest) (*models.ExchangeBinding, error) {
	// Validate request
	if err := request.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Check if name already exists for this user
	existing, err := s.repo.GetByNameAndUser(ctx, request.Name, request.UserID)
	if err != nil && err != models.ErrExchangeBindingNotFound {
		return nil, fmt.Errorf("failed to check existing binding: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("exchange binding with name '%s' already exists", request.Name)
	}

	// For private bindings, check for duplicate API credentials
	if request.Type == "private" && request.APIKey != "" {
		// Check for duplicate API key
		existingByAPIKey, err := s.repo.GetByAPIKey(ctx, request.APIKey, request.UserID)
		if err != nil && err != models.ErrExchangeBindingNotFound {
			return nil, fmt.Errorf("failed to check existing API key: %w", err)
		}
		if existingByAPIKey != nil {
			return nil, fmt.Errorf("API key already in use")
		}

		// Check for duplicate API secret
		existingByAPISecret, err := s.repo.GetByAPISecret(ctx, request.APISecret, request.UserID)
		if err != nil && err != models.ErrExchangeBindingNotFound {
			return nil, fmt.Errorf("failed to check existing API secret: %w", err)
		}
		if existingByAPISecret != nil {
			return nil, fmt.Errorf("API secret already in use")
		}
	}

	// Create the exchange binding
	binding := request.ToExchangeBinding()
	
	// Generate ID if not set
	if binding.ID == uuid.Nil {
		binding.ID = uuid.New()
	}

	if err := s.repo.Create(ctx, binding); err != nil {
		return nil, fmt.Errorf("failed to create exchange binding: %w", err)
	}

	return binding, nil
}

// GetExchangeBinding retrieves an exchange binding by ID
func (s *exchangeBindingService) GetExchangeBinding(ctx context.Context, id uuid.UUID) (*models.ExchangeBinding, error) {
	binding, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return binding, nil
}

// GetUserExchangeBindings retrieves exchange bindings for a user
func (s *exchangeBindingService) GetUserExchangeBindings(ctx context.Context, userID uuid.UUID, params models.PaginationParams) ([]*models.ExchangeBinding, *models.PaginationResult, error) {
	bindings, pagination, err := s.repo.GetByUserID(ctx, userID, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user exchange bindings: %w", err)
	}

	return bindings, pagination, nil
}

// GetPublicExchangeBindings retrieves public exchange bindings
func (s *exchangeBindingService) GetPublicExchangeBindings(ctx context.Context, exchange string) ([]*models.ExchangeBinding, error) {
	bindings, err := s.repo.GetPublicBindings(ctx, exchange)
	if err != nil {
		return nil, fmt.Errorf("failed to get public exchange bindings: %w", err)
	}

	return bindings, nil
}

// UpdateExchangeBinding updates an exchange binding
func (s *exchangeBindingService) UpdateExchangeBinding(ctx context.Context, id uuid.UUID, request *models.UpdateExchangeBindingRequest) (*models.ExchangeBinding, error) {
	// Validate request
	if err := request.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Get update map
	updates := request.ToUpdateMap()
	if len(updates) == 0 {
		return s.GetExchangeBinding(ctx, id)
	}

	// Perform update
	if err := s.repo.Update(ctx, id, updates); err != nil {
		return nil, fmt.Errorf("failed to update exchange binding: %w", err)
	}

	// Return updated binding
	return s.GetExchangeBinding(ctx, id)
}

// DeleteExchangeBinding deletes an exchange binding
func (s *exchangeBindingService) DeleteExchangeBinding(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		if strings.Contains(err.Error(), "foreign key constraint") {
			return fmt.Errorf("cannot delete exchange binding that is currently in use by one or more tradings")
		}
		return fmt.Errorf("failed to delete exchange binding: %w", err)
	}

	return nil
}

// ValidateExchangeBindingAccess validates if a user has access to an exchange binding
func (s *exchangeBindingService) ValidateExchangeBindingAccess(ctx context.Context, userID uuid.UUID, bindingID uuid.UUID) (bool, error) {
	binding, err := s.repo.GetByID(ctx, bindingID)
	if err != nil {
		return false, err
	}

	// Public bindings are accessible to everyone
	if binding.IsPublic() {
		return true, nil
	}

	// Private bindings are only accessible to their owner
	if binding.IsPrivate() && binding.UserID != nil && *binding.UserID == userID {
		return true, nil
	}

	return false, nil
}