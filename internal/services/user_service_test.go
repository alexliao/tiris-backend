package services

import (
	"context"
	"testing"

	"tiris-backend/internal/models"
	"tiris-backend/internal/repositories"
	"tiris-backend/test/fixtures"
	"tiris-backend/test/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type UserServiceTestSuite struct {
	suite.Suite
	mockRepos   *repositories.Repositories
	userService *UserService
	ctx         context.Context
}

func (suite *UserServiceTestSuite) SetupTest() {
	suite.ctx = context.Background()
	
	// Create mock repositories
	suite.mockRepos = &repositories.Repositories{
		User:             &mocks.MockUserRepository{},
		Exchange:         &mocks.MockExchangeRepository{},
		SubAccount:       &mocks.MockSubAccountRepository{},
		Transaction:      &mocks.MockTransactionRepository{},
		TradingLog:       &mocks.MockTradingLogRepository{},
		OAuthToken:       &mocks.MockOAuthTokenRepository{},
		EventProcessing:  &mocks.MockEventProcessingRepository{},
	}
	
	// Create service with mocked dependencies
	suite.userService = NewUserService(suite.mockRepos)
}

func (suite *UserServiceTestSuite) TestGetCurrentUser_Success() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	expectedUser := fixtures.UserFixtures.ValidUser
	
	mockUserRepo := suite.mockRepos.User.(*mocks.MockUserRepository)
	mockUserRepo.On("GetByID", suite.ctx, userID).Return(expectedUser, nil)
	
	// Act
	result, err := suite.userService.GetCurrentUser(suite.ctx, userID)
	
	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), expectedUser.ID, result.ID)
	assert.Equal(suite.T(), expectedUser.Email, result.Email)
	assert.Equal(suite.T(), expectedUser.Username, result.Username)
	
	mockUserRepo.AssertExpectations(suite.T())
}

func (suite *UserServiceTestSuite) TestGetCurrentUser_NotFound() {
	// Arrange
	userID := uuid.New()
	
	mockUserRepo := suite.mockRepos.User.(*mocks.MockUserRepository)
	mockUserRepo.On("GetByID", suite.ctx, userID).Return((*models.User)(nil), repositories.ErrUserNotFound)
	
	// Act
	result, err := suite.userService.GetCurrentUser(suite.ctx, userID)
	
	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "failed to get user")
	
	mockUserRepo.AssertExpectations(suite.T())
}

func (suite *UserServiceTestSuite) TestUpdateCurrentUser_Success() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	newUsername := "newusername"
	newAvatar := "https://example.com/new-avatar.png"
	req := &UpdateUserRequest{
		Username: &newUsername,
		Avatar:   &newAvatar,
		Settings: map[string]interface{}{
			"theme": "light",
			"notifications": true,
		},
	}
	
	existingUser := &models.User{
		ID:       userID,
		Username: fixtures.UserFixtures.ValidUser.Username,
		Email:    fixtures.UserFixtures.ValidUser.Email,
		Avatar:   fixtures.UserFixtures.ValidUser.Avatar,
		Settings: fixtures.UserFixtures.ValidUser.Settings,
		Info:     fixtures.UserFixtures.ValidUser.Info,
		CreatedAt: fixtures.UserFixtures.ValidUser.CreatedAt,
		UpdatedAt: fixtures.UserFixtures.ValidUser.UpdatedAt,
	}
	
	mockUserRepo := suite.mockRepos.User.(*mocks.MockUserRepository)
	mockUserRepo.On("GetByID", suite.ctx, userID).Return(existingUser, nil)
	mockUserRepo.On("GetByUsername", suite.ctx, newUsername).Return((*models.User)(nil), nil)
	mockUserRepo.On("Update", suite.ctx, mock.MatchedBy(func(user *models.User) bool {
		return user.ID == userID && 
			   user.Username == newUsername &&
			   user.Avatar != nil && *user.Avatar == newAvatar
	})).Return(nil)
	
	// Act
	result, err := suite.userService.UpdateCurrentUser(suite.ctx, userID, req)
	
	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), newUsername, result.Username)
	assert.NotNil(suite.T(), result.Avatar)
	assert.Equal(suite.T(), newAvatar, *result.Avatar)
	
	mockUserRepo.AssertExpectations(suite.T())
}

func (suite *UserServiceTestSuite) TestUpdateCurrentUser_UserNotFound() {
	// Arrange
	userID := uuid.New()
	req := &UpdateUserRequest{
		Username: stringPtr("newusername"),
	}
	
	mockUserRepo := suite.mockRepos.User.(*mocks.MockUserRepository)
	mockUserRepo.On("GetByID", suite.ctx, userID).Return((*models.User)(nil), repositories.ErrUserNotFound)
	
	// Act
	result, err := suite.userService.UpdateCurrentUser(suite.ctx, userID, req)
	
	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "failed to get user")
	
	mockUserRepo.AssertExpectations(suite.T())
}

func (suite *UserServiceTestSuite) TestUpdateCurrentUser_UsernameAlreadyTaken() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	existingUsername := "existinguser"
	req := &UpdateUserRequest{
		Username: &existingUsername,
	}
	
	existingUser := fixtures.UserFixtures.ValidUser
	userWithSameUsername := &models.User{
		ID:       uuid.New(), // Different user
		Username: existingUsername,
		Email:    "different@example.com",
	}
	
	mockUserRepo := suite.mockRepos.User.(*mocks.MockUserRepository)
	mockUserRepo.On("GetByID", suite.ctx, userID).Return(existingUser, nil)
	mockUserRepo.On("GetByUsername", suite.ctx, existingUsername).Return(userWithSameUsername, nil)
	
	// Act
	result, err := suite.userService.UpdateCurrentUser(suite.ctx, userID, req)
	
	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "username already taken")
	
	mockUserRepo.AssertExpectations(suite.T())
}

func (suite *UserServiceTestSuite) TestListUsers_Success() {
	// Arrange
	limit, offset := 10, 0
	expectedUsers := []*models.User{
		fixtures.UserFixtures.ValidUser,
		fixtures.UserFixtures.AdminUser,
	}
	expectedTotal := int64(2)
	
	mockUserRepo := suite.mockRepos.User.(*mocks.MockUserRepository)
	mockUserRepo.On("List", suite.ctx, limit, offset).Return(expectedUsers, expectedTotal, nil)
	
	// Act
	result, total, err := suite.userService.ListUsers(suite.ctx, limit, offset)
	
	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), expectedTotal, total)
	assert.Len(suite.T(), result, 2)
	assert.Equal(suite.T(), expectedUsers[0].Username, result[0].Username)
	assert.Equal(suite.T(), expectedUsers[1].Username, result[1].Username)
	
	mockUserRepo.AssertExpectations(suite.T())
}

func (suite *UserServiceTestSuite) TestDisableUser_Success() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	
	mockUserRepo := suite.mockRepos.User.(*mocks.MockUserRepository)
	mockUserRepo.On("GetByID", suite.ctx, userID).Return(fixtures.UserFixtures.ValidUser, nil)
	mockUserRepo.On("Delete", suite.ctx, userID).Return(nil)
	
	// Act
	err := suite.userService.DisableUser(suite.ctx, userID)
	
	// Assert
	assert.NoError(suite.T(), err)
	
	mockUserRepo.AssertExpectations(suite.T())
}

func (suite *UserServiceTestSuite) TestDisableUser_UserNotFound() {
	// Arrange
	userID := uuid.New()
	
	mockUserRepo := suite.mockRepos.User.(*mocks.MockUserRepository)
	mockUserRepo.On("GetByID", suite.ctx, userID).Return((*models.User)(nil), repositories.ErrUserNotFound)
	
	// Act
	err := suite.userService.DisableUser(suite.ctx, userID)
	
	// Assert
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "failed to get user")
	
	mockUserRepo.AssertExpectations(suite.T())
}

func (suite *UserServiceTestSuite) TestGetUserByID_Success() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	expectedUser := fixtures.UserFixtures.ValidUser
	
	mockUserRepo := suite.mockRepos.User.(*mocks.MockUserRepository)
	mockUserRepo.On("GetByID", suite.ctx, userID).Return(expectedUser, nil)
	
	// Act
	result, err := suite.userService.GetUserByID(suite.ctx, userID)
	
	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), expectedUser.ID, result.ID)
	assert.Equal(suite.T(), expectedUser.Email, result.Email)
	
	mockUserRepo.AssertExpectations(suite.T())
}

func (suite *UserServiceTestSuite) TestGetUserStats_Success() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	expectedExchanges := []*models.Exchange{
		{ID: uuid.New(), UserID: userID, Name: "binance"},
		{ID: uuid.New(), UserID: userID, Name: "okx"},
	}
	expectedSubAccounts := []*models.SubAccount{
		{ID: uuid.New(), UserID: userID, Balance: 1000.0},
		{ID: uuid.New(), UserID: userID, Balance: 2000.0},
		{ID: uuid.New(), UserID: userID, Balance: 500.0},
	}
	
	mockExchangeRepo := suite.mockRepos.Exchange.(*mocks.MockExchangeRepository)
	mockSubAccountRepo := suite.mockRepos.SubAccount.(*mocks.MockSubAccountRepository)
	
	mockExchangeRepo.On("GetByUserID", suite.ctx, userID).Return(expectedExchanges, nil)
	mockSubAccountRepo.On("GetByUserID", suite.ctx, userID, (*uuid.UUID)(nil)).Return(expectedSubAccounts, nil)
	
	// Act
	result, err := suite.userService.GetUserStats(suite.ctx, userID)
	
	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), len(expectedExchanges), result["exchanges_count"])
	assert.Equal(suite.T(), len(expectedSubAccounts), result["sub_accounts_count"])
	assert.Equal(suite.T(), 3500.0, result["total_balance"]) // 1000 + 2000 + 500
	
	mockExchangeRepo.AssertExpectations(suite.T())
	mockSubAccountRepo.AssertExpectations(suite.T())
}

func TestUserServiceTestSuite(t *testing.T) {
	suite.Run(t, new(UserServiceTestSuite))
}

// Helper function
func stringPtr(s string) *string {
	return &s
}