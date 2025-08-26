package test

import (
	"context"
	"testing"

	"tiris-backend/internal/models"
	"tiris-backend/internal/repositories"
	"tiris-backend/internal/services"
	"tiris-backend/test/config"
	"tiris-backend/test/helpers"
	"tiris-backend/test/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestUserService_GetCurrentUser tests the GetCurrentUser functionality
func TestUserService_GetCurrentUser(t *testing.T) {
	// Setup test config
	_ = config.GetProfileConfig(config.ProfileQuick)

	// Create mocks
	mockUserRepo := &mocks.MockUserRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            mockUserRepo,
		Exchange:        &mocks.MockExchangeRepository{},
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create service
	userService := services.NewUserService(repos)

	// Create test user
	userFactory := helpers.NewUserFactory()
	testUser := userFactory.Build()
	testUser.ID = uuid.New()

	// Test successful retrieval
	t.Run("successful_retrieval", func(t *testing.T) {
		// Setup mock expectations
		mockUserRepo.On("GetByID", mock.Anything, testUser.ID).
			Return(testUser, nil).Once()

		// Execute test
		result, err := userService.GetCurrentUser(context.Background(), testUser.ID)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, testUser.ID, result.ID)
		assert.Equal(t, testUser.Username, result.Username)
		assert.Equal(t, testUser.Email, result.Email)

		// Verify mock expectations
		mockUserRepo.AssertExpectations(t)
	})

	// Test user not found
	t.Run("user_not_found", func(t *testing.T) {
		userID := uuid.New()

		// Setup mock expectations
		mockUserRepo.On("GetByID", mock.Anything, userID).
			Return(nil, nil).Once()

		// Execute test
		result, err := userService.GetCurrentUser(context.Background(), userID)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "user not found")

		// Verify mock expectations
		mockUserRepo.AssertExpectations(t)
	})
}

// TestUserService_UpdateCurrentUser tests the UpdateCurrentUser functionality
func TestUserService_UpdateCurrentUser(t *testing.T) {
	// Setup test config
	_ = config.GetProfileConfig(config.ProfileQuick)

	// Create mocks
	mockUserRepo := &mocks.MockUserRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            mockUserRepo,
		Exchange:        &mocks.MockExchangeRepository{},
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create service
	userService := services.NewUserService(repos)

	// Create test user
	userFactory := helpers.NewUserFactory()
	testUser := userFactory.Build()
	testUser.ID = uuid.New()

	// Test successful username update
	t.Run("successful_username_update", func(t *testing.T) {
		newUsername := "newusername"
		request := &services.UpdateUserRequest{
			Username: &newUsername,
		}

		// Setup mock expectations
		mockUserRepo.On("GetByID", mock.Anything, testUser.ID).
			Return(testUser, nil).Once()
		mockUserRepo.On("GetByUsername", mock.Anything, newUsername).
			Return(nil, nil).Once() // Username available
		mockUserRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.User")).
			Return(nil).Once()

		// Execute test
		result, err := userService.UpdateCurrentUser(context.Background(), testUser.ID, request)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, newUsername, result.Username)

		// Verify mock expectations
		mockUserRepo.AssertExpectations(t)
	})

	// Test username already taken
	t.Run("username_already_taken", func(t *testing.T) {
		existingUsername := "existinguser"
		request := &services.UpdateUserRequest{
			Username: &existingUsername,
		}

		// Create existing user with different ID
		existingUser := userFactory.Build()
		existingUser.ID = uuid.New()
		existingUser.Username = existingUsername

		// Setup mock expectations
		mockUserRepo.On("GetByID", mock.Anything, testUser.ID).
			Return(testUser, nil).Once()
		mockUserRepo.On("GetByUsername", mock.Anything, existingUsername).
			Return(existingUser, nil).Once() // Username taken

		// Execute test
		result, err := userService.UpdateCurrentUser(context.Background(), testUser.ID, request)

		// Verify results
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "username already taken")

		// Verify mock expectations
		mockUserRepo.AssertExpectations(t)
	})
}

// TestUserService_ListUsers tests the ListUsers functionality
func TestUserService_ListUsers(t *testing.T) {
	// Create mocks
	mockUserRepo := &mocks.MockUserRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            mockUserRepo,
		Exchange:        &mocks.MockExchangeRepository{},
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create service
	userService := services.NewUserService(repos)

	// Create test users
	userFactory := helpers.NewUserFactory()
	testUsers := []*models.User{
		userFactory.Build(),
		userFactory.Build(),
		userFactory.Build(),
	}
	testUsers[1].Username = "user2"
	testUsers[2].Username = "user3"

	// Test successful list
	t.Run("successful_list", func(t *testing.T) {
		limit, offset := 10, 0
		totalCount := int64(3)

		// Setup mock expectations
		mockUserRepo.On("List", mock.Anything, limit, offset).
			Return(testUsers, totalCount, nil).Once()

		// Execute test
		users, total, err := userService.ListUsers(context.Background(), limit, offset)

		// Verify results
		require.NoError(t, err)
		assert.Len(t, users, 3)
		assert.Equal(t, totalCount, total)

		// Verify user data
		for i, user := range users {
			assert.Equal(t, testUsers[i].ID, user.ID)
			assert.Equal(t, testUsers[i].Username, user.Username)
			assert.Equal(t, testUsers[i].Email, user.Email)
		}

		// Verify mock expectations
		mockUserRepo.AssertExpectations(t)
	})
}

// TestUserService_GetUserStats tests the GetUserStats functionality
func TestUserService_GetUserStats(t *testing.T) {
	// Create mocks
	mockUserRepo := &mocks.MockUserRepository{}
	mockExchRepo := &mocks.MockExchangeRepository{}
	mockSubRepo := &mocks.MockSubAccountRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            mockUserRepo,
		Exchange:        mockExchRepo,
		SubAccount:      mockSubRepo,
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create service
	userService := services.NewUserService(repos)

	// Create test data
	userID := uuid.New()
	testExchanges := []*models.Exchange{
		{ID: uuid.New(), UserID: userID},
		{ID: uuid.New(), UserID: userID},
	}
	testSubAccounts := []*models.SubAccount{
		{ID: uuid.New(), UserID: userID, Balance: 1000.0},
		{ID: uuid.New(), UserID: userID, Balance: 2000.0},
		{ID: uuid.New(), UserID: userID, Balance: 3000.0},
	}

	// Test successful stats calculation
	t.Run("successful_stats_calculation", func(t *testing.T) {
		// Setup mock expectations
		mockExchRepo.On("GetByUserID", mock.Anything, userID).
			Return(testExchanges, nil).Once()
		mockSubRepo.On("GetByUserID", mock.Anything, userID, mock.Anything).
			Return(testSubAccounts, nil).Once()

		// Execute test
		stats, err := userService.GetUserStats(context.Background(), userID)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, stats)

		assert.Equal(t, 2, stats["total_exchanges"])
		assert.Equal(t, 3, stats["total_subaccounts"])
		assert.Equal(t, 6000.0, stats["total_balance"]) // 1000 + 2000 + 3000
		assert.Equal(t, 2, stats["active_exchanges"])

		// Verify mock expectations
		mockExchRepo.AssertExpectations(t)
		mockSubRepo.AssertExpectations(t)
	})
}

// TestUserService_DisableUser tests the DisableUser functionality
func TestUserService_DisableUser(t *testing.T) {
	// Create mocks
	mockUserRepo := &mocks.MockUserRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            mockUserRepo,
		Exchange:        &mocks.MockExchangeRepository{},
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create service
	userService := services.NewUserService(repos)

	// Create test user
	userFactory := helpers.NewUserFactory()
	testUser := userFactory.Build()
	testUser.ID = uuid.New()

	// Test successful disable
	t.Run("successful_disable", func(t *testing.T) {
		// Setup mock expectations
		mockUserRepo.On("GetByID", mock.Anything, testUser.ID).
			Return(testUser, nil).Once()
		mockUserRepo.On("Delete", mock.Anything, testUser.ID).
			Return(nil).Once()

		// Execute test
		err := userService.DisableUser(context.Background(), testUser.ID)

		// Verify results
		require.NoError(t, err)

		// Verify mock expectations
		mockUserRepo.AssertExpectations(t)
	})

	// Test user not found
	t.Run("user_not_found", func(t *testing.T) {
		userID := uuid.New()

		// Setup mock expectations
		mockUserRepo.On("GetByID", mock.Anything, userID).
			Return(nil, nil).Once()

		// Execute test
		err := userService.DisableUser(context.Background(), userID)

		// Verify results
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")

		// Verify mock expectations
		mockUserRepo.AssertExpectations(t)
	})
}

// Performance test example
func TestUserService_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Create mocks
	mockUserRepo := &mocks.MockUserRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            mockUserRepo,
		Exchange:        &mocks.MockExchangeRepository{},
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create service
	userService := services.NewUserService(repos)

	// Create test user
	userFactory := helpers.NewUserFactory()
	testUser := userFactory.Build()
	testUser.ID = uuid.New()

	t.Run("get_current_user_performance", func(t *testing.T) {
		// Setup mock for multiple calls
		mockUserRepo.On("GetByID", mock.Anything, testUser.ID).
			Return(testUser, nil).Times(100)

		timer := helpers.NewPerformanceTimer()
		timer.Start()

		// Run operation multiple times
		for i := 0; i < 100; i++ {
			_, err := userService.GetCurrentUser(context.Background(), testUser.ID)
			require.NoError(t, err)
		}

		timer.Stop()

		// Check performance within reasonable bounds
		assert.Less(t, timer.Duration().Milliseconds(), int64(1000),
			"100 GetCurrentUser operations should complete within 1 second")

		mockUserRepo.AssertExpectations(t)
	})
}
