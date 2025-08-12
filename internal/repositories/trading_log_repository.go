package repositories

import (
	"context"
	"errors"
	"time"

	"tiris-backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type tradingLogRepository struct {
	db *gorm.DB
}

// NewTradingLogRepository creates a new trading log repository instance
func NewTradingLogRepository(db *gorm.DB) TradingLogRepository {
	return &tradingLogRepository{db: db}
}

func (r *tradingLogRepository) Create(ctx context.Context, log *models.TradingLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *tradingLogRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.TradingLog, error) {
	var log models.TradingLog
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&log).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &log, nil
}

func (r *tradingLogRepository) GetByUserID(ctx context.Context, userID uuid.UUID, filters TradingLogFilters) ([]*models.TradingLog, int64, error) {
	return r.getTradingLogs(ctx, filters, "user_id = ?", userID)
}

func (r *tradingLogRepository) GetBySubAccountID(ctx context.Context, subAccountID uuid.UUID, filters TradingLogFilters) ([]*models.TradingLog, int64, error) {
	return r.getTradingLogs(ctx, filters, "sub_account_id = ?", subAccountID)
}

func (r *tradingLogRepository) GetByExchangeID(ctx context.Context, exchangeID uuid.UUID, filters TradingLogFilters) ([]*models.TradingLog, int64, error) {
	return r.getTradingLogs(ctx, filters, "exchange_id = ?", exchangeID)
}

func (r *tradingLogRepository) GetByTimeRange(ctx context.Context, startTime, endTime time.Time, filters TradingLogFilters) ([]*models.TradingLog, int64, error) {
	return r.getTradingLogs(ctx, filters, "timestamp BETWEEN ? AND ?", startTime, endTime)
}

func (r *tradingLogRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.TradingLog{}, id).Error
}

func (r *tradingLogRepository) getTradingLogs(ctx context.Context, filters TradingLogFilters, whereClause string, whereArgs ...interface{}) ([]*models.TradingLog, int64, error) {
	var logs []*models.TradingLog
	var total int64

	// Build base query
	query := r.db.WithContext(ctx).Model(&models.TradingLog{}).Where(whereClause, whereArgs...)
	
	// Apply filters
	if filters.Type != nil {
		query = query.Where("type = ?", *filters.Type)
	}
	if filters.Source != nil {
		query = query.Where("source = ?", *filters.Source)
	}
	if filters.StartDate != nil {
		query = query.Where("timestamp >= ?", *filters.StartDate)
	}
	if filters.EndDate != nil {
		query = query.Where("timestamp <= ?", *filters.EndDate)
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

	err := query.Order("timestamp DESC").Find(&logs).Error
	if err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}