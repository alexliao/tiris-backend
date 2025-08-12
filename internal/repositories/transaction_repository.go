package repositories

import (
	"context"
	"errors"
	"time"

	"tiris-backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type transactionRepository struct {
	db *gorm.DB
}

// NewTransactionRepository creates a new transaction repository instance
func NewTransactionRepository(db *gorm.DB) TransactionRepository {
	return &transactionRepository{db: db}
}

func (r *transactionRepository) Create(ctx context.Context, transaction *models.Transaction) error {
	return r.db.WithContext(ctx).Create(transaction).Error
}

func (r *transactionRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Transaction, error) {
	var transaction models.Transaction
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&transaction).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &transaction, nil
}

func (r *transactionRepository) GetByUserID(ctx context.Context, userID uuid.UUID, filters TransactionFilters) ([]*models.Transaction, int64, error) {
	return r.getTransactions(ctx, filters, "user_id = ?", userID)
}

func (r *transactionRepository) GetBySubAccountID(ctx context.Context, subAccountID uuid.UUID, filters TransactionFilters) ([]*models.Transaction, int64, error) {
	return r.getTransactions(ctx, filters, "sub_account_id = ?", subAccountID)
}

func (r *transactionRepository) GetByExchangeID(ctx context.Context, exchangeID uuid.UUID, filters TransactionFilters) ([]*models.Transaction, int64, error) {
	return r.getTransactions(ctx, filters, "exchange_id = ?", exchangeID)
}

func (r *transactionRepository) GetByTimeRange(ctx context.Context, startTime, endTime time.Time, filters TransactionFilters) ([]*models.Transaction, int64, error) {
	return r.getTransactions(ctx, filters, "timestamp BETWEEN ? AND ?", startTime, endTime)
}

func (r *transactionRepository) getTransactions(ctx context.Context, filters TransactionFilters, whereClause string, whereArgs ...interface{}) ([]*models.Transaction, int64, error) {
	var transactions []*models.Transaction
	var total int64

	// Build base query
	query := r.db.WithContext(ctx).Model(&models.Transaction{}).Where(whereClause, whereArgs...)
	
	// Apply filters
	if filters.Direction != nil {
		query = query.Where("direction = ?", *filters.Direction)
	}
	if filters.Reason != nil {
		query = query.Where("reason = ?", *filters.Reason)
	}
	if filters.StartDate != nil {
		query = query.Where("timestamp >= ?", *filters.StartDate)
	}
	if filters.EndDate != nil {
		query = query.Where("timestamp <= ?", *filters.EndDate)
	}
	if filters.MinAmount != nil {
		query = query.Where("amount >= ?", *filters.MinAmount)
	}
	if filters.MaxAmount != nil {
		query = query.Where("amount <= ?", *filters.MaxAmount)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination and ordering
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	err := query.Order("timestamp DESC").Find(&transactions).Error
	if err != nil {
		return nil, 0, err
	}

	return transactions, total, nil
}