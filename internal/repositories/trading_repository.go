package repositories

import (
	"context"
	"errors"

	"tiris-backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type tradingRepository struct {
	db *gorm.DB
}

// NewTradingRepository creates a new trading repository instance
func NewTradingRepository(db *gorm.DB) TradingRepository {
	return &tradingRepository{db: db}
}

func (r *tradingRepository) Create(ctx context.Context, trading *models.Trading) error {
	// Validate that the exchange binding exists and user has access to it
	if trading.ExchangeBindingID != uuid.Nil {
		var binding models.ExchangeBinding
		err := r.db.WithContext(ctx).
			Where("id = ?", trading.ExchangeBindingID).
			First(&binding).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return models.ErrExchangeBindingNotFound
			}
			return err
		}

		// Validate access to the exchange binding
		if binding.IsPrivate() && (binding.UserID == nil || *binding.UserID != trading.UserID) {
			return errors.New("access denied to exchange binding")
		}
	}

	return r.db.WithContext(ctx).Create(trading).Error
}

func (r *tradingRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Trading, error) {
	var trading models.Trading
	err := r.db.WithContext(ctx).
		Preload("ExchangeBinding").
		Where("id = ?", id).
		First(&trading).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &trading, nil
}

func (r *tradingRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Trading, error) {
	var tradings []*models.Trading
	err := r.db.WithContext(ctx).
		Preload("ExchangeBinding").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&tradings).Error
	if err != nil {
		return nil, err
	}
	return tradings, nil
}

func (r *tradingRepository) GetByUserIDAndType(ctx context.Context, userID uuid.UUID, tradingType string) ([]*models.Trading, error) {
	var tradings []*models.Trading
	err := r.db.WithContext(ctx).
		Preload("ExchangeBinding").
		Where("user_id = ? AND type = ?", userID, tradingType).
		Order("created_at DESC").
		Find(&tradings).Error
	if err != nil {
		return nil, err
	}
	return tradings, nil
}

func (r *tradingRepository) Update(ctx context.Context, trading *models.Trading) error {
	// Validate exchange binding if it's being updated
	if trading.ExchangeBindingID != uuid.Nil {
		var binding models.ExchangeBinding
		err := r.db.WithContext(ctx).
			Where("id = ?", trading.ExchangeBindingID).
			First(&binding).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return models.ErrExchangeBindingNotFound
			}
			return err
		}

		// Validate access to the exchange binding
		if binding.IsPrivate() && (binding.UserID == nil || *binding.UserID != trading.UserID) {
			return errors.New("access denied to exchange binding")
		}
	}

	return r.db.WithContext(ctx).Save(trading).Error
}

func (r *tradingRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Check if there are any sub-accounts associated with this trading
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.SubAccount{}).Where("trading_id = ?", id).Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		return errors.New("cannot delete trading with existing sub-accounts")
	}

	return r.db.WithContext(ctx).Delete(&models.Trading{}, id).Error
}

func (r *tradingRepository) GetByExchangeBinding(ctx context.Context, bindingID uuid.UUID) ([]*models.Trading, error) {
	var tradings []*models.Trading
	err := r.db.WithContext(ctx).
		Preload("ExchangeBinding").
		Where("exchange_binding_id = ?", bindingID).
		Order("created_at DESC").
		Find(&tradings).Error
	if err != nil {
		return nil, err
	}
	return tradings, nil
}