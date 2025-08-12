package repositories

import (
	"context"
	"errors"

	"tiris-backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type oauthTokenRepository struct {
	db *gorm.DB
}

// NewOAuthTokenRepository creates a new OAuth token repository instance
func NewOAuthTokenRepository(db *gorm.DB) OAuthTokenRepository {
	return &oauthTokenRepository{db: db}
}

func (r *oauthTokenRepository) Create(ctx context.Context, token *models.OAuthToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *oauthTokenRepository) GetByUserIDAndProvider(ctx context.Context, userID uuid.UUID, provider string) (*models.OAuthToken, error) {
	var token models.OAuthToken
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND provider = ?", userID, provider).
		First(&token).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &token, nil
}

func (r *oauthTokenRepository) GetByProviderUserID(ctx context.Context, provider, providerUserID string) (*models.OAuthToken, error) {
	var token models.OAuthToken
	err := r.db.WithContext(ctx).
		Where("provider = ? AND provider_user_id = ?", provider, providerUserID).
		First(&token).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &token, nil
}

func (r *oauthTokenRepository) Update(ctx context.Context, token *models.OAuthToken) error {
	return r.db.WithContext(ctx).Save(token).Error
}

func (r *oauthTokenRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.OAuthToken{}, id).Error
}

func (r *oauthTokenRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&models.OAuthToken{}).Error
}