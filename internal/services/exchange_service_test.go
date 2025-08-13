package services

import (
	"context"
	"encoding/json"
	"fmt"
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

type ExchangeServiceTestSuite struct {
	suite.Suite
	mockRepos       *repositories.Repositories
	exchangeService *ExchangeService
	ctx             context.Context
}

func (suite *ExchangeServiceTestSuite) SetupTest() {
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
	suite.exchangeService = NewExchangeService(suite.mockRepos)
}

func (suite *ExchangeServiceTestSuite) TestCreateExchange_Success() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	req := &CreateExchangeRequest{
		Name:      "Test Exchange",
		Type:      "binance",
		APIKey:    "test-api-key",
		APISecret: "test-api-secret",
	}

	mockExchangeRepo := suite.mockRepos.Exchange.(*mocks.MockExchangeRepository)
	mockExchangeRepo.On("GetByUserID", suite.ctx, userID).Return([]*models.Exchange{}, nil)
	mockExchangeRepo.On("Create", suite.ctx, mock.AnythingOfType("*models.Exchange")).Return(nil)

	// Act
	result, err := suite.exchangeService.CreateExchange(suite.ctx, userID, req)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), userID, result.UserID)
	assert.Equal(suite.T(), req.Name, result.Name)
	assert.Equal(suite.T(), req.Type, result.Type)
	assert.Equal(suite.T(), "active", result.Status)
	assert.Contains(suite.T(), result.APIKey, "****") // Masked API key
	mockExchangeRepo.AssertExpectations(suite.T())
}

func (suite *ExchangeServiceTestSuite) TestCreateExchange_MaxExchangesReached() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	req := &CreateExchangeRequest{
		Name:      "Test Exchange",
		Type:      "binance",
		APIKey:    "test-api-key",
		APISecret: "test-api-secret",
	}

	// Create 10 existing exchanges
	existingExchanges := make([]*models.Exchange, 10)
	for i := 0; i < 10; i++ {
		existingExchanges[i] = &models.Exchange{
			ID:     uuid.New(),
			UserID: userID,
			Name:   fmt.Sprintf("Exchange %d", i),
		}
	}

	mockExchangeRepo := suite.mockRepos.Exchange.(*mocks.MockExchangeRepository)
	mockExchangeRepo.On("GetByUserID", suite.ctx, userID).Return(existingExchanges, nil)

	// Act
	result, err := suite.exchangeService.CreateExchange(suite.ctx, userID, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "maximum number of exchanges reached")
	mockExchangeRepo.AssertExpectations(suite.T())
}

func (suite *ExchangeServiceTestSuite) TestCreateExchange_DuplicateName() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	req := &CreateExchangeRequest{
		Name:      "Existing Exchange",
		Type:      "binance",
		APIKey:    "test-api-key",
		APISecret: "test-api-secret",
	}

	existingExchange := &models.Exchange{
		ID:     uuid.New(),
		UserID: userID,
		Name:   "Existing Exchange",
	}

	mockExchangeRepo := suite.mockRepos.Exchange.(*mocks.MockExchangeRepository)
	mockExchangeRepo.On("GetByUserID", suite.ctx, userID).Return([]*models.Exchange{existingExchange}, nil)

	// Act
	result, err := suite.exchangeService.CreateExchange(suite.ctx, userID, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "exchange name already exists")
	mockExchangeRepo.AssertExpectations(suite.T())
}

func (suite *ExchangeServiceTestSuite) TestGetUserExchanges_Success() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	infoJSON, _ := json.Marshal(map[string]interface{}{"test": "data"})
	exchanges := []*models.Exchange{
		{
			ID:        uuid.New(),
			UserID:    userID,
			Name:      "Exchange 1",
			Type:      "binance",
			APIKey:    "test-api-key-1",
			Status:    "active",
			Info:      datatypes.JSON(infoJSON),
			CreatedAt: fixtures.FixedTime,
			UpdatedAt: fixtures.FixedTime,
		},
		{
			ID:        uuid.New(),
			UserID:    userID,
			Name:      "Exchange 2",
			Type:      "okx",
			APIKey:    "test-api-key-2",
			Status:    "inactive",
			Info:      datatypes.JSON(infoJSON),
			CreatedAt: fixtures.FixedTime,
			UpdatedAt: fixtures.FixedTime,
		},
	}

	mockExchangeRepo := suite.mockRepos.Exchange.(*mocks.MockExchangeRepository)
	mockExchangeRepo.On("GetByUserID", suite.ctx, userID).Return(exchanges, nil)

	// Act
	result, err := suite.exchangeService.GetUserExchanges(suite.ctx, userID)

	// Assert
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), result, 2)
	assert.Equal(suite.T(), exchanges[0].Name, result[0].Name)
	assert.Equal(suite.T(), exchanges[1].Name, result[1].Name)
	mockExchangeRepo.AssertExpectations(suite.T())
}

func (suite *ExchangeServiceTestSuite) TestGetExchange_Success() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	exchangeID := uuid.New()
	infoJSON, _ := json.Marshal(map[string]interface{}{"test": "data"})
	exchange := &models.Exchange{
		ID:        exchangeID,
		UserID:    userID,
		Name:      "Test Exchange",
		Type:      "binance",
		APIKey:    "test-api-key",
		Status:    "active",
		Info:      datatypes.JSON(infoJSON),
		CreatedAt: fixtures.FixedTime,
		UpdatedAt: fixtures.FixedTime,
	}

	mockExchangeRepo := suite.mockRepos.Exchange.(*mocks.MockExchangeRepository)
	mockExchangeRepo.On("GetByID", suite.ctx, exchangeID).Return(exchange, nil)

	// Act
	result, err := suite.exchangeService.GetExchange(suite.ctx, userID, exchangeID)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), exchange.ID, result.ID)
	assert.Equal(suite.T(), exchange.Name, result.Name)
	mockExchangeRepo.AssertExpectations(suite.T())
}

func (suite *ExchangeServiceTestSuite) TestGetExchange_NotFound() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	exchangeID := uuid.New()

	mockExchangeRepo := suite.mockRepos.Exchange.(*mocks.MockExchangeRepository)
	mockExchangeRepo.On("GetByID", suite.ctx, exchangeID).Return(nil, nil)

	// Act
	result, err := suite.exchangeService.GetExchange(suite.ctx, userID, exchangeID)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "exchange not found")
	mockExchangeRepo.AssertExpectations(suite.T())
}

func (suite *ExchangeServiceTestSuite) TestGetExchange_WrongUser() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	otherUserID := uuid.New()
	exchangeID := uuid.New()
	exchange := &models.Exchange{
		ID:     exchangeID,
		UserID: otherUserID, // Different user
		Name:   "Test Exchange",
	}

	mockExchangeRepo := suite.mockRepos.Exchange.(*mocks.MockExchangeRepository)
	mockExchangeRepo.On("GetByID", suite.ctx, exchangeID).Return(exchange, nil)

	// Act
	result, err := suite.exchangeService.GetExchange(suite.ctx, userID, exchangeID)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "exchange not found")
	mockExchangeRepo.AssertExpectations(suite.T())
}

func (suite *ExchangeServiceTestSuite) TestUpdateExchange_Success() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	exchangeID := uuid.New()
	newName := "Updated Exchange"
	newAPIKey := "new-api-key"
	newStatus := "inactive"

	req := &UpdateExchangeRequest{
		Name:   &newName,
		APIKey: &newAPIKey,
		Status: &newStatus,
	}

	exchange := &models.Exchange{
		ID:     exchangeID,
		UserID: userID,
		Name:   "Original Exchange",
		Type:   "binance",
		APIKey: "old-api-key",
		Status: "active",
	}

	mockExchangeRepo := suite.mockRepos.Exchange.(*mocks.MockExchangeRepository)
	mockExchangeRepo.On("GetByID", suite.ctx, exchangeID).Return(exchange, nil)
	mockExchangeRepo.On("GetByUserID", suite.ctx, userID).Return([]*models.Exchange{exchange}, nil)
	mockExchangeRepo.On("Update", suite.ctx, mock.AnythingOfType("*models.Exchange")).Return(nil)

	// Act
	result, err := suite.exchangeService.UpdateExchange(suite.ctx, userID, exchangeID, req)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), newName, result.Name)
	assert.Equal(suite.T(), newStatus, result.Status)
	mockExchangeRepo.AssertExpectations(suite.T())
}

func (suite *ExchangeServiceTestSuite) TestUpdateExchange_DuplicateName() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	exchangeID := uuid.New()
	otherExchangeID := uuid.New()
	newName := "Other Exchange"

	req := &UpdateExchangeRequest{
		Name: &newName,
	}

	exchange := &models.Exchange{
		ID:     exchangeID,
		UserID: userID,
		Name:   "Original Exchange",
	}

	otherExchange := &models.Exchange{
		ID:     otherExchangeID,
		UserID: userID,
		Name:   "Other Exchange",
	}

	mockExchangeRepo := suite.mockRepos.Exchange.(*mocks.MockExchangeRepository)
	mockExchangeRepo.On("GetByID", suite.ctx, exchangeID).Return(exchange, nil)
	mockExchangeRepo.On("GetByUserID", suite.ctx, userID).Return([]*models.Exchange{exchange, otherExchange}, nil)

	// Act
	result, err := suite.exchangeService.UpdateExchange(suite.ctx, userID, exchangeID, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "exchange name already exists")
	mockExchangeRepo.AssertExpectations(suite.T())
}

func (suite *ExchangeServiceTestSuite) TestDeleteExchange_Success() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	exchangeID := uuid.New()
	exchange := &models.Exchange{
		ID:     exchangeID,
		UserID: userID,
		Name:   "Test Exchange",
	}

	mockExchangeRepo := suite.mockRepos.Exchange.(*mocks.MockExchangeRepository)
	mockSubAccountRepo := suite.mockRepos.SubAccount.(*mocks.MockSubAccountRepository)

	mockExchangeRepo.On("GetByID", suite.ctx, exchangeID).Return(exchange, nil)
	mockSubAccountRepo.On("GetByExchangeID", suite.ctx, exchangeID).Return([]*models.SubAccount{}, nil)
	mockExchangeRepo.On("Delete", suite.ctx, exchangeID).Return(nil)

	// Act
	err := suite.exchangeService.DeleteExchange(suite.ctx, userID, exchangeID)

	// Assert
	assert.NoError(suite.T(), err)
	mockExchangeRepo.AssertExpectations(suite.T())
	mockSubAccountRepo.AssertExpectations(suite.T())
}

func (suite *ExchangeServiceTestSuite) TestDeleteExchange_HasSubAccounts() {
	// Arrange
	userID := fixtures.UserFixtures.ValidUser.ID
	exchangeID := uuid.New()
	exchange := &models.Exchange{
		ID:     exchangeID,
		UserID: userID,
		Name:   "Test Exchange",
	}

	subAccount := &models.SubAccount{
		ID:         uuid.New(),
		ExchangeID: exchangeID,
		Name:       "Test SubAccount",
	}

	mockExchangeRepo := suite.mockRepos.Exchange.(*mocks.MockExchangeRepository)
	mockSubAccountRepo := suite.mockRepos.SubAccount.(*mocks.MockSubAccountRepository)

	mockExchangeRepo.On("GetByID", suite.ctx, exchangeID).Return(exchange, nil)
	mockSubAccountRepo.On("GetByExchangeID", suite.ctx, exchangeID).Return([]*models.SubAccount{subAccount}, nil)

	// Act
	err := suite.exchangeService.DeleteExchange(suite.ctx, userID, exchangeID)

	// Assert
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "cannot delete exchange with existing sub-accounts")
	mockExchangeRepo.AssertExpectations(suite.T())
	mockSubAccountRepo.AssertExpectations(suite.T())
}

func (suite *ExchangeServiceTestSuite) TestGetExchangeByID_Success() {
	// Arrange
	exchangeID := uuid.New()
	infoJSON, _ := json.Marshal(map[string]interface{}{"test": "data"})
	exchange := &models.Exchange{
		ID:        exchangeID,
		UserID:    uuid.New(),
		Name:      "Test Exchange",
		Type:      "binance",
		APIKey:    "test-api-key",
		Status:    "active",
		Info:      datatypes.JSON(infoJSON),
		CreatedAt: fixtures.FixedTime,
		UpdatedAt: fixtures.FixedTime,
	}

	mockExchangeRepo := suite.mockRepos.Exchange.(*mocks.MockExchangeRepository)
	mockExchangeRepo.On("GetByID", suite.ctx, exchangeID).Return(exchange, nil)

	// Act
	result, err := suite.exchangeService.GetExchangeByID(suite.ctx, exchangeID)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), exchange.ID, result.ID)
	assert.Equal(suite.T(), exchange.Name, result.Name)
	mockExchangeRepo.AssertExpectations(suite.T())
}

func (suite *ExchangeServiceTestSuite) TestConvertToExchangeResponse_MasksAPIKey() {
	// Arrange
	longAPIKey := "abcd1234567890efgh"
	shortAPIKey := "short"
	infoJSON, _ := json.Marshal(map[string]interface{}{"test": "data"})

	exchangeLong := &models.Exchange{
		ID:        uuid.New(),
		APIKey:    longAPIKey,
		Info:      datatypes.JSON(infoJSON),
		CreatedAt: fixtures.FixedTime,
		UpdatedAt: fixtures.FixedTime,
	}

	exchangeShort := &models.Exchange{
		ID:        uuid.New(),
		APIKey:    shortAPIKey,
		Info:      datatypes.JSON(infoJSON),
		CreatedAt: fixtures.FixedTime,
		UpdatedAt: fixtures.FixedTime,
	}

	// Act
	resultLong := suite.exchangeService.convertToExchangeResponse(exchangeLong)
	resultShort := suite.exchangeService.convertToExchangeResponse(exchangeShort)

	// Assert
	assert.Equal(suite.T(), "abcd****efgh", resultLong.APIKey)
	assert.Equal(suite.T(), shortAPIKey, resultShort.APIKey) // Short keys not masked
}

func TestExchangeServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ExchangeServiceTestSuite))
}