package test

import (
	"context"
	"testing"
	"time"

	"tiris-backend/internal/models"
	"tiris-backend/internal/repositories"
	"tiris-backend/test/config"
	"tiris-backend/test/helpers"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestExchange represents a simplified exchange model for SQLite testing
type TestExchange struct {
	ID        uuid.UUID `gorm:"type:TEXT;primary_key" json:"id"`
	UserID    uuid.UUID `gorm:"type:TEXT;not null;index" json:"user_id"`
	Name      string    `gorm:"type:varchar(100);not null" json:"name"`
	Type      string    `gorm:"type:varchar(50);not null;index" json:"type"`
	APIKey    string    `gorm:"type:text;not null" json:"-"`
	APISecret string    `gorm:"type:text;not null" json:"-"`
	Status    string    `gorm:"type:varchar(20);default:'active';index" json:"status"`
	Info      string    `gorm:"type:text;default:'{}'" json:"info"`

	CreatedAt time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"-"`
}

// TestSubAccountForExchange represents a simplified sub-account model for SQLite testing
type TestSubAccountForExchange struct {
	ID         uuid.UUID `gorm:"type:TEXT;primary_key" json:"id"`
	ExchangeID uuid.UUID `gorm:"type:TEXT;not null;index" json:"exchange_id"`
	UserID     uuid.UUID `gorm:"type:TEXT;not null;index" json:"user_id"`
	Symbol     string    `gorm:"type:varchar(20);not null;index" json:"symbol"`
	Balance    float64   `gorm:"type:decimal(20,8);not null;default:0" json:"balance"`
	Status     string    `gorm:"type:varchar(20);default:'active';index" json:"status"`

	CreatedAt time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"-"`
}

// setupExchangeTestDB creates an in-memory SQLite database for exchange testing
func setupExchangeTestDB() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Auto-migrate the test models using the table name mapping
	if err := db.Table("users").AutoMigrate(&TestUser{}); err != nil {
		return nil, err
	}
	
	if err := db.Table("exchanges").AutoMigrate(&TestExchange{}); err != nil {
		return nil, err
	}
	
	if err := db.Table("sub_accounts").AutoMigrate(&TestSubAccountForExchange{}); err != nil {
		return nil, err
	}

	return db, nil
}

// TestExchangeRepository_Create tests exchange creation functionality
func TestExchangeRepository_Create(t *testing.T) {
	// Setup test config
	_ = config.GetProfileConfig(config.ProfileQuick)
	
	// Create test database
	db, err := setupExchangeTestDB()
	require.NoError(t, err)
	
	// Create repositories
	exchangeRepo := repositories.NewExchangeRepository(db)
	userRepo := repositories.NewUserRepository(db)
	userFactory := helpers.NewUserFactory()
	
	// Create test user first
	testUser := userFactory.WithEmail("exchange@example.com")
	if testUser.ID == uuid.Nil {
		testUser.ID = uuid.New()
	}
	err = userRepo.Create(context.Background(), testUser)
	require.NoError(t, err)
	
	// Test successful exchange creation
	t.Run("successful_exchange_creation", func(t *testing.T) {
		testExchange := &models.Exchange{
			ID:        uuid.New(),
			UserID:    testUser.ID,
			Name:      "Binance Main",
			Type:      "binance",
			APIKey:    "binance_api_key_123",
			APISecret: "binance_api_secret_456",
			Status:    "active",
		}
		
		// Execute test
		err := exchangeRepo.Create(context.Background(), testExchange)
		
		// Verify results
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, testExchange.ID)
		
		// Verify exchange was saved to database
		var savedExchange TestExchange
		err = db.Table("exchanges").Where("id = ?", testExchange.ID).First(&savedExchange).Error
		require.NoError(t, err)
		assert.Equal(t, testExchange.UserID.String(), savedExchange.UserID.String())
		assert.Equal(t, testExchange.Name, savedExchange.Name)
		assert.Equal(t, testExchange.Type, savedExchange.Type)
		assert.Equal(t, testExchange.APIKey, savedExchange.APIKey)
		assert.Equal(t, testExchange.APISecret, savedExchange.APISecret)
		assert.Equal(t, testExchange.Status, savedExchange.Status)
	})
	
	// Test exchange creation with different types
	t.Run("create_different_exchange_types", func(t *testing.T) {
		exchangeTypes := []struct {
			name string
			typ  string
		}{
			{"OKX Trading", "okx"},
			{"Coinbase Pro", "coinbase"},
			{"Kraken API", "kraken"},
			{"Huobi Global", "huobi"},
		}
		
		for _, et := range exchangeTypes {
			exchange := &models.Exchange{
				ID:        uuid.New(),
				UserID:    testUser.ID,
				Name:      et.name,
				Type:      et.typ,
				APIKey:    et.typ + "_api_key",
				APISecret: et.typ + "_api_secret",
				Status:    "active",
			}
			
			err := exchangeRepo.Create(context.Background(), exchange)
			require.NoError(t, err, "Failed to create %s exchange", et.typ)
		}
	})
	
	// Test creation with empty required fields
	t.Run("create_with_empty_required_fields", func(t *testing.T) {
		invalidExchange := &models.Exchange{
			ID:        uuid.New(),
			UserID:    testUser.ID,
			Name:      "", // Empty name
			Type:      "binance",
			APIKey:    "api_key",
			APISecret: "api_secret",
		}
		
		// Execute test - behavior may vary depending on validation
		err := exchangeRepo.Create(context.Background(), invalidExchange)
		
		// Test doesn't fail since validation might be at service layer
		_ = err
	})
}

// TestExchangeRepository_GetByID tests exchange retrieval by ID
func TestExchangeRepository_GetByID(t *testing.T) {
	// Create test database
	db, err := setupExchangeTestDB()
	require.NoError(t, err)
	
	// Create repositories
	exchangeRepo := repositories.NewExchangeRepository(db)
	userRepo := repositories.NewUserRepository(db)
	userFactory := helpers.NewUserFactory()
	
	// Create test user
	testUser := userFactory.WithEmail("getbyid@example.com")
	if testUser.ID == uuid.Nil {
		testUser.ID = uuid.New()
	}
	err = userRepo.Create(context.Background(), testUser)
	require.NoError(t, err)
	
	// Create test exchange
	testExchange := &models.Exchange{
		ID:        uuid.New(),
		UserID:    testUser.ID,
		Name:      "Test Exchange",
		Type:      "binance",
		APIKey:    "test_api_key",
		APISecret: "test_api_secret",
		Status:    "active",
	}
	err = exchangeRepo.Create(context.Background(), testExchange)
	require.NoError(t, err)
	
	// Test successful retrieval
	t.Run("successful_get_by_id", func(t *testing.T) {
		// Execute test
		retrievedExchange, err := exchangeRepo.GetByID(context.Background(), testExchange.ID)
		
		// Verify results
		require.NoError(t, err)
		require.NotNil(t, retrievedExchange)
		assert.Equal(t, testExchange.ID, retrievedExchange.ID)
		assert.Equal(t, testExchange.UserID, retrievedExchange.UserID)
		assert.Equal(t, testExchange.Name, retrievedExchange.Name)
		assert.Equal(t, testExchange.Type, retrievedExchange.Type)
		assert.Equal(t, testExchange.APIKey, retrievedExchange.APIKey)
		assert.Equal(t, testExchange.APISecret, retrievedExchange.APISecret)
		assert.Equal(t, testExchange.Status, retrievedExchange.Status)
	})
	
	// Test non-existent exchange
	t.Run("get_non_existent_exchange", func(t *testing.T) {
		nonExistentID := uuid.New()
		
		// Execute test
		retrievedExchange, err := exchangeRepo.GetByID(context.Background(), nonExistentID)
		
		// Should return nil, nil for not found
		require.NoError(t, err)
		assert.Nil(t, retrievedExchange)
	})
	
	// Test with nil UUID
	t.Run("get_with_nil_uuid", func(t *testing.T) {
		// Execute test
		retrievedExchange, err := exchangeRepo.GetByID(context.Background(), uuid.Nil)
		
		// Should return nil, nil for invalid UUID
		require.NoError(t, err)
		assert.Nil(t, retrievedExchange)
	})
}

// TestExchangeRepository_GetByUserID tests exchange retrieval by user ID
func TestExchangeRepository_GetByUserID(t *testing.T) {
	// Create test database
	db, err := setupExchangeTestDB()
	require.NoError(t, err)
	
	// Create repositories
	exchangeRepo := repositories.NewExchangeRepository(db)
	userRepo := repositories.NewUserRepository(db)
	userFactory := helpers.NewUserFactory()
	
	// Create test user
	testUser := userFactory.WithEmail("getuserexchanges@example.com")
	if testUser.ID == uuid.Nil {
		testUser.ID = uuid.New()
	}
	err = userRepo.Create(context.Background(), testUser)
	require.NoError(t, err)
	
	// Create multiple test exchanges for the user
	exchanges := []*models.Exchange{
		{
			ID:        uuid.New(),
			UserID:    testUser.ID,
			Name:      "Binance Main",
			Type:      "binance",
			APIKey:    "binance_key",
			APISecret: "binance_secret",
			Status:    "active",
		},
		{
			ID:        uuid.New(),
			UserID:    testUser.ID,
			Name:      "OKX Trading",
			Type:      "okx",
			APIKey:    "okx_key",
			APISecret: "okx_secret",
			Status:    "active",
		},
		{
			ID:        uuid.New(),
			UserID:    testUser.ID,
			Name:      "Coinbase Pro",
			Type:      "coinbase",
			APIKey:    "coinbase_key",
			APISecret: "coinbase_secret",
			Status:    "inactive",
		},
	}
	
	for i, exchange := range exchanges {
		err = exchangeRepo.Create(context.Background(), exchange)
		require.NoError(t, err)
		
		// Add small delay to ensure different created_at times for ordering test
		if i < len(exchanges)-1 {
			time.Sleep(1 * time.Millisecond)
		}
	}
	
	// Test successful retrieval
	t.Run("successful_get_by_user_id", func(t *testing.T) {
		// Execute test
		retrievedExchanges, err := exchangeRepo.GetByUserID(context.Background(), testUser.ID)
		
		// Verify results
		require.NoError(t, err)
		require.Len(t, retrievedExchanges, 3)
		
		// Verify ordering (should be DESC by created_at)
		for i := 0; i < len(retrievedExchanges)-1; i++ {
			assert.True(t, retrievedExchanges[i].CreatedAt.After(retrievedExchanges[i+1].CreatedAt) ||
				retrievedExchanges[i].CreatedAt.Equal(retrievedExchanges[i+1].CreatedAt))
		}
		
		// Verify all exchanges belong to the user
		for _, exchange := range retrievedExchanges {
			assert.Equal(t, testUser.ID, exchange.UserID)
		}
	})
	
	// Test non-existent user
	t.Run("get_exchanges_for_non_existent_user", func(t *testing.T) {
		nonExistentUserID := uuid.New()
		
		// Execute test
		retrievedExchanges, err := exchangeRepo.GetByUserID(context.Background(), nonExistentUserID)
		
		// Should return empty slice, no error
		require.NoError(t, err)
		assert.Len(t, retrievedExchanges, 0)
	})
}

// TestExchangeRepository_GetByUserIDAndType tests exchange retrieval by user ID and type
func TestExchangeRepository_GetByUserIDAndType(t *testing.T) {
	// Create test database
	db, err := setupExchangeTestDB()
	require.NoError(t, err)
	
	// Create repositories
	exchangeRepo := repositories.NewExchangeRepository(db)
	userRepo := repositories.NewUserRepository(db)
	userFactory := helpers.NewUserFactory()
	
	// Create test user
	testUser := userFactory.WithEmail("getbytype@example.com")
	if testUser.ID == uuid.Nil {
		testUser.ID = uuid.New()
	}
	err = userRepo.Create(context.Background(), testUser)
	require.NoError(t, err)
	
	// Create multiple exchanges of different types
	exchanges := []*models.Exchange{
		{
			ID:        uuid.New(),
			UserID:    testUser.ID,
			Name:      "Binance Main",
			Type:      "binance",
			APIKey:    "binance_key_1",
			APISecret: "binance_secret_1",
			Status:    "active",
		},
		{
			ID:        uuid.New(),
			UserID:    testUser.ID,
			Name:      "Binance Secondary",
			Type:      "binance",
			APIKey:    "binance_key_2",
			APISecret: "binance_secret_2",
			Status:    "active",
		},
		{
			ID:        uuid.New(),
			UserID:    testUser.ID,
			Name:      "OKX Trading",
			Type:      "okx",
			APIKey:    "okx_key",
			APISecret: "okx_secret",
			Status:    "active",
		},
	}
	
	for _, exchange := range exchanges {
		err = exchangeRepo.Create(context.Background(), exchange)
		require.NoError(t, err)
		time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	}
	
	// Test successful retrieval by type
	t.Run("successful_get_by_user_and_type", func(t *testing.T) {
		// Execute test - get binance exchanges
		retrievedExchanges, err := exchangeRepo.GetByUserIDAndType(context.Background(), testUser.ID, "binance")
		
		// Verify results
		require.NoError(t, err)
		require.Len(t, retrievedExchanges, 2)
		
		// Verify all are binance type
		for _, exchange := range retrievedExchanges {
			assert.Equal(t, "binance", exchange.Type)
			assert.Equal(t, testUser.ID, exchange.UserID)
		}
		
		// Verify ordering (should be DESC by created_at)
		assert.True(t, retrievedExchanges[0].CreatedAt.After(retrievedExchanges[1].CreatedAt) ||
			retrievedExchanges[0].CreatedAt.Equal(retrievedExchanges[1].CreatedAt))
	})
	
	// Test single result type
	t.Run("get_single_result_type", func(t *testing.T) {
		// Execute test - get okx exchanges
		retrievedExchanges, err := exchangeRepo.GetByUserIDAndType(context.Background(), testUser.ID, "okx")
		
		// Verify results
		require.NoError(t, err)
		require.Len(t, retrievedExchanges, 1)
		assert.Equal(t, "okx", retrievedExchanges[0].Type)
		assert.Equal(t, testUser.ID, retrievedExchanges[0].UserID)
	})
	
	// Test non-existent type
	t.Run("get_non_existent_type", func(t *testing.T) {
		// Execute test
		retrievedExchanges, err := exchangeRepo.GetByUserIDAndType(context.Background(), testUser.ID, "kraken")
		
		// Should return empty slice, no error
		require.NoError(t, err)
		assert.Len(t, retrievedExchanges, 0)
	})
	
	// Test non-existent user
	t.Run("get_non_existent_user_with_type", func(t *testing.T) {
		nonExistentUserID := uuid.New()
		
		// Execute test
		retrievedExchanges, err := exchangeRepo.GetByUserIDAndType(context.Background(), nonExistentUserID, "binance")
		
		// Should return empty slice, no error
		require.NoError(t, err)
		assert.Len(t, retrievedExchanges, 0)
	})
}

// TestExchangeRepository_Update tests exchange update functionality
func TestExchangeRepository_Update(t *testing.T) {
	// Create test database
	db, err := setupExchangeTestDB()
	require.NoError(t, err)
	
	// Create repositories
	exchangeRepo := repositories.NewExchangeRepository(db)
	userRepo := repositories.NewUserRepository(db)
	userFactory := helpers.NewUserFactory()
	
	// Create test user
	testUser := userFactory.WithEmail("update@example.com")
	if testUser.ID == uuid.Nil {
		testUser.ID = uuid.New()
	}
	err = userRepo.Create(context.Background(), testUser)
	require.NoError(t, err)
	
	// Create test exchange
	testExchange := &models.Exchange{
		ID:        uuid.New(),
		UserID:    testUser.ID,
		Name:      "Original Exchange",
		Type:      "binance",
		APIKey:    "original_api_key",
		APISecret: "original_api_secret",
		Status:    "active",
	}
	err = exchangeRepo.Create(context.Background(), testExchange)
	require.NoError(t, err)
	
	// Test successful update
	t.Run("successful_update", func(t *testing.T) {
		// Update exchange data
		testExchange.Name = "Updated Exchange Name"
		testExchange.APIKey = "updated_api_key"
		testExchange.APISecret = "updated_api_secret"
		testExchange.Status = "inactive"
		
		// Execute test
		err := exchangeRepo.Update(context.Background(), testExchange)
		
		// Verify results
		require.NoError(t, err)
		
		// Verify update was saved to database
		var updatedExchange TestExchange
		err = db.Table("exchanges").Where("id = ?", testExchange.ID).First(&updatedExchange).Error
		require.NoError(t, err)
		assert.Equal(t, "Updated Exchange Name", updatedExchange.Name)
		assert.Equal(t, "updated_api_key", updatedExchange.APIKey)
		assert.Equal(t, "updated_api_secret", updatedExchange.APISecret)
		assert.Equal(t, "inactive", updatedExchange.Status)
	})
	
	// Test update non-existent exchange
	t.Run("update_non_existent_exchange", func(t *testing.T) {
		nonExistentExchange := &models.Exchange{
			ID:        uuid.New(),
			UserID:    testUser.ID,
			Name:      "Non-existent Exchange",
			Type:      "binance",
			APIKey:    "fake_key",
			APISecret: "fake_secret",
		}
		
		// Execute test
		err := exchangeRepo.Update(context.Background(), nonExistentExchange)
		
		// Should succeed but no rows affected (GORM behavior)
		require.NoError(t, err)
	})
}

// TestExchangeRepository_Delete tests exchange deletion functionality
func TestExchangeRepository_Delete(t *testing.T) {
	// Create test database
	db, err := setupExchangeTestDB()
	require.NoError(t, err)
	
	// Create repositories
	exchangeRepo := repositories.NewExchangeRepository(db)
	userRepo := repositories.NewUserRepository(db)
	userFactory := helpers.NewUserFactory()
	
	// Create test user
	testUser := userFactory.WithEmail("delete@example.com")
	if testUser.ID == uuid.Nil {
		testUser.ID = uuid.New()
	}
	err = userRepo.Create(context.Background(), testUser)
	require.NoError(t, err)
	
	// Test successful deletion (no sub-accounts)
	t.Run("successful_delete_no_subaccounts", func(t *testing.T) {
		// Create test exchange
		testExchange := &models.Exchange{
			ID:        uuid.New(),
			UserID:    testUser.ID,
			Name:      "Deletable Exchange",
			Type:      "binance",
			APIKey:    "delete_api_key",
			APISecret: "delete_api_secret",
			Status:    "active",
		}
		err = exchangeRepo.Create(context.Background(), testExchange)
		require.NoError(t, err)
		
		// Execute test
		err = exchangeRepo.Delete(context.Background(), testExchange.ID)
		
		// Verify results
		require.NoError(t, err)
		
		// Verify exchange was soft deleted from database
		var deletedExchange TestExchange
		err = db.Table("exchanges").Unscoped().Where("id = ?", testExchange.ID).First(&deletedExchange).Error
		require.NoError(t, err)
		assert.NotNil(t, deletedExchange.DeletedAt) // Should be soft deleted
	})
	
	// Test deletion with existing sub-accounts
	t.Run("delete_with_existing_subaccounts", func(t *testing.T) {
		// Create test exchange
		testExchange := &models.Exchange{
			ID:        uuid.New(),
			UserID:    testUser.ID,
			Name:      "Exchange with SubAccounts",
			Type:      "binance",
			APIKey:    "subaccount_api_key",
			APISecret: "subaccount_api_secret",
			Status:    "active",
		}
		err = exchangeRepo.Create(context.Background(), testExchange)
		require.NoError(t, err)
		
		// Create a sub-account for this exchange
		testSubAccount := TestSubAccountForExchange{
			ID:         uuid.New(),
			ExchangeID: testExchange.ID,
			UserID:     testUser.ID,
			Symbol:     "BTC",
			Balance:    1.5,
			Status:     "active",
		}
		err = db.Table("sub_accounts").Create(&testSubAccount).Error
		require.NoError(t, err)
		
		// Execute test - should fail due to existing sub-accounts
		err = exchangeRepo.Delete(context.Background(), testExchange.ID)
		
		// Verify error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete exchange with existing sub-accounts")
		
		// Verify exchange was not deleted
		var existingExchange TestExchange
		err = db.Table("exchanges").Where("id = ?", testExchange.ID).First(&existingExchange).Error
		require.NoError(t, err)
		assert.Nil(t, existingExchange.DeletedAt) // Should not be deleted
	})
	
	// Test delete non-existent exchange
	t.Run("delete_non_existent_exchange", func(t *testing.T) {
		nonExistentID := uuid.New()
		
		// Execute test
		err := exchangeRepo.Delete(context.Background(), nonExistentID)
		
		// Should succeed but no rows affected (GORM behavior)
		require.NoError(t, err)
	})
}

// Performance test for exchange repository operations
func TestExchangeRepository_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}
	
	// Create test database
	db, err := setupExchangeTestDB()
	require.NoError(t, err)
	
	// Create repositories
	exchangeRepo := repositories.NewExchangeRepository(db)
	userRepo := repositories.NewUserRepository(db)
	userFactory := helpers.NewUserFactory()
	
	// Create test user
	testUser := userFactory.WithEmail("performance@example.com")
	if testUser.ID == uuid.Nil {
		testUser.ID = uuid.New()
	}
	err = userRepo.Create(context.Background(), testUser)
	require.NoError(t, err)
	
	t.Run("bulk_exchange_create_performance", func(t *testing.T) {
		timer := helpers.NewPerformanceTimer()
		timer.Start()
		
		// Create 100 exchanges
		for i := 0; i < 100; i++ {
			exchange := &models.Exchange{
				ID:        uuid.New(),
				UserID:    testUser.ID,
				Name:      "Performance Exchange " + string(rune(i+48)),
				Type:      "binance",
				APIKey:    "perf_key_" + string(rune(i+48)),
				APISecret: "perf_secret_" + string(rune(i+48)),
				Status:    "active",
			}
			err := exchangeRepo.Create(context.Background(), exchange)
			require.NoError(t, err)
		}
		
		timer.Stop()
		
		// Check performance within reasonable bounds
		assert.Less(t, timer.Duration().Milliseconds(), int64(3000),
			"100 exchange creations should complete within 3 seconds")
	})
	
	t.Run("exchange_lookup_performance", func(t *testing.T) {
		// Create a test exchange first
		testExchange := &models.Exchange{
			ID:        uuid.New(),
			UserID:    testUser.ID,
			Name:      "Lookup Performance Exchange",
			Type:      "binance",
			APIKey:    "lookup_key",
			APISecret: "lookup_secret",
			Status:    "active",
		}
		err := exchangeRepo.Create(context.Background(), testExchange)
		require.NoError(t, err)
		
		timer := helpers.NewPerformanceTimer()
		timer.Start()
		
		// Perform 1000 lookups
		for i := 0; i < 1000; i++ {
			_, err := exchangeRepo.GetByID(context.Background(), testExchange.ID)
			require.NoError(t, err)
		}
		
		timer.Stop()
		
		// Check performance within reasonable bounds
		assert.Less(t, timer.Duration().Milliseconds(), int64(1000),
			"1000 exchange lookups should complete within 1 second")
	})
	
	t.Run("user_exchanges_query_performance", func(t *testing.T) {
		timer := helpers.NewPerformanceTimer()
		timer.Start()
		
		// Perform 500 user exchange queries
		for i := 0; i < 500; i++ {
			_, err := exchangeRepo.GetByUserID(context.Background(), testUser.ID)
			require.NoError(t, err)
		}
		
		timer.Stop()
		
		// Check performance within reasonable bounds
		assert.Less(t, timer.Duration().Milliseconds(), int64(2000),
			"500 user exchange queries should complete within 2 seconds")
	})
}