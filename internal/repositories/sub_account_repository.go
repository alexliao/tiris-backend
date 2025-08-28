package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"tiris-backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type subAccountRepository struct {
	db *gorm.DB
}

// NewSubAccountRepository creates a new sub-account repository instance
func NewSubAccountRepository(db *gorm.DB) SubAccountRepository {
	return &subAccountRepository{db: db}
}

func (r *subAccountRepository) Create(ctx context.Context, subAccount *models.SubAccount) error {
	return r.db.WithContext(ctx).Create(subAccount).Error
}

func (r *subAccountRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.SubAccount, error) {
	var subAccount models.SubAccount
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&subAccount).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &subAccount, nil
}

func (r *subAccountRepository) GetByUserID(ctx context.Context, userID uuid.UUID, tradingID *uuid.UUID) ([]*models.SubAccount, error) {
	var subAccounts []*models.SubAccount
	query := r.db.WithContext(ctx).Where("user_id = ?", userID)

	if tradingID != nil {
		query = query.Where("trading_id = ?", *tradingID)
	}

	err := query.Order("created_at DESC").Find(&subAccounts).Error
	if err != nil {
		return nil, err
	}
	return subAccounts, nil
}

func (r *subAccountRepository) GetByTradingID(ctx context.Context, tradingID uuid.UUID) ([]*models.SubAccount, error) {
	var subAccounts []*models.SubAccount
	err := r.db.WithContext(ctx).
		Where("trading_id = ?", tradingID).
		Order("created_at DESC").
		Find(&subAccounts).Error
	if err != nil {
		return nil, err
	}
	return subAccounts, nil
}

func (r *subAccountRepository) GetBySymbol(ctx context.Context, userID uuid.UUID, symbol string) ([]*models.SubAccount, error) {
	var subAccounts []*models.SubAccount
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND symbol = ?", userID, symbol).
		Order("created_at DESC").
		Find(&subAccounts).Error
	if err != nil {
		return nil, err
	}
	return subAccounts, nil
}

func (r *subAccountRepository) Update(ctx context.Context, subAccount *models.SubAccount) error {
	return r.db.WithContext(ctx).Save(subAccount).Error
}

func (r *subAccountRepository) UpdateBalance(ctx context.Context, subAccountID uuid.UUID, newBalance float64, amount float64, direction, reason string, info interface{}) (*uuid.UUID, error) {
	// Convert info to JSON string if it's not already
	var infoJSON string
	if info != nil {
		if str, ok := info.(string); ok {
			infoJSON = str
		} else {
			infoBytes, err := json.Marshal(info)
			if err != nil {
				return nil, err
			}
			infoJSON = string(infoBytes)
		}
	} else {
		infoJSON = "{}"
	}

	// Call the database function
	var transactionIDStr string
	err := r.db.WithContext(ctx).Raw(
		"SELECT update_sub_account_balance(?, ?, ?, ?, ?, ?::jsonb)",
		subAccountID, newBalance, amount, direction, reason, infoJSON,
	).Row().Scan(&transactionIDStr)

	if err != nil {
		return nil, err
	}

	// Parse the string result to UUID
	transactionID, err := uuid.Parse(transactionIDStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse transaction ID: %w", err)
	}

	return &transactionID, nil
}

func (r *subAccountRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// First check if balance is zero
	var subAccount models.SubAccount
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&subAccount).Error; err != nil {
		return err
	}

	if subAccount.Balance != 0 {
		return errors.New("cannot delete sub-account with non-zero balance")
	}

	return r.db.WithContext(ctx).Delete(&models.SubAccount{}, id).Error
}
