package test

import (
	"context"
	"testing"
	"time"

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
	"gorm.io/gorm"
)

// TestTradingLogService_EventTime tests event_time field functionality
func TestTradingLogService_EventTime(t *testing.T) {
	// Setup test config
	_ = config.GetProfileConfig(config.ProfileQuick)

	// Create mocks
	mockTradingLogRepo := &mocks.MockTradingLogRepository{}
	mockTradingRepo := &mocks.MockTradingRepository{}
	mockSubAccountRepo := &mocks.MockSubAccountRepository{}
	mockTransactionRepo := &mocks.MockTransactionRepository{}

	// Create repositories with mocks
	repos := &repositories.Repositories{
		User:            &mocks.MockUserRepository{},
		Trading:        mockTradingRepo,
		SubAccount:      mockSubAccountRepo,
		Transaction:     mockTransactionRepo,
		TradingLog:      mockTradingLogRepo,
		OAuthToken:      &mocks.MockOAuthTokenRepository{},
		EventProcessing: &mocks.MockEventProcessingRepository{},
	}

	// Create mock database - use nil since we're mocking the repository layer
	var mockDB *gorm.DB

	// Create service
	tradingLogService := services.NewTradingLogService(repos, mockDB)

	// Create test data
	userID := uuid.New()
	tradingID := uuid.New()
	tradingFactory := helpers.NewTradingFactory()
	testTrading := tradingFactory.WithUserID(userID)
	testTrading.ID = tradingID

	t.Run("event_time_null_by_default", func(t *testing.T) {
		request := &services.CreateTradingLogRequest{
			TradingID: tradingID,
			Type:      "trade",
			Source:    "manual",
			Message:   "Manual trade executed successfully",
			Info:      map[string]interface{}{"symbol": "BTCUSDT", "side": "buy"},
			// EventTime not provided - should be null
		}

		// Setup mock expectations
		mockTradingRepo.On("GetByID", mock.Anything, tradingID).
			Return(testTrading, nil).Once()
		
		var capturedTradingLog *models.TradingLog
		mockTradingLogRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.TradingLog")).
			Run(func(args mock.Arguments) {
				capturedTradingLog = args.Get(1).(*models.TradingLog)
			}).
			Return(nil).Once()

		// Execute test
		result, err := tradingLogService.CreateTradingLog(context.Background(), userID, request)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Nil(t, capturedTradingLog.EventTime, "EventTime should be nil when not provided")

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
		mockTradingLogRepo.AssertExpectations(t)
	})

	t.Run("event_time_set_for_live_trading", func(t *testing.T) {
		eventTime := time.Now().UTC()
		request := &services.CreateTradingLogRequest{
			TradingID: tradingID,
			Type:      "trade",
			Source:    "bot",
			Message:   "Live trading bot executed trade",
			EventTime: &eventTime,
			Info:      map[string]interface{}{"symbol": "ETHUSDT", "side": "sell"},
		}

		// Setup mock expectations
		mockTradingRepo.On("GetByID", mock.Anything, tradingID).
			Return(testTrading, nil).Once()
		
		var capturedTradingLog *models.TradingLog
		mockTradingLogRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.TradingLog")).
			Run(func(args mock.Arguments) {
				capturedTradingLog = args.Get(1).(*models.TradingLog)
			}).
			Return(nil).Once()

		// Execute test
		result, err := tradingLogService.CreateTradingLog(context.Background(), userID, request)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, capturedTradingLog.EventTime, "EventTime should be set when provided")
		assert.WithinDuration(t, eventTime, *capturedTradingLog.EventTime, time.Second, "EventTime should match the provided time")

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
		mockTradingLogRepo.AssertExpectations(t)
	})

	t.Run("event_time_set_for_backtesting", func(t *testing.T) {
		// Historical time for backtesting (1 year ago)
		eventTime := time.Now().UTC().AddDate(-1, 0, 0)
		request := &services.CreateTradingLogRequest{
			TradingID: tradingID,
			Type:      "custom", // Use non-business logic type to avoid database transaction issues
			Source:    "bot",
			Message:   "Backtesting custom event",
			EventTime: &eventTime,
			Info: map[string]interface{}{
				"backtest_symbol": "ETH",
				"historical_data": true,
				"notes":           "Historical event from backtesting",
			},
		}

		// Setup mock expectations
		mockTradingRepo.On("GetByID", mock.Anything, tradingID).
			Return(testTrading, nil).Once()
		
		var capturedTradingLog *models.TradingLog
		mockTradingLogRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.TradingLog")).
			Run(func(args mock.Arguments) {
				capturedTradingLog = args.Get(1).(*models.TradingLog)
			}).
			Return(nil).Once()

		// Execute test
		result, err := tradingLogService.CreateTradingLog(context.Background(), userID, request)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, capturedTradingLog.EventTime, "EventTime should be set for backtesting")
		assert.WithinDuration(t, eventTime, *capturedTradingLog.EventTime, time.Second, "EventTime should match the historical time")
		
		// Verify that EventTime is significantly different from current time (backtesting scenario)
		timeDiff := time.Since(*capturedTradingLog.EventTime)
		assert.True(t, timeDiff > 350*24*time.Hour, "EventTime should be historical (more than 350 days ago)")

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
		mockTradingLogRepo.AssertExpectations(t)
	})

	// Note: Business logic types (long, short, stop_loss, deposit, withdraw) are tested in integration tests
	// since they require database transactions which are difficult to mock properly

	t.Run("event_time_validation", func(t *testing.T) {
		// Test with very old timestamp
		veryOldTime := time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)
		request := &services.CreateTradingLogRequest{
			TradingID: tradingID,
			Type:      "custom",
			Source:    "manual",
			Message:   "Test with very old timestamp",
			EventTime: &veryOldTime,
			Info:      map[string]interface{}{"test": "value"},
		}

		// Setup mock expectations
		mockTradingRepo.On("GetByID", mock.Anything, tradingID).
			Return(testTrading, nil).Once()
		
		var capturedTradingLog *models.TradingLog
		mockTradingLogRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.TradingLog")).
			Run(func(args mock.Arguments) {
				capturedTradingLog = args.Get(1).(*models.TradingLog)
			}).
			Return(nil).Once()

		// Execute test
		result, err := tradingLogService.CreateTradingLog(context.Background(), userID, request)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, capturedTradingLog.EventTime, "EventTime should be set even for very old timestamps")
		assert.Equal(t, veryOldTime.Unix(), capturedTradingLog.EventTime.Unix(), "EventTime should preserve very old timestamps")

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
		mockTradingLogRepo.AssertExpectations(t)
	})

	t.Run("event_time_future_timestamp", func(t *testing.T) {
		// Test with future timestamp (should be allowed)
		futureTime := time.Now().UTC().Add(24 * time.Hour)
		request := &services.CreateTradingLogRequest{
			TradingID: tradingID,
			Type:      "custom",
			Source:    "manual",
			Message:   "Test with future timestamp",
			EventTime: &futureTime,
			Info:      map[string]interface{}{"test": "future_value"},
		}

		// Setup mock expectations
		mockTradingRepo.On("GetByID", mock.Anything, tradingID).
			Return(testTrading, nil).Once()
		
		var capturedTradingLog *models.TradingLog
		mockTradingLogRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.TradingLog")).
			Run(func(args mock.Arguments) {
				capturedTradingLog = args.Get(1).(*models.TradingLog)
			}).
			Return(nil).Once()

		// Execute test
		result, err := tradingLogService.CreateTradingLog(context.Background(), userID, request)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, capturedTradingLog.EventTime, "EventTime should be set for future timestamps")
		assert.WithinDuration(t, futureTime, *capturedTradingLog.EventTime, time.Second, "EventTime should preserve future timestamps")

		// Verify mock expectations
		mockTradingRepo.AssertExpectations(t)
		mockTradingLogRepo.AssertExpectations(t)
	})
}

// TestTradingLogService_EventTimeResponse tests that event_time is properly included in responses
func TestTradingLogService_EventTimeResponse(t *testing.T) {
	// This test would be implemented when we update the service to handle response formatting
	t.Skip("TODO: Implement after updating service response handling")
}

// TestTradingLogService_EventTimeIndexes tests that event_time indexes are used properly
func TestTradingLogService_EventTimeIndexes(t *testing.T) {
	// This test would be implemented as integration test with real database
	t.Skip("TODO: Implement as integration test after database migration")
}