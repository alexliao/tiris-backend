package services

import (
	"context"
	"fmt"

	"tiris-backend/internal/models"
	"tiris-backend/internal/repositories"

	"github.com/google/uuid"
)

// UserService handles user business logic
type UserService struct {
	repos *repositories.Repositories
}

// NewUserService creates a new user service
func NewUserService(repos *repositories.Repositories) *UserService {
	return &UserService{
		repos: repos,
	}
}

// UserResponse represents user information in responses
type UserResponse struct {
	ID        uuid.UUID              `json:"id"`
	Username  string                 `json:"username"`
	Email     string                 `json:"email"`
	Avatar    *string                `json:"avatar,omitempty"`
	Settings  map[string]interface{} `json:"settings"`
	Info      map[string]interface{} `json:"info"`
	CreatedAt string                 `json:"created_at"`
	UpdatedAt string                 `json:"updated_at"`
}

// UpdateUserRequest represents user update request
type UpdateUserRequest struct {
	Username *string                `json:"username,omitempty" binding:"omitempty,min=3,max=50" example:"johndoe_trader"`
	Avatar   *string                `json:"avatar,omitempty" binding:"omitempty,url" example:"https://example.com/avatars/johndoe.jpg"`
	Settings map[string]interface{} `json:"settings,omitempty"`
}

// GetCurrentUser retrieves current user profile
func (s *UserService) GetCurrentUser(ctx context.Context, userID uuid.UUID) (*UserResponse, error) {
	user, err := s.repos.User.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	return s.convertToUserResponse(user), nil
}

// UpdateCurrentUser updates current user profile
func (s *UserService) UpdateCurrentUser(ctx context.Context, userID uuid.UUID, req *UpdateUserRequest) (*UserResponse, error) {
	// Get existing user
	user, err := s.repos.User.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Update fields if provided
	if req.Username != nil {
		// Check if username is already taken
		existingUser, err := s.repos.User.GetByUsername(ctx, *req.Username)
		if err != nil {
			return nil, fmt.Errorf("failed to check username availability: %w", err)
		}
		if existingUser != nil && existingUser.ID != userID {
			return nil, fmt.Errorf("username already taken")
		}
		user.Username = *req.Username
	}

	if req.Avatar != nil {
		user.Avatar = req.Avatar
	}

	if req.Settings != nil {
		// Merge with existing settings
		var existingSettings map[string]interface{}
		if len(user.Settings) > 0 {
			existingSettings = user.Settings
		} else {
			existingSettings = make(map[string]interface{})
		}

		// Merge new settings
		for key, value := range req.Settings {
			existingSettings[key] = value
		}

		user.Settings = models.JSON(existingSettings)
	}

	// Save updated user
	if err := s.repos.User.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return s.convertToUserResponse(user), nil
}

// DisableUser disables a user account (admin only)
func (s *UserService) DisableUser(ctx context.Context, userID uuid.UUID) error {
	user, err := s.repos.User.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}

	// Use soft delete to disable the user
	if err := s.repos.User.Delete(ctx, userID); err != nil {
		return fmt.Errorf("failed to disable user: %w", err)
	}

	return nil
}

// ListUsers lists all users (admin only) with pagination
func (s *UserService) ListUsers(ctx context.Context, limit, offset int) ([]*UserResponse, int64, error) {
	users, total, err := s.repos.User.List(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}

	var responses []*UserResponse
	for _, user := range users {
		responses = append(responses, s.convertToUserResponse(user))
	}

	return responses, total, nil
}

// GetUserByID retrieves user by ID (admin only)
func (s *UserService) GetUserByID(ctx context.Context, userID uuid.UUID) (*UserResponse, error) {
	user, err := s.repos.User.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	return s.convertToUserResponse(user), nil
}

// GetUserStats retrieves user statistics
func (s *UserService) GetUserStats(ctx context.Context, userID uuid.UUID) (map[string]interface{}, error) {
	// Get user's exchanges count
	exchanges, err := s.repos.Exchange.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user exchanges: %w", err)
	}

	// Get user's sub-accounts count
	subAccounts, err := s.repos.SubAccount.GetByUserID(ctx, userID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get user sub-accounts: %w", err)
	}

	// Calculate total balance across all sub-accounts
	var totalBalance float64
	for _, subAccount := range subAccounts {
		totalBalance += subAccount.Balance
	}

	// Get recent transaction count (last 30 days)
	// For simplicity, we'll skip complex date filtering here
	// In a real implementation, you'd add a method to get transaction count by date range

	stats := map[string]interface{}{
		"total_exchanges":    len(exchanges),
		"total_subaccounts":  len(subAccounts),
		"total_balance":      totalBalance,
		"active_exchanges":   len(exchanges), // Assuming all are active for now
	}

	return stats, nil
}

// convertToUserResponse converts a user model to response format
func (s *UserService) convertToUserResponse(user *models.User) *UserResponse {
	var settings map[string]interface{}
	if len(user.Settings) > 0 {
		settings = user.Settings
	} else {
		settings = make(map[string]interface{})
	}

	var info map[string]interface{}
	if len(user.Info) > 0 {
		info = user.Info
	} else {
		info = make(map[string]interface{})
	}

	return &UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Avatar:    user.Avatar,
		Settings:  settings,
		Info:      info,
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
