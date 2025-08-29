package repositories

import (
	"context"
	"errors"
	"fmt"

	"tiris-backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ExchangeBindingRepository defines the interface for exchange binding operations
type ExchangeBindingRepository interface {
	Create(ctx context.Context, binding *models.ExchangeBinding) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.ExchangeBinding, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, params models.PaginationParams) ([]*models.ExchangeBinding, *models.PaginationResult, error)
	GetPublicBindings(ctx context.Context, exchange string) ([]*models.ExchangeBinding, error)
	Update(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByNameAndUser(ctx context.Context, name string, userID *uuid.UUID) (*models.ExchangeBinding, error)
	GetByAPIKey(ctx context.Context, apiKey string, userID *uuid.UUID) (*models.ExchangeBinding, error)
	GetByAPISecret(ctx context.Context, apiSecret string, userID *uuid.UUID) (*models.ExchangeBinding, error)
}

// exchangeBindingRepository implements ExchangeBindingRepository
type exchangeBindingRepository struct {
	db *gorm.DB
}

// NewExchangeBindingRepository creates a new exchange binding repository
func NewExchangeBindingRepository(db *gorm.DB) ExchangeBindingRepository {
	return &exchangeBindingRepository{
		db: db,
	}
}

// Create creates a new exchange binding
func (r *exchangeBindingRepository) Create(ctx context.Context, binding *models.ExchangeBinding) error {
	if err := binding.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if err := r.db.WithContext(ctx).Create(binding).Error; err != nil {
		if isDuplicateKeyError(err) {
			return models.ErrExchangeBindingNameExists
		}
		return fmt.Errorf("failed to create exchange binding: %w", err)
	}

	return nil
}

// GetByID retrieves an exchange binding by ID
func (r *exchangeBindingRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.ExchangeBinding, error) {
	var binding models.ExchangeBinding
	
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&binding).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, models.ErrExchangeBindingNotFound
		}
		return nil, fmt.Errorf("failed to get exchange binding: %w", err)
	}

	return &binding, nil
}

// GetByUserID retrieves exchange bindings for a specific user with pagination
func (r *exchangeBindingRepository) GetByUserID(ctx context.Context, userID uuid.UUID, params models.PaginationParams) ([]*models.ExchangeBinding, *models.PaginationResult, error) {
	var bindings []*models.ExchangeBinding
	var total int64

	// Set default pagination values
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.Limit <= 0 {
		params.Limit = 10
	}
	if params.Limit > 100 {
		params.Limit = 100
	}

	// Count total records
	err := r.db.WithContext(ctx).
		Model(&models.ExchangeBinding{}).
		Where("user_id = ?", userID).
		Count(&total).Error

	if err != nil {
		return nil, nil, fmt.Errorf("failed to count exchange bindings: %w", err)
	}

	// Calculate offset
	offset := (params.Page - 1) * params.Limit

	// Get paginated results
	err = r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(params.Limit).
		Offset(offset).
		Find(&bindings).Error

	if err != nil {
		return nil, nil, fmt.Errorf("failed to get exchange bindings: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int((total + int64(params.Limit) - 1) / int64(params.Limit))
	
	paginationResult := &models.PaginationResult{
		Total:      total,
		Page:       params.Page,
		Limit:      params.Limit,
		TotalPages: totalPages,
	}

	return bindings, paginationResult, nil
}

// GetPublicBindings retrieves public exchange bindings
func (r *exchangeBindingRepository) GetPublicBindings(ctx context.Context, exchange string) ([]*models.ExchangeBinding, error) {
	var bindings []*models.ExchangeBinding
	
	query := r.db.WithContext(ctx).
		Where("type = ?", models.ExchangeBindingTypePublic).
		Where("status = ?", models.ExchangeBindingStatusActive).
		Order("name ASC")

	if exchange != "" {
		query = query.Where("exchange = ?", exchange)
	}

	err := query.Find(&bindings).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get public exchange bindings: %w", err)
	}

	return bindings, nil
}

// Update updates an exchange binding
func (r *exchangeBindingRepository) Update(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	result := r.db.WithContext(ctx).
		Model(&models.ExchangeBinding{}).
		Where("id = ?", id).
		Updates(updates)

	if result.Error != nil {
		if isDuplicateKeyError(result.Error) {
			return models.ErrExchangeBindingNameExists
		}
		return fmt.Errorf("failed to update exchange binding: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return models.ErrExchangeBindingNotFound
	}

	return nil
}

// Delete deletes an exchange binding
func (r *exchangeBindingRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&models.ExchangeBinding{})

	if result.Error != nil {
		if isForeignKeyError(result.Error) {
			return models.ErrExchangeBindingInUse
		}
		return fmt.Errorf("failed to delete exchange binding: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return models.ErrExchangeBindingNotFound
	}

	return nil
}

// GetByNameAndUser retrieves an exchange binding by name and user
func (r *exchangeBindingRepository) GetByNameAndUser(ctx context.Context, name string, userID *uuid.UUID) (*models.ExchangeBinding, error) {
	var binding models.ExchangeBinding
	
	query := r.db.WithContext(ctx).Where("name = ?", name)
	
	if userID == nil {
		// For public bindings
		query = query.Where("user_id IS NULL")
	} else {
		// For private bindings
		query = query.Where("user_id = ?", *userID)
	}

	err := query.First(&binding).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, models.ErrExchangeBindingNotFound
		}
		return nil, fmt.Errorf("failed to get exchange binding by name: %w", err)
	}

	return &binding, nil
}

// GetByAPIKey retrieves an exchange binding by API key and user
func (r *exchangeBindingRepository) GetByAPIKey(ctx context.Context, apiKey string, userID *uuid.UUID) (*models.ExchangeBinding, error) {
	var binding models.ExchangeBinding
	
	query := r.db.WithContext(ctx).Where("api_key = ? AND deleted_at IS NULL", apiKey)
	
	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}

	err := query.First(&binding).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, models.ErrExchangeBindingNotFound
		}
		return nil, fmt.Errorf("failed to get exchange binding by API key: %w", err)
	}

	return &binding, nil
}

// GetByAPISecret retrieves an exchange binding by API secret and user
func (r *exchangeBindingRepository) GetByAPISecret(ctx context.Context, apiSecret string, userID *uuid.UUID) (*models.ExchangeBinding, error) {
	var binding models.ExchangeBinding
	
	query := r.db.WithContext(ctx).Where("api_secret = ? AND deleted_at IS NULL", apiSecret)
	
	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}

	err := query.First(&binding).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, models.ErrExchangeBindingNotFound
		}
		return nil, fmt.Errorf("failed to get exchange binding by API secret: %w", err)
	}

	return &binding, nil
}

// Helper functions for error detection

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	// Check for PostgreSQL duplicate key error
	return containsString(err.Error(), "duplicate key value") ||
		   containsString(err.Error(), "violates unique constraint") ||
		   containsString(err.Error(), "UNIQUE constraint failed")
}

func isForeignKeyError(err error) bool {
	if err == nil {
		return false
	}
	// Check for PostgreSQL foreign key error
	return containsString(err.Error(), "foreign key constraint") ||
		   containsString(err.Error(), "violates foreign key constraint") ||
		   containsString(err.Error(), "FOREIGN KEY constraint failed")
}

func containsString(str, substr string) bool {
	return len(str) >= len(substr) && 
		   (str == substr || 
			(len(str) > len(substr) && 
			 (str[:len(substr)] == substr || 
			  str[len(str)-len(substr):] == substr ||
			  containsSubstring(str, substr))))
}

func containsSubstring(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}