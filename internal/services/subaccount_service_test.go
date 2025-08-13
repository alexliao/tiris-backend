package services

import (
	"context"
	"encoding/json"
	"testing"

	"tiris-backend/internal/models"
	"tiris-backend/internal/repositories"
	"tiris-backend/test/fixtures"
	"tiris-backend/test/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gorm.io/datatypes"
)

type SubAccountServiceTestSuite struct {
	suite.Suite
	mockRepos          *repositories.Repositories
	subAccountService  *SubAccountService
	ctx                context.Context
}

func (suite *SubAccountServiceTestSuite) SetupTest() {
	suite.ctx = context.Background()

	// Create mock repositories
	suite.mockRepos = &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Exchange:        &mocks.MockExchangeRepository{},
		SubAccount:      &mocks.MockSubAccountRepository{},
		Transaction:     &mocks.MockTransactionRepository{},
		TradingLog:      &mocks.MockTradingLogRepository{},
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create service with mocked dependencies
	suite.subAccountService = NewSubAccountService(suite.mockRepos)
}

func (suite *SubAccountServiceTestSuite) TestCreateSubAccount_Success() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	exchangeID := uuid.New()
	req := &CreateSubAccountRequest{
		ExchangeID: exchangeID,
		Name:       "Test SubAccount",
		Symbol:     "BTC",
	}

	exchange := &models.Exchange{
		ID:     exchangeID,
		UserID: userID,
		Type:   "binance",
	}

	mockExchangeRepo := suite.mockRepos.Exchange.(*mocks.MockExchangeRepository)
	mockSubAccountRepo := suite.mockRepos.SubAccount.(*mocks.MockSubAccountRepository)

	mockExchangeRepo.On("GetByID", suite.ctx, exchangeID).Return(exchange, nil)
	mockSubAccountRepo.On("GetByUserID", suite.ctx, userID, &exchangeID).Return([]*models.SubAccount{}, nil)
	mockSubAccountRepo.On("Create", suite.ctx, mock.AnythingOfType("*models.SubAccount")).Return(nil)

	// Act
	result, err := suite.subAccountService.CreateSubAccount(suite.ctx, userID, req)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), userID, result.UserID)
	assert.Equal(suite.T(), exchangeID, result.ExchangeID)
	assert.Equal(suite.T(), req.Name, result.Name)
	assert.Equal(suite.T(), req.Symbol, result.Symbol)
	assert.Equal(suite.T(), 0.0, result.Balance)
	mockExchangeRepo.AssertExpectations(suite.T())
	mockSubAccountRepo.AssertExpectations(suite.T())
}

func (suite *SubAccountServiceTestSuite) TestCreateSubAccount_ExchangeNotFound() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	exchangeID := uuid.New()
	req := &CreateSubAccountRequest{
		ExchangeID: exchangeID,
		Name:       "Test SubAccount",
		Symbol:     "BTC",
	}

	mockExchangeRepo := suite.mockRepos.Exchange.(*mocks.MockExchangeRepository)
	mockExchangeRepo.On("GetByID", suite.ctx, exchangeID).Return(nil, nil)

	// Act
	result, err := suite.subAccountService.CreateSubAccount(suite.ctx, userID, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "exchange not found")
	mockExchangeRepo.AssertExpectations(suite.T())
}

func (suite *SubAccountServiceTestSuite) TestCreateSubAccount_WrongUser() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	exchangeID := uuid.New()
	otherUserID := uuid.New()
	req := &CreateSubAccountRequest{
		ExchangeID: exchangeID,
		Name:       "Test SubAccount",
		Symbol:     "BTC",
	}

	exchange := &models.Exchange{
		ID:     exchangeID,
		UserID: otherUserID, // Different user
		Type:   "binance",
	}

	mockExchangeRepo := suite.mockRepos.Exchange.(*mocks.MockExchangeRepository)
	mockExchangeRepo.On("GetByID", suite.ctx, exchangeID).Return(exchange, nil)

	// Act
	result, err := suite.subAccountService.CreateSubAccount(suite.ctx, userID, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "exchange not found")
	mockExchangeRepo.AssertExpectations(suite.T())
}

func (suite *SubAccountServiceTestSuite) TestCreateSubAccount_DuplicateName() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	exchangeID := uuid.New()
	req := &CreateSubAccountRequest{
		ExchangeID: exchangeID,
		Name:       "Existing SubAccount",
		Symbol:     "BTC",
	}

	exchange := &models.Exchange{
		ID:     exchangeID,
		UserID: userID,
		Type:   "binance",
	}

	existingSubAccount := &models.SubAccount{
		ID:         uuid.New(),
		UserID:     userID,
		ExchangeID: exchangeID,
		Name:       "Existing SubAccount",
	}

	mockExchangeRepo := suite.mockRepos.Exchange.(*mocks.MockExchangeRepository)
	mockSubAccountRepo := suite.mockRepos.SubAccount.(*mocks.MockSubAccountRepository)

	mockExchangeRepo.On("GetByID", suite.ctx, exchangeID).Return(exchange, nil)
	mockSubAccountRepo.On("GetByUserID", suite.ctx, userID, &exchangeID).Return([]*models.SubAccount{existingSubAccount}, nil)

	// Act
	result, err := suite.subAccountService.CreateSubAccount(suite.ctx, userID, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "sub-account name already exists")
	mockExchangeRepo.AssertExpectations(suite.T())
	mockSubAccountRepo.AssertExpectations(suite.T())
}

func (suite *SubAccountServiceTestSuite) TestGetUserSubAccounts_Success() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	exchangeID := uuid.New()
	infoJSON, _ := json.Marshal(map[string]interface{}{"test": "data"})

	subAccounts := []*models.SubAccount{
		{
			ID:         uuid.New(),
			UserID:     userID,
			ExchangeID: exchangeID,
			Name:       "SubAccount 1",
			Symbol:     "BTC",
			Balance:    100.5,
			Info:       datatypes.JSON(infoJSON),
			CreatedAt:  fixtures.FixedTime,
			UpdatedAt:  fixtures.FixedTime,
		},
		{
			ID:         uuid.New(),
			UserID:     userID,
			ExchangeID: exchangeID,
			Name:       "SubAccount 2",
			Symbol:     "ETH",
			Balance:    50.25,
			Info:       datatypes.JSON(infoJSON),
			CreatedAt:  fixtures.FixedTime,
			UpdatedAt:  fixtures.FixedTime,
		},
	}

	exchange := &models.Exchange{
		ID:     exchangeID,
		UserID: userID,
	}

	mockExchangeRepo := suite.mockRepos.Exchange.(*mocks.MockExchangeRepository)
	mockSubAccountRepo := suite.mockRepos.SubAccount.(*mocks.MockSubAccountRepository)

	mockExchangeRepo.On("GetByID", suite.ctx, exchangeID).Return(exchange, nil)
	mockSubAccountRepo.On("GetByUserID", suite.ctx, userID, &exchangeID).Return(subAccounts, nil)

	// Act
	result, err := suite.subAccountService.GetUserSubAccounts(suite.ctx, userID, &exchangeID)

	// Assert
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), result, 2)
	assert.Equal(suite.T(), subAccounts[0].Name, result[0].Name)
	assert.Equal(suite.T(), subAccounts[1].Name, result[1].Name)
	mockExchangeRepo.AssertExpectations(suite.T())
	mockSubAccountRepo.AssertExpectations(suite.T())
}

func (suite *SubAccountServiceTestSuite) TestGetUserSubAccounts_NoExchangeFilter() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	infoJSON, _ := json.Marshal(map[string]interface{}{"test": "data"})

	subAccounts := []*models.SubAccount{
		{
			ID:        uuid.New(),
			UserID:    userID,
			Name:      "SubAccount 1",
			Symbol:    "BTC",
			Balance:   100.5,
			Info:      datatypes.JSON(infoJSON),
			CreatedAt: fixtures.FixedTime,
			UpdatedAt: fixtures.FixedTime,
		},
	}

	mockSubAccountRepo := suite.mockRepos.SubAccount.(*mocks.MockSubAccountRepository)
	mockSubAccountRepo.On("GetByUserID", suite.ctx, userID, (*uuid.UUID)(nil)).Return(subAccounts, nil)

	// Act
	result, err := suite.subAccountService.GetUserSubAccounts(suite.ctx, userID, nil)

	// Assert
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), result, 1)
	assert.Equal(suite.T(), subAccounts[0].Name, result[0].Name)
	mockSubAccountRepo.AssertExpectations(suite.T())
}

func (suite *SubAccountServiceTestSuite) TestGetSubAccount_Success() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	subAccountID := uuid.New()
	infoJSON, _ := json.Marshal(map[string]interface{}{"test": "data"})

	subAccount := &models.SubAccount{
		ID:        subAccountID,
		UserID:    userID,
		Name:      "Test SubAccount",
		Symbol:    "BTC",
		Balance:   100.5,
		Info:      datatypes.JSON(infoJSON),
		CreatedAt: fixtures.FixedTime,
		UpdatedAt: fixtures.FixedTime,
	}

	mockSubAccountRepo := suite.mockRepos.SubAccount.(*mocks.MockSubAccountRepository)
	mockSubAccountRepo.On("GetByID", suite.ctx, subAccountID).Return(subAccount, nil)

	// Act
	result, err := suite.subAccountService.GetSubAccount(suite.ctx, userID, subAccountID)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), subAccount.ID, result.ID)
	assert.Equal(suite.T(), subAccount.Name, result.Name)
	assert.Equal(suite.T(), subAccount.Balance, result.Balance)
	mockSubAccountRepo.AssertExpectations(suite.T())
}

func (suite *SubAccountServiceTestSuite) TestGetSubAccount_NotFound() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	subAccountID := uuid.New()

	mockSubAccountRepo := suite.mockRepos.SubAccount.(*mocks.MockSubAccountRepository)
	mockSubAccountRepo.On("GetByID", suite.ctx, subAccountID).Return(nil, nil)

	// Act
	result, err := suite.subAccountService.GetSubAccount(suite.ctx, userID, subAccountID)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "sub-account not found")
	mockSubAccountRepo.AssertExpectations(suite.T())
}

func (suite *SubAccountServiceTestSuite) TestUpdateSubAccount_Success() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	subAccountID := uuid.New()
	exchangeID := uuid.New()
	newName := "Updated SubAccount"
	newSymbol := "ETH"
	newBalance := 200.0

	req := &UpdateSubAccountRequest{
		Name:    &newName,
		Symbol:  &newSymbol,
		Balance: &newBalance,
	}

	subAccount := &models.SubAccount{
		ID:         subAccountID,
		UserID:     userID,
		ExchangeID: exchangeID,
		Name:       "Original SubAccount",
		Symbol:     "BTC",
		Balance:    100.0,
	}

	mockSubAccountRepo := suite.mockRepos.SubAccount.(*mocks.MockSubAccountRepository)
	mockSubAccountRepo.On("GetByID", suite.ctx, subAccountID).Return(subAccount, nil)
	mockSubAccountRepo.On("GetByUserID", suite.ctx, userID, &exchangeID).Return([]*models.SubAccount{subAccount}, nil)
	mockSubAccountRepo.On("Update", suite.ctx, mock.AnythingOfType("*models.SubAccount")).Return(nil)

	// Act
	result, err := suite.subAccountService.UpdateSubAccount(suite.ctx, userID, subAccountID, req)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), newName, result.Name)
	assert.Equal(suite.T(), newSymbol, result.Symbol)
	assert.Equal(suite.T(), newBalance, result.Balance)
	mockSubAccountRepo.AssertExpectations(suite.T())
}

func (suite *SubAccountServiceTestSuite) TestUpdateBalance_Success() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	subAccountID := uuid.New()
	req := &UpdateBalanceRequest{
		Amount:    50.0,
		Direction: "credit",
		Reason:    "Test deposit",
		Info:      map[string]interface{}{"source": "test"},
	}

	subAccount := &models.SubAccount{
		ID:      subAccountID,
		UserID:  userID,
		Balance: 100.0,
	}

	updatedSubAccount := &models.SubAccount{
		ID:      subAccountID,
		UserID:  userID,
		Balance: 150.0,
	}

	mockSubAccountRepo := suite.mockRepos.SubAccount.(*mocks.MockSubAccountRepository)
	// First call in UpdateBalance to verify ownership
	mockSubAccountRepo.On("GetByID", suite.ctx, subAccountID).Return(subAccount, nil).Once()
	// Call to UpdateBalance repo method
	mockSubAccountRepo.On("UpdateBalance", suite.ctx, subAccountID, 150.0, 50.0, "credit", "Test deposit", req.Info).Return(&subAccountID, nil)
	// Second call in GetSubAccount (called at end)
	mockSubAccountRepo.On("GetByID", suite.ctx, subAccountID).Return(updatedSubAccount, nil).Once()

	// Act
	result, err := suite.subAccountService.UpdateBalance(suite.ctx, userID, subAccountID, req)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), 150.0, result.Balance)
	mockSubAccountRepo.AssertExpectations(suite.T())
}

func (suite *SubAccountServiceTestSuite) TestUpdateBalance_InsufficientBalance() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	subAccountID := uuid.New()
	req := &UpdateBalanceRequest{
		Amount:    200.0,
		Direction: "debit",
		Reason:    "Test withdrawal",
	}

	subAccount := &models.SubAccount{
		ID:      subAccountID,
		UserID:  userID,
		Balance: 100.0, // Not enough for 200 debit
	}

	mockSubAccountRepo := suite.mockRepos.SubAccount.(*mocks.MockSubAccountRepository)
	mockSubAccountRepo.On("GetByID", suite.ctx, subAccountID).Return(subAccount, nil)

	// Act
	result, err := suite.subAccountService.UpdateBalance(suite.ctx, userID, subAccountID, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "insufficient balance")
	mockSubAccountRepo.AssertExpectations(suite.T())
}

func (suite *SubAccountServiceTestSuite) TestDeleteSubAccount_Success() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	subAccountID := uuid.New()
	subAccount := &models.SubAccount{
		ID:      subAccountID,
		UserID:  userID,
		Balance: 0.0, // Zero balance allows deletion
	}

	mockSubAccountRepo := suite.mockRepos.SubAccount.(*mocks.MockSubAccountRepository)
	mockSubAccountRepo.On("GetByID", suite.ctx, subAccountID).Return(subAccount, nil)
	mockSubAccountRepo.On("Delete", suite.ctx, subAccountID).Return(nil)

	// Act
	err := suite.subAccountService.DeleteSubAccount(suite.ctx, userID, subAccountID)

	// Assert
	assert.NoError(suite.T(), err)
	mockSubAccountRepo.AssertExpectations(suite.T())
}

func (suite *SubAccountServiceTestSuite) TestDeleteSubAccount_HasBalance() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	subAccountID := uuid.New()
	subAccount := &models.SubAccount{
		ID:      subAccountID,
		UserID:  userID,
		Balance: 100.0, // Positive balance prevents deletion
	}

	mockSubAccountRepo := suite.mockRepos.SubAccount.(*mocks.MockSubAccountRepository)
	mockSubAccountRepo.On("GetByID", suite.ctx, subAccountID).Return(subAccount, nil)

	// Act
	err := suite.subAccountService.DeleteSubAccount(suite.ctx, userID, subAccountID)

	// Assert
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "cannot delete sub-account with positive balance")
	mockSubAccountRepo.AssertExpectations(suite.T())
}

func (suite *SubAccountServiceTestSuite) TestGetSubAccountsBySymbol_Success() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	symbol := "BTC"
	infoJSON, _ := json.Marshal(map[string]interface{}{"test": "data"})

	subAccounts := []*models.SubAccount{
		{
			ID:        uuid.New(),
			UserID:    userID,
			Name:      "BTC Account 1",
			Symbol:    symbol,
			Balance:   100.0,
			Info:      datatypes.JSON(infoJSON),
			CreatedAt: fixtures.FixedTime,
			UpdatedAt: fixtures.FixedTime,
		},
		{
			ID:        uuid.New(),
			UserID:    userID,
			Name:      "BTC Account 2",
			Symbol:    symbol,
			Balance:   200.0,
			Info:      datatypes.JSON(infoJSON),
			CreatedAt: fixtures.FixedTime,
			UpdatedAt: fixtures.FixedTime,
		},
	}

	mockSubAccountRepo := suite.mockRepos.SubAccount.(*mocks.MockSubAccountRepository)
	mockSubAccountRepo.On("GetBySymbol", suite.ctx, userID, symbol).Return(subAccounts, nil)

	// Act
	result, err := suite.subAccountService.GetSubAccountsBySymbol(suite.ctx, userID, symbol)

	// Assert
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), result, 2)
	assert.Equal(suite.T(), symbol, result[0].Symbol)
	assert.Equal(suite.T(), symbol, result[1].Symbol)
	mockSubAccountRepo.AssertExpectations(suite.T())
}

func TestSubAccountServiceTestSuite(t *testing.T) {
	suite.Run(t, new(SubAccountServiceTestSuite))
}