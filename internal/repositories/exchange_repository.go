package repositories

import (
	"context"
	"errors"

	"tiris-backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type exchangeRepository struct {
	db *gorm.DB
}

// NewExchangeRepository creates a new exchange repository instance
func NewExchangeRepository(db *gorm.DB) ExchangeRepository {
	return &exchangeRepository{db: db}
}

func (r *exchangeRepository) Create(ctx context.Context, exchange *models.Exchange) error {
	return r.db.WithContext(ctx).Create(exchange).Error
}

func (r *exchangeRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Exchange, error) {
	var exchange models.Exchange
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&exchange).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &exchange, nil
}

func (r *exchangeRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Exchange, error) {
	var exchanges []*models.Exchange
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&exchanges).Error
	if err != nil {
		return nil, err
	}
	return exchanges, nil
}

func (r *exchangeRepository) GetByUserIDAndType(ctx context.Context, userID uuid.UUID, exchangeType string) ([]*models.Exchange, error) {
	var exchanges []*models.Exchange
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND type = ?", userID, exchangeType).
		Order("created_at DESC").
		Find(&exchanges).Error
	if err != nil {
		return nil, err
	}
	return exchanges, nil
}

func (r *exchangeRepository) Update(ctx context.Context, exchange *models.Exchange) error {
	return r.db.WithContext(ctx).Save(exchange).Error
}

func (r *exchangeRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Check if there are any sub-accounts associated with this exchange
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.SubAccount{}).Where("exchange_id = ?", id).Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		return errors.New("cannot delete exchange with existing sub-accounts")
	}

	return r.db.WithContext(ctx).Delete(&models.Exchange{}, id).Error
}