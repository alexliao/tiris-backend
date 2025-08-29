package services

import (
	"context"
	"errors"
	"testing"

	"tiris-backend/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockExchangeBindingRepository for testing
type MockExchangeBindingRepository struct {
	mock.Mock
}

func (m *MockExchangeBindingRepository) Create(ctx context.Context, binding *models.ExchangeBinding) error {
	args := m.Called(ctx, binding)
	return args.Error(0)
}

func (m *MockExchangeBindingRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.ExchangeBinding, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ExchangeBinding), args.Error(1)
}

func (m *MockExchangeBindingRepository) GetByUserID(ctx context.Context, userID uuid.UUID, params models.PaginationParams) ([]*models.ExchangeBinding, *models.PaginationResult, error) {
	args := m.Called(ctx, userID, params)
	if args.Get(0) == nil {
		return nil, args.Get(1).(*models.PaginationResult), args.Error(2)
	}
	return args.Get(0).([]*models.ExchangeBinding), args.Get(1).(*models.PaginationResult), args.Error(2)
}

func (m *MockExchangeBindingRepository) GetPublicBindings(ctx context.Context, exchange string) ([]*models.ExchangeBinding, error) {
	args := m.Called(ctx, exchange)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ExchangeBinding), args.Error(1)
}

func (m *MockExchangeBindingRepository) Update(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	args := m.Called(ctx, id, updates)
	return args.Error(0)
}

func (m *MockExchangeBindingRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockExchangeBindingRepository) GetByNameAndUser(ctx context.Context, name string, userID *uuid.UUID) (*models.ExchangeBinding, error) {
	args := m.Called(ctx, name, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ExchangeBinding), args.Error(1)
}

func (m *MockExchangeBindingRepository) GetByAPIKey(ctx context.Context, apiKey string, userID *uuid.UUID) (*models.ExchangeBinding, error) {
	args := m.Called(ctx, apiKey, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ExchangeBinding), args.Error(1)
}

func (m *MockExchangeBindingRepository) GetByAPISecret(ctx context.Context, apiSecret string, userID *uuid.UUID) (*models.ExchangeBinding, error) {
	args := m.Called(ctx, apiSecret, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ExchangeBinding), args.Error(1)
}

// TestExchangeBindingService_CreateExchangeBinding tests the CreateExchangeBinding functionality
func TestExchangeBindingService_CreateExchangeBinding(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	t.Run("create_private_binding_success", func(t *testing.T) {
		mockRepo := &MockExchangeBindingRepository{}
		service := NewExchangeBindingService(mockRepo)

		request := &models.CreateExchangeBindingRequest{
			UserID:    &userID,
			Name:      "My Binance",
			Exchange:  "binance",
			Type:      "private",
			APIKey:    "test_api_key",
			APISecret: "test_api_secret",
		}

		// Mock repository calls
		mockRepo.On("GetByNameAndUser", ctx, "My Binance", &userID).Return(nil, models.ErrExchangeBindingNotFound)
		mockRepo.On("GetByAPIKey", ctx, "test_api_key", &userID).Return(nil, models.ErrExchangeBindingNotFound)
		mockRepo.On("GetByAPISecret", ctx, "test_api_secret", &userID).Return(nil, models.ErrExchangeBindingNotFound)
		mockRepo.On("Create", ctx, mock.AnythingOfType("*models.ExchangeBinding")).Return(nil)

		binding, err := service.CreateExchangeBinding(ctx, request)

		require.NoError(t, err)
		assert.NotNil(t, binding)
		assert.Equal(t, "My Binance", binding.Name)
		assert.Equal(t, "binance", binding.Exchange)
		assert.Equal(t, "private", binding.Type)

		mockRepo.AssertExpectations(t)
	})

	t.Run("create_public_binding_success", func(t *testing.T) {
		mockRepo := &MockExchangeBindingRepository{}
		service := NewExchangeBindingService(mockRepo)

		request := &models.CreateExchangeBindingRequest{
			UserID:    nil,
			Name:      "Public Binance",
			Exchange:  "binance",
			Type:      "public",
			APIKey:    "",
			APISecret: "",
		}

		// Mock repository calls
		mockRepo.On("GetByNameAndUser", ctx, "Public Binance", (*uuid.UUID)(nil)).Return(nil, models.ErrExchangeBindingNotFound)
		mockRepo.On("Create", ctx, mock.AnythingOfType("*models.ExchangeBinding")).Return(nil)

		binding, err := service.CreateExchangeBinding(ctx, request)

		require.NoError(t, err)
		assert.NotNil(t, binding)
		assert.Equal(t, "Public Binance", binding.Name)
		assert.Equal(t, "public", binding.Type)

		mockRepo.AssertExpectations(t)
	})

	t.Run("create_binding_name_exists_error", func(t *testing.T) {
		mockRepo := &MockExchangeBindingRepository{}
		service := NewExchangeBindingService(mockRepo)

		request := &models.CreateExchangeBindingRequest{
			UserID:    &userID,
			Name:      "Existing Binding",
			Exchange:  "binance",
			Type:      "private",
			APIKey:    "test_api_key",
			APISecret: "test_api_secret",
		}

		existingBinding := &models.ExchangeBinding{
			ID:   uuid.New(),
			Name: "Existing Binding",
		}

		mockRepo.On("GetByNameAndUser", ctx, "Existing Binding", &userID).Return(existingBinding, nil)

		binding, err := service.CreateExchangeBinding(ctx, request)

		require.Error(t, err)
		assert.Nil(t, binding)
		assert.Contains(t, err.Error(), "already exists")

		mockRepo.AssertExpectations(t)
	})

	t.Run("create_binding_invalid_request", func(t *testing.T) {
		mockRepo := &MockExchangeBindingRepository{}
		service := NewExchangeBindingService(mockRepo)

		request := &models.CreateExchangeBindingRequest{
			UserID:    &userID,
			Name:      "",
			Exchange:  "binance",
			Type:      "private",
			APIKey:    "test_api_key",
			APISecret: "test_api_secret",
		}

		binding, err := service.CreateExchangeBinding(ctx, request)

		require.Error(t, err)
		assert.Nil(t, binding)
		assert.Contains(t, err.Error(), "name is required")

		// No repository calls should be made for invalid requests
		mockRepo.AssertExpectations(t)
	})
}

// TestExchangeBindingService_GetExchangeBinding tests the GetExchangeBinding functionality
func TestExchangeBindingService_GetExchangeBinding(t *testing.T) {
	ctx := context.Background()
	bindingID := uuid.New()

	t.Run("get_binding_success", func(t *testing.T) {
		mockRepo := &MockExchangeBindingRepository{}
		service := NewExchangeBindingService(mockRepo)

		expectedBinding := &models.ExchangeBinding{
			ID:       bindingID,
			Name:     "Test Binding",
			Exchange: "binance",
			Type:     "private",
		}

		mockRepo.On("GetByID", ctx, bindingID).Return(expectedBinding, nil)

		binding, err := service.GetExchangeBinding(ctx, bindingID)

		require.NoError(t, err)
		assert.Equal(t, expectedBinding, binding)

		mockRepo.AssertExpectations(t)
	})

	t.Run("get_binding_not_found", func(t *testing.T) {
		mockRepo := &MockExchangeBindingRepository{}
		service := NewExchangeBindingService(mockRepo)

		mockRepo.On("GetByID", ctx, bindingID).Return(nil, models.ErrExchangeBindingNotFound)

		binding, err := service.GetExchangeBinding(ctx, bindingID)

		require.Error(t, err)
		assert.Nil(t, binding)
		assert.Contains(t, err.Error(), "exchange binding not found")

		mockRepo.AssertExpectations(t)
	})
}

// TestExchangeBindingService_GetUserExchangeBindings tests getting user's bindings
func TestExchangeBindingService_GetUserExchangeBindings(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	t.Run("get_user_bindings_success", func(t *testing.T) {
		mockRepo := &MockExchangeBindingRepository{}
		service := NewExchangeBindingService(mockRepo)

		expectedBindings := []*models.ExchangeBinding{
			{ID: uuid.New(), Name: "Binding 1"},
			{ID: uuid.New(), Name: "Binding 2"},
		}

		expectedPagination := &models.PaginationResult{
			Total:       2,
			Page:        1,
			Limit:       10,
			TotalPages:  1,
		}

		params := models.PaginationParams{Page: 1, Limit: 10}

		mockRepo.On("GetByUserID", ctx, userID, params).Return(expectedBindings, expectedPagination, nil)

		bindings, pagination, err := service.GetUserExchangeBindings(ctx, userID, params)

		require.NoError(t, err)
		assert.Equal(t, expectedBindings, bindings)
		assert.Equal(t, expectedPagination, pagination)

		mockRepo.AssertExpectations(t)
	})

	t.Run("get_user_bindings_empty", func(t *testing.T) {
		mockRepo := &MockExchangeBindingRepository{}
		service := NewExchangeBindingService(mockRepo)

		expectedPagination := &models.PaginationResult{
			Total:       0,
			Page:        1,
			Limit:       10,
			TotalPages:  0,
		}

		params := models.PaginationParams{Page: 1, Limit: 10}

		mockRepo.On("GetByUserID", ctx, userID, params).Return([]*models.ExchangeBinding{}, expectedPagination, nil)

		bindings, pagination, err := service.GetUserExchangeBindings(ctx, userID, params)

		require.NoError(t, err)
		assert.Empty(t, bindings)
		assert.Equal(t, expectedPagination, pagination)

		mockRepo.AssertExpectations(t)
	})
}

// TestExchangeBindingService_GetPublicExchangeBindings tests getting public bindings
func TestExchangeBindingService_GetPublicExchangeBindings(t *testing.T) {
	ctx := context.Background()

	t.Run("get_all_public_bindings", func(t *testing.T) {
		mockRepo := &MockExchangeBindingRepository{}
		service := NewExchangeBindingService(mockRepo)

		expectedBindings := []*models.ExchangeBinding{
			{ID: uuid.New(), Name: "Public Binance", Type: "public"},
			{ID: uuid.New(), Name: "Public Kraken", Type: "public"},
		}

		mockRepo.On("GetPublicBindings", ctx, "").Return(expectedBindings, nil)

		bindings, err := service.GetPublicExchangeBindings(ctx, "")

		require.NoError(t, err)
		assert.Equal(t, expectedBindings, bindings)

		mockRepo.AssertExpectations(t)
	})

	t.Run("get_public_bindings_by_exchange", func(t *testing.T) {
		mockRepo := &MockExchangeBindingRepository{}
		service := NewExchangeBindingService(mockRepo)

		expectedBindings := []*models.ExchangeBinding{
			{ID: uuid.New(), Name: "Public Binance", Exchange: "binance", Type: "public"},
		}

		mockRepo.On("GetPublicBindings", ctx, "binance").Return(expectedBindings, nil)

		bindings, err := service.GetPublicExchangeBindings(ctx, "binance")

		require.NoError(t, err)
		assert.Equal(t, expectedBindings, bindings)

		mockRepo.AssertExpectations(t)
	})
}

// TestExchangeBindingService_UpdateExchangeBinding tests updating bindings
func TestExchangeBindingService_UpdateExchangeBinding(t *testing.T) {
	ctx := context.Background()
	bindingID := uuid.New()

	t.Run("update_binding_success", func(t *testing.T) {
		mockRepo := &MockExchangeBindingRepository{}
		service := NewExchangeBindingService(mockRepo)

		newName := "Updated Name"
		request := &models.UpdateExchangeBindingRequest{
			Name: &newName,
		}

		expectedBinding := &models.ExchangeBinding{
			ID:   bindingID,
			Name: "Updated Name",
		}

		mockRepo.On("Update", ctx, bindingID, mock.AnythingOfType("map[string]interface {}")).Return(nil)
		mockRepo.On("GetByID", ctx, bindingID).Return(expectedBinding, nil)

		binding, err := service.UpdateExchangeBinding(ctx, bindingID, request)

		require.NoError(t, err)
		assert.Equal(t, expectedBinding, binding)

		mockRepo.AssertExpectations(t)
	})

	t.Run("update_binding_not_found", func(t *testing.T) {
		mockRepo := &MockExchangeBindingRepository{}
		service := NewExchangeBindingService(mockRepo)

		newName := "Updated Name"
		request := &models.UpdateExchangeBindingRequest{
			Name: &newName,
		}

		mockRepo.On("Update", ctx, bindingID, mock.AnythingOfType("map[string]interface {}")).Return(models.ErrExchangeBindingNotFound)

		binding, err := service.UpdateExchangeBinding(ctx, bindingID, request)

		require.Error(t, err)
		assert.Nil(t, binding)
		assert.Contains(t, err.Error(), "exchange binding not found")

		mockRepo.AssertExpectations(t)
	})
}

// TestExchangeBindingService_DeleteExchangeBinding tests deleting bindings
func TestExchangeBindingService_DeleteExchangeBinding(t *testing.T) {
	ctx := context.Background()
	bindingID := uuid.New()

	t.Run("delete_binding_success", func(t *testing.T) {
		mockRepo := &MockExchangeBindingRepository{}
		service := NewExchangeBindingService(mockRepo)

		mockRepo.On("Delete", ctx, bindingID).Return(nil)

		err := service.DeleteExchangeBinding(ctx, bindingID)

		require.NoError(t, err)

		mockRepo.AssertExpectations(t)
	})

	t.Run("delete_binding_not_found", func(t *testing.T) {
		mockRepo := &MockExchangeBindingRepository{}
		service := NewExchangeBindingService(mockRepo)

		mockRepo.On("Delete", ctx, bindingID).Return(models.ErrExchangeBindingNotFound)

		err := service.DeleteExchangeBinding(ctx, bindingID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "exchange binding not found")

		mockRepo.AssertExpectations(t)
	})

	t.Run("delete_binding_in_use", func(t *testing.T) {
		mockRepo := &MockExchangeBindingRepository{}
		service := NewExchangeBindingService(mockRepo)

		mockRepo.On("Delete", ctx, bindingID).Return(errors.New("foreign key constraint violation"))

		err := service.DeleteExchangeBinding(ctx, bindingID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete exchange binding that is currently in use")

		mockRepo.AssertExpectations(t)
	})
}

// TestExchangeBindingService_ValidateExchangeBindingAccess tests access validation
func TestExchangeBindingService_ValidateExchangeBindingAccess(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	otherUserID := uuid.New()
	bindingID := uuid.New()

	t.Run("access_own_private_binding", func(t *testing.T) {
		mockRepo := &MockExchangeBindingRepository{}
		service := NewExchangeBindingService(mockRepo)

		binding := &models.ExchangeBinding{
			ID:     bindingID,
			UserID: &userID,
			Type:   "private",
		}

		mockRepo.On("GetByID", ctx, bindingID).Return(binding, nil)

		hasAccess, err := service.ValidateExchangeBindingAccess(ctx, userID, bindingID)

		require.NoError(t, err)
		assert.True(t, hasAccess)

		mockRepo.AssertExpectations(t)
	})

	t.Run("access_public_binding", func(t *testing.T) {
		mockRepo := &MockExchangeBindingRepository{}
		service := NewExchangeBindingService(mockRepo)

		binding := &models.ExchangeBinding{
			ID:     bindingID,
			UserID: nil,
			Type:   "public",
		}

		mockRepo.On("GetByID", ctx, bindingID).Return(binding, nil)

		hasAccess, err := service.ValidateExchangeBindingAccess(ctx, userID, bindingID)

		require.NoError(t, err)
		assert.True(t, hasAccess)

		mockRepo.AssertExpectations(t)
	})

	t.Run("no_access_other_user_binding", func(t *testing.T) {
		mockRepo := &MockExchangeBindingRepository{}
		service := NewExchangeBindingService(mockRepo)

		binding := &models.ExchangeBinding{
			ID:     bindingID,
			UserID: &otherUserID,
			Type:   "private",
		}

		mockRepo.On("GetByID", ctx, bindingID).Return(binding, nil)

		hasAccess, err := service.ValidateExchangeBindingAccess(ctx, userID, bindingID)

		require.NoError(t, err)
		assert.False(t, hasAccess)

		mockRepo.AssertExpectations(t)
	})

	t.Run("binding_not_found", func(t *testing.T) {
		mockRepo := &MockExchangeBindingRepository{}
		service := NewExchangeBindingService(mockRepo)

		mockRepo.On("GetByID", ctx, bindingID).Return(nil, models.ErrExchangeBindingNotFound)

		hasAccess, err := service.ValidateExchangeBindingAccess(ctx, userID, bindingID)

		require.Error(t, err)
		assert.False(t, hasAccess)
		assert.Contains(t, err.Error(), "exchange binding not found")

		mockRepo.AssertExpectations(t)
	})
}