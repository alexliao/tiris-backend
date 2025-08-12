package repositories

import (
	"context"
	"errors"
	"time"

	"tiris-backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type eventProcessingRepository struct {
	db *gorm.DB
}

// NewEventProcessingRepository creates a new event processing repository instance
func NewEventProcessingRepository(db *gorm.DB) EventProcessingRepository {
	return &eventProcessingRepository{db: db}
}

func (r *eventProcessingRepository) Create(ctx context.Context, event *models.EventProcessing) error {
	return r.db.WithContext(ctx).Create(event).Error
}

func (r *eventProcessingRepository) GetByEventID(ctx context.Context, eventID string) (*models.EventProcessing, error) {
	var event models.EventProcessing
	err := r.db.WithContext(ctx).Where("event_id = ?", eventID).First(&event).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &event, nil
}

func (r *eventProcessingRepository) GetByEventType(ctx context.Context, eventType string, filters EventProcessingFilters) ([]*models.EventProcessing, int64, error) {
	var events []*models.EventProcessing
	var total int64

	// Build base query
	query := r.db.WithContext(ctx).Model(&models.EventProcessing{}).Where("event_type = ?", eventType)
	
	// Apply filters
	if filters.Status != nil {
		query = query.Where("status = ?", *filters.Status)
	}
	if filters.StartDate != nil {
		query = query.Where("processed_at >= ?", *filters.StartDate)
	}
	if filters.EndDate != nil {
		query = query.Where("processed_at <= ?", *filters.EndDate)
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

	err := query.Order("processed_at DESC").Find(&events).Error
	if err != nil {
		return nil, 0, err
	}

	return events, total, nil
}

func (r *eventProcessingRepository) Update(ctx context.Context, event *models.EventProcessing) error {
	return r.db.WithContext(ctx).Save(event).Error
}

func (r *eventProcessingRepository) MarkAsProcessed(ctx context.Context, eventID string) error {
	return r.db.WithContext(ctx).
		Model(&models.EventProcessing{}).
		Where("event_id = ?", eventID).
		Updates(map[string]interface{}{
			"status":       "processed",
			"processed_at": time.Now(),
		}).Error
}

func (r *eventProcessingRepository) MarkAsFailed(ctx context.Context, eventID string, errorMessage string, retryCount int) error {
	return r.db.WithContext(ctx).
		Model(&models.EventProcessing{}).
		Where("event_id = ?", eventID).
		Updates(map[string]interface{}{
			"status":        "failed",
			"error_message": errorMessage,
			"retry_count":   retryCount,
			"processed_at":  time.Now(),
		}).Error
}

func (r *eventProcessingRepository) GetFailedEvents(ctx context.Context, maxRetries int) ([]*models.EventProcessing, error) {
	var events []*models.EventProcessing
	err := r.db.WithContext(ctx).
		Where("status = ? AND retry_count < ?", "failed", maxRetries).
		Order("processed_at ASC").
		Find(&events).Error
	if err != nil {
		return nil, err
	}
	return events, nil
}

func (r *eventProcessingRepository) DeleteOldEvents(ctx context.Context, olderThan time.Time) error {
	return r.db.WithContext(ctx).
		Where("processed_at < ? AND status = ?", olderThan, "processed").
		Delete(&models.EventProcessing{}).Error
}