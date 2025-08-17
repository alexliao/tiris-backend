package test

import (
	"context"
	"strings"
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

// TestSubAccountModel represents a simplified sub-account model for SQLite testing
type TestSubAccountModel struct {
	ID         uuid.UUID `gorm:"type:TEXT;primary_key" json:"id"`
	UserID     uuid.UUID `gorm:"type:TEXT;not null;index" json:"user_id"`
	ExchangeID uuid.UUID `gorm:"type:TEXT;not null;index" json:"exchange_id"`
	Name       string    `gorm:"type:varchar(100);not null" json:"name"`
	Symbol     string    `gorm:"type:varchar(20);not null;index" json:"symbol"`
	Balance    float64   `gorm:"type:decimal(20,8);default:0" json:"balance"`
	Info       string    `gorm:"type:text;default:'{}'" json:"info"`

	CreatedAt time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"-"`
}

// setupSubAccountTestDB creates an in-memory SQLite database for sub-account testing
func setupSubAccountTestDB() (*gorm.DB, error) {
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
	
	if err := db.Table("sub_accounts").AutoMigrate(&TestSubAccountModel{}); err != nil {
		return nil, err
	}

	return db, nil
}

// TestSubAccountRepository_Create tests sub-account creation functionality
func TestSubAccountRepository_Create(t *testing.T) {
	// Setup test config
	_ = config.GetProfileConfig(config.ProfileQuick)
	
	// Create test database
	db, err := setupSubAccountTestDB()
	require.NoError(t, err)
	
	// Create repositories
	subAccountRepo := repositories.NewSubAccountRepository(db)
	userRepo := repositories.NewUserRepository(db)
	exchangeRepo := repositories.NewExchangeRepository(db)
	userFactory := helpers.NewUserFactory()
	
	// Create test user first
	testUser := userFactory.WithEmail("subaccount@example.com")
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
		APIKey:    "test_key",
		APISecret: "test_secret",
		Status:    "active",
	}
	err = exchangeRepo.Create(context.Background(), testExchange)
	require.NoError(t, err)
	
	// Test successful sub-account creation
	t.Run("successful_subaccount_creation", func(t *testing.T) {
		testSubAccount := &models.SubAccount{
			ID:         uuid.New(),
			UserID:     testUser.ID,
			ExchangeID: testExchange.ID,
			Name:       "BTC Spot Account",
			Symbol:     "BTC",
			Balance:    1.5,
		}
		
		// Execute test
		err := subAccountRepo.Create(context.Background(), testSubAccount)
		
		// Verify results
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, testSubAccount.ID)
		
		// Verify sub-account was saved to database
		var savedSubAccount TestSubAccountModel
		err = db.Table("sub_accounts").Where("id = ?", testSubAccount.ID).First(&savedSubAccount).Error
		require.NoError(t, err)
		assert.Equal(t, testSubAccount.UserID.String(), savedSubAccount.UserID.String())
		assert.Equal(t, testSubAccount.ExchangeID.String(), savedSubAccount.ExchangeID.String())
		assert.Equal(t, testSubAccount.Name, savedSubAccount.Name)
		assert.Equal(t, testSubAccount.Symbol, savedSubAccount.Symbol)
		assert.Equal(t, testSubAccount.Balance, savedSubAccount.Balance)
	})
	
	// Test sub-account creation with different symbols
	t.Run("create_different_symbols", func(t *testing.T) {
		symbols := []struct {
			symbol  string
			name    string
			balance float64
		}{
			{"ETH", "Ethereum Account", 10.5},
			{"USDT", "USDT Holdings", 5000.0},
			{"BNB", "BNB Staking", 25.75},
			{"ADA", "Cardano Spot", 1000.0},
		}
		
		for _, s := range symbols {
			subAccount := &models.SubAccount{
				ID:         uuid.New(),
				UserID:     testUser.ID,
				ExchangeID: testExchange.ID,
				Name:       s.name,
				Symbol:     s.symbol,
				Balance:    s.balance,
			}
			
			err := subAccountRepo.Create(context.Background(), subAccount)
			require.NoError(t, err, "Failed to create %s sub-account", s.symbol)
		}
	})
	
	// Test creation with zero balance
	t.Run("create_with_zero_balance", func(t *testing.T) {
		zeroBalanceAccount := &models.SubAccount{
			ID:         uuid.New(),
			UserID:     testUser.ID,
			ExchangeID: testExchange.ID,
			Name:       "Empty DOT Account",
			Symbol:     "DOT",
			Balance:    0.0,
		}
		
		err := subAccountRepo.Create(context.Background(), zeroBalanceAccount)
		require.NoError(t, err)
		
		// Verify zero balance is preserved
		var savedAccount TestSubAccountModel
		err = db.Table("sub_accounts").Where("id = ?", zeroBalanceAccount.ID).First(&savedAccount).Error
		require.NoError(t, err)
		assert.Equal(t, 0.0, savedAccount.Balance)
	})
}

// TestSubAccountRepository_GetByID tests sub-account retrieval by ID
func TestSubAccountRepository_GetByID(t *testing.T) {
	// Create test database
	db, err := setupSubAccountTestDB()
	require.NoError(t, err)
	
	// Create repositories
	subAccountRepo := repositories.NewSubAccountRepository(db)
	userRepo := repositories.NewUserRepository(db)
	exchangeRepo := repositories.NewExchangeRepository(db)
	userFactory := helpers.NewUserFactory()
	
	// Create test user and exchange
	testUser := userFactory.WithEmail("getbyid@example.com")
	if testUser.ID == uuid.Nil {
		testUser.ID = uuid.New()
	}
	err = userRepo.Create(context.Background(), testUser)
	require.NoError(t, err)
	
	testExchange := &models.Exchange{
		ID:        uuid.New(),
		UserID:    testUser.ID,
		Name:      "Test Exchange",
		Type:      "binance",
		APIKey:    "test_key",
		APISecret: "test_secret",
		Status:    "active",
	}
	err = exchangeRepo.Create(context.Background(), testExchange)
	require.NoError(t, err)
	
	// Create test sub-account
	testSubAccount := &models.SubAccount{
		ID:         uuid.New(),
		UserID:     testUser.ID,
		ExchangeID: testExchange.ID,
		Name:       "ETH Trading Account",
		Symbol:     "ETH",
		Balance:    5.25,
	}
	err = subAccountRepo.Create(context.Background(), testSubAccount)
	require.NoError(t, err)
	
	// Test successful retrieval
	t.Run("successful_get_by_id", func(t *testing.T) {
		// Execute test
		retrievedSubAccount, err := subAccountRepo.GetByID(context.Background(), testSubAccount.ID)
		
		// Verify results
		require.NoError(t, err)
		require.NotNil(t, retrievedSubAccount)
		assert.Equal(t, testSubAccount.ID, retrievedSubAccount.ID)
		assert.Equal(t, testSubAccount.UserID, retrievedSubAccount.UserID)
		assert.Equal(t, testSubAccount.ExchangeID, retrievedSubAccount.ExchangeID)
		assert.Equal(t, testSubAccount.Name, retrievedSubAccount.Name)
		assert.Equal(t, testSubAccount.Symbol, retrievedSubAccount.Symbol)
		assert.Equal(t, testSubAccount.Balance, retrievedSubAccount.Balance)
	})
	
	// Test non-existent sub-account
	t.Run("get_non_existent_subaccount", func(t *testing.T) {
		nonExistentID := uuid.New()
		
		// Execute test
		retrievedSubAccount, err := subAccountRepo.GetByID(context.Background(), nonExistentID)
		
		// Should return nil, nil for not found
		require.NoError(t, err)
		assert.Nil(t, retrievedSubAccount)
	})
	
	// Test with nil UUID
	t.Run("get_with_nil_uuid", func(t *testing.T) {
		// Execute test
		retrievedSubAccount, err := subAccountRepo.GetByID(context.Background(), uuid.Nil)
		
		// Should return nil, nil for invalid UUID
		require.NoError(t, err)
		assert.Nil(t, retrievedSubAccount)
	})
}

// TestSubAccountRepository_GetByUserID tests sub-account retrieval by user ID
func TestSubAccountRepository_GetByUserID(t *testing.T) {
	// Create test database
	db, err := setupSubAccountTestDB()
	require.NoError(t, err)
	
	// Create repositories
	subAccountRepo := repositories.NewSubAccountRepository(db)
	userRepo := repositories.NewUserRepository(db)
	exchangeRepo := repositories.NewExchangeRepository(db)
	userFactory := helpers.NewUserFactory()
	
	// Create test user
	testUser := userFactory.WithEmail("getbyuser@example.com")
	if testUser.ID == uuid.Nil {
		testUser.ID = uuid.New()
	}
	err = userRepo.Create(context.Background(), testUser)
	require.NoError(t, err)
	
	// Create multiple test exchanges
	exchange1 := &models.Exchange{
		ID:        uuid.New(),
		UserID:    testUser.ID,
		Name:      "Binance",
		Type:      "binance",
		APIKey:    "binance_key",
		APISecret: "binance_secret",
		Status:    "active",
	}
	err = exchangeRepo.Create(context.Background(), exchange1)
	require.NoError(t, err)
	
	exchange2 := &models.Exchange{
		ID:        uuid.New(),
		UserID:    testUser.ID,
		Name:      "OKX",
		Type:      "okx",
		APIKey:    "okx_key",
		APISecret: "okx_secret",
		Status:    "active",
	}
	err = exchangeRepo.Create(context.Background(), exchange2)
	require.NoError(t, err)
	
	// Create multiple sub-accounts across exchanges
	subAccounts := []*models.SubAccount{
		{
			ID:         uuid.New(),
			UserID:     testUser.ID,
			ExchangeID: exchange1.ID,
			Name:       "BTC Binance",
			Symbol:     "BTC",
			Balance:    1.0,
		},
		{
			ID:         uuid.New(),
			UserID:     testUser.ID,
			ExchangeID: exchange1.ID,
			Name:       "ETH Binance",
			Symbol:     "ETH",
			Balance:    10.0,
		},
		{
			ID:         uuid.New(),
			UserID:     testUser.ID,
			ExchangeID: exchange2.ID,
			Name:       "BTC OKX",
			Symbol:     "BTC",
			Balance:    0.5,
		},
	}
	
	for i, subAccount := range subAccounts {
		err = subAccountRepo.Create(context.Background(), subAccount)
		require.NoError(t, err)
		
		// Add small delay to ensure different created_at times for ordering test
		if i < len(subAccounts)-1 {
			time.Sleep(1 * time.Millisecond)
		}
	}
	
	// Test get all sub-accounts for user
	t.Run("get_all_subaccounts_for_user", func(t *testing.T) {
		// Execute test
		retrievedSubAccounts, err := subAccountRepo.GetByUserID(context.Background(), testUser.ID, nil)
		
		// Verify results
		require.NoError(t, err)
		require.Len(t, retrievedSubAccounts, 3)
		
		// Verify ordering (should be DESC by created_at)
		for i := 0; i < len(retrievedSubAccounts)-1; i++ {
			assert.True(t, retrievedSubAccounts[i].CreatedAt.After(retrievedSubAccounts[i+1].CreatedAt) ||
				retrievedSubAccounts[i].CreatedAt.Equal(retrievedSubAccounts[i+1].CreatedAt))
		}
		
		// Verify all sub-accounts belong to the user
		for _, subAccount := range retrievedSubAccounts {
			assert.Equal(t, testUser.ID, subAccount.UserID)
		}
	})
	
	// Test get sub-accounts for user filtered by exchange
	t.Run("get_subaccounts_filtered_by_exchange", func(t *testing.T) {
		// Execute test - get only exchange1 sub-accounts
		retrievedSubAccounts, err := subAccountRepo.GetByUserID(context.Background(), testUser.ID, &exchange1.ID)
		
		// Verify results
		require.NoError(t, err)
		require.Len(t, retrievedSubAccounts, 2)
		
		// Verify all belong to exchange1
		for _, subAccount := range retrievedSubAccounts {
			assert.Equal(t, testUser.ID, subAccount.UserID)
			assert.Equal(t, exchange1.ID, subAccount.ExchangeID)
		}
	})
	
	// Test non-existent user
	t.Run("get_subaccounts_for_non_existent_user", func(t *testing.T) {
		nonExistentUserID := uuid.New()
		
		// Execute test
		retrievedSubAccounts, err := subAccountRepo.GetByUserID(context.Background(), nonExistentUserID, nil)
		
		// Should return empty slice, no error
		require.NoError(t, err)
		assert.Len(t, retrievedSubAccounts, 0)
	})
}

// TestSubAccountRepository_GetByExchangeID tests sub-account retrieval by exchange ID
func TestSubAccountRepository_GetByExchangeID(t *testing.T) {
	// Create test database
	db, err := setupSubAccountTestDB()
	require.NoError(t, err)
	
	// Create repositories and test data (similar setup)
	subAccountRepo := repositories.NewSubAccountRepository(db)
	userRepo := repositories.NewUserRepository(db)
	exchangeRepo := repositories.NewExchangeRepository(db)
	userFactory := helpers.NewUserFactory()
	
	// Create test user and exchange
	testUser := userFactory.WithEmail("getbyexchange@example.com")
	if testUser.ID == uuid.Nil {
		testUser.ID = uuid.New()
	}
	err = userRepo.Create(context.Background(), testUser)
	require.NoError(t, err)
	
	testExchange := &models.Exchange{
		ID:        uuid.New(),
		UserID:    testUser.ID,
		Name:      "Test Exchange",
		Type:      "binance",
		APIKey:    "test_key",
		APISecret: "test_secret",
		Status:    "active",
	}
	err = exchangeRepo.Create(context.Background(), testExchange)
	require.NoError(t, err)
	
	// Create multiple sub-accounts for the exchange
	subAccounts := []*models.SubAccount{
		{
			ID:         uuid.New(),
			UserID:     testUser.ID,
			ExchangeID: testExchange.ID,
			Name:       "BTC Account",
			Symbol:     "BTC",
			Balance:    1.0,
		},
		{
			ID:         uuid.New(),
			UserID:     testUser.ID,
			ExchangeID: testExchange.ID,
			Name:       "ETH Account",
			Symbol:     "ETH",
			Balance:    10.0,
		},
	}
	
	for _, subAccount := range subAccounts {
		err = subAccountRepo.Create(context.Background(), subAccount)
		require.NoError(t, err)
		time.Sleep(1 * time.Millisecond)
	}
	
	// Test successful retrieval
	t.Run("successful_get_by_exchange_id", func(t *testing.T) {
		// Execute test
		retrievedSubAccounts, err := subAccountRepo.GetByExchangeID(context.Background(), testExchange.ID)
		
		// Verify results
		require.NoError(t, err)
		require.Len(t, retrievedSubAccounts, 2)
		
		// Verify all belong to the exchange
		for _, subAccount := range retrievedSubAccounts {
			assert.Equal(t, testExchange.ID, subAccount.ExchangeID)
		}
		
		// Verify ordering
		assert.True(t, retrievedSubAccounts[0].CreatedAt.After(retrievedSubAccounts[1].CreatedAt) ||
			retrievedSubAccounts[0].CreatedAt.Equal(retrievedSubAccounts[1].CreatedAt))
	})
	
	// Test non-existent exchange
	t.Run("get_by_non_existent_exchange", func(t *testing.T) {
		nonExistentExchangeID := uuid.New()
		
		// Execute test
		retrievedSubAccounts, err := subAccountRepo.GetByExchangeID(context.Background(), nonExistentExchangeID)
		
		// Should return empty slice, no error
		require.NoError(t, err)
		assert.Len(t, retrievedSubAccounts, 0)
	})
}

// TestSubAccountRepository_GetBySymbol tests sub-account retrieval by symbol
func TestSubAccountRepository_GetBySymbol(t *testing.T) {
	// Create test database and setup similar to above
	db, err := setupSubAccountTestDB()
	require.NoError(t, err)
	
	subAccountRepo := repositories.NewSubAccountRepository(db)
	userRepo := repositories.NewUserRepository(db)
	exchangeRepo := repositories.NewExchangeRepository(db)
	userFactory := helpers.NewUserFactory()
	
	// Create test user and exchanges
	testUser := userFactory.WithEmail("getbysymbol@example.com")
	if testUser.ID == uuid.Nil {
		testUser.ID = uuid.New()
	}
	err = userRepo.Create(context.Background(), testUser)
	require.NoError(t, err)
	
	exchange1 := &models.Exchange{
		ID:        uuid.New(),
		UserID:    testUser.ID,
		Name:      "Binance",
		Type:      "binance",
		APIKey:    "binance_key",
		APISecret: "binance_secret",
		Status:    "active",
	}
	err = exchangeRepo.Create(context.Background(), exchange1)
	require.NoError(t, err)
	
	exchange2 := &models.Exchange{
		ID:        uuid.New(),
		UserID:    testUser.ID,
		Name:      "OKX",
		Type:      "okx",
		APIKey:    "okx_key",
		APISecret: "okx_secret",
		Status:    "active",
	}
	err = exchangeRepo.Create(context.Background(), exchange2)
	require.NoError(t, err)
	
	// Create sub-accounts with same symbol across different exchanges
	subAccounts := []*models.SubAccount{
		{
			ID:         uuid.New(),
			UserID:     testUser.ID,
			ExchangeID: exchange1.ID,
			Name:       "BTC Binance",
			Symbol:     "BTC",
			Balance:    1.0,
		},
		{
			ID:         uuid.New(),
			UserID:     testUser.ID,
			ExchangeID: exchange2.ID,
			Name:       "BTC OKX",
			Symbol:     "BTC",
			Balance:    0.5,
		},
		{
			ID:         uuid.New(),
			UserID:     testUser.ID,
			ExchangeID: exchange1.ID,
			Name:       "ETH Binance",
			Symbol:     "ETH",
			Balance:    10.0,
		},
	}
	
	for _, subAccount := range subAccounts {
		err = subAccountRepo.Create(context.Background(), subAccount)
		require.NoError(t, err)
		time.Sleep(1 * time.Millisecond)
	}
	
	// Test get by symbol
	t.Run("get_by_symbol_multiple_exchanges", func(t *testing.T) {
		// Execute test - get BTC accounts
		retrievedSubAccounts, err := subAccountRepo.GetBySymbol(context.Background(), testUser.ID, "BTC")
		
		// Verify results
		require.NoError(t, err)
		require.Len(t, retrievedSubAccounts, 2)
		
		// Verify all are BTC
		for _, subAccount := range retrievedSubAccounts {
			assert.Equal(t, "BTC", subAccount.Symbol)
			assert.Equal(t, testUser.ID, subAccount.UserID)
		}
	})
	
	// Test get by symbol single result
	t.Run("get_by_symbol_single_result", func(t *testing.T) {
		// Execute test - get ETH accounts
		retrievedSubAccounts, err := subAccountRepo.GetBySymbol(context.Background(), testUser.ID, "ETH")
		
		// Verify results
		require.NoError(t, err)
		require.Len(t, retrievedSubAccounts, 1)
		assert.Equal(t, "ETH", retrievedSubAccounts[0].Symbol)
	})
	
	// Test non-existent symbol
	t.Run("get_by_non_existent_symbol", func(t *testing.T) {
		// Execute test
		retrievedSubAccounts, err := subAccountRepo.GetBySymbol(context.Background(), testUser.ID, "XRP")
		
		// Should return empty slice, no error
		require.NoError(t, err)
		assert.Len(t, retrievedSubAccounts, 0)
	})
}

// TestSubAccountRepository_Update tests sub-account update functionality
func TestSubAccountRepository_Update(t *testing.T) {
	// Create test database and setup
	db, err := setupSubAccountTestDB()
	require.NoError(t, err)
	
	subAccountRepo := repositories.NewSubAccountRepository(db)
	userRepo := repositories.NewUserRepository(db)
	exchangeRepo := repositories.NewExchangeRepository(db)
	userFactory := helpers.NewUserFactory()
	
	// Create test user and exchange
	testUser := userFactory.WithEmail("update@example.com")
	if testUser.ID == uuid.Nil {
		testUser.ID = uuid.New()
	}
	err = userRepo.Create(context.Background(), testUser)
	require.NoError(t, err)
	
	testExchange := &models.Exchange{
		ID:        uuid.New(),
		UserID:    testUser.ID,
		Name:      "Test Exchange",
		Type:      "binance",
		APIKey:    "test_key",
		APISecret: "test_secret",
		Status:    "active",
	}
	err = exchangeRepo.Create(context.Background(), testExchange)
	require.NoError(t, err)
	
	// Create test sub-account
	testSubAccount := &models.SubAccount{
		ID:         uuid.New(),
		UserID:     testUser.ID,
		ExchangeID: testExchange.ID,
		Name:       "Original Account",
		Symbol:     "BTC",
		Balance:    1.0,
	}
	err = subAccountRepo.Create(context.Background(), testSubAccount)
	require.NoError(t, err)
	
	// Test successful update
	t.Run("successful_update", func(t *testing.T) {
		// Update sub-account data
		testSubAccount.Name = "Updated Account Name"
		testSubAccount.Balance = 2.5
		
		// Execute test
		err := subAccountRepo.Update(context.Background(), testSubAccount)
		
		// Verify results
		require.NoError(t, err)
		
		// Verify update was saved to database
		var updatedSubAccount TestSubAccountModel
		err = db.Table("sub_accounts").Where("id = ?", testSubAccount.ID).First(&updatedSubAccount).Error
		require.NoError(t, err)
		assert.Equal(t, "Updated Account Name", updatedSubAccount.Name)
		assert.Equal(t, 2.5, updatedSubAccount.Balance)
	})
}

// TestSubAccountRepository_UpdateBalance tests balance update functionality
func TestSubAccountRepository_UpdateBalance(t *testing.T) {
	// Create test database
	db, err := setupSubAccountTestDB()
	require.NoError(t, err)
	
	subAccountRepo := repositories.NewSubAccountRepository(db)
	userRepo := repositories.NewUserRepository(db)
	exchangeRepo := repositories.NewExchangeRepository(db)
	userFactory := helpers.NewUserFactory()
	
	// Create test user and exchange
	testUser := userFactory.WithEmail("updatebalance@example.com")
	if testUser.ID == uuid.Nil {
		testUser.ID = uuid.New()
	}
	err = userRepo.Create(context.Background(), testUser)
	require.NoError(t, err)
	
	testExchange := &models.Exchange{
		ID:        uuid.New(),
		UserID:    testUser.ID,
		Name:      "Test Exchange",
		Type:      "binance",
		APIKey:    "test_key",
		APISecret: "test_secret",
		Status:    "active",
	}
	err = exchangeRepo.Create(context.Background(), testExchange)
	require.NoError(t, err)
	
	// Create test sub-account
	testSubAccount := &models.SubAccount{
		ID:         uuid.New(),
		UserID:     testUser.ID,
		ExchangeID: testExchange.ID,
		Name:       "Balance Test Account",
		Symbol:     "BTC",
		Balance:    1.0,
	}
	err = subAccountRepo.Create(context.Background(), testSubAccount)
	require.NoError(t, err)
	
	// Test UpdateBalance - this will fail in SQLite since it calls a PostgreSQL function
	// But we test that it doesn't panic and handles the error gracefully
	t.Run("update_balance_database_function_call", func(t *testing.T) {
		// Execute test - this should fail gracefully since update_sub_account_balance doesn't exist in SQLite
		transactionID, err := subAccountRepo.UpdateBalance(
			context.Background(),
			testSubAccount.ID,
			2.0,     // new balance
			1.0,     // amount
			"credit", // direction
			"deposit", // reason
			map[string]interface{}{"source": "bank_transfer"},
		)
		
		// In SQLite, this will fail because the database function doesn't exist
		// But the repository should handle this gracefully
		if err != nil {
			// Expected in test environment with SQLite
			// Error could be "no such function" or "unrecognized token" depending on SQLite version
			assert.True(t, 
				strings.Contains(err.Error(), "no such function") || 
				strings.Contains(err.Error(), "unrecognized token"),
				"Expected database function error, got: %s", err.Error())
			assert.Nil(t, transactionID)
		} else {
			// If it somehow succeeds, verify we get a transaction ID
			assert.NotNil(t, transactionID)
			assert.NotEqual(t, uuid.Nil, *transactionID)
		}
	})
	
	// Test UpdateBalance with nil info
	t.Run("update_balance_with_nil_info", func(t *testing.T) {
		// Execute test
		transactionID, err := subAccountRepo.UpdateBalance(
			context.Background(),
			testSubAccount.ID,
			1.5,
			0.5,
			"credit",
			"interest",
			nil, // nil info
		)
		
		// Should fail in SQLite but handle nil info properly
		if err != nil {
			assert.True(t, 
				strings.Contains(err.Error(), "no such function") || 
				strings.Contains(err.Error(), "unrecognized token"),
				"Expected database function error, got: %s", err.Error())
			assert.Nil(t, transactionID)
		}
	})
	
	// Test UpdateBalance with string info
	t.Run("update_balance_with_string_info", func(t *testing.T) {
		// Execute test
		transactionID, err := subAccountRepo.UpdateBalance(
			context.Background(),
			testSubAccount.ID,
			1.2,
			0.2,
			"debit",
			"withdrawal",
			`{"destination": "wallet"}`, // string info
		)
		
		// Should fail in SQLite but handle string info properly
		if err != nil {
			assert.True(t, 
				strings.Contains(err.Error(), "no such function") || 
				strings.Contains(err.Error(), "unrecognized token"),
				"Expected database function error, got: %s", err.Error())
			assert.Nil(t, transactionID)
		}
	})
}

// TestSubAccountRepository_Delete tests sub-account deletion functionality
func TestSubAccountRepository_Delete(t *testing.T) {
	// Create test database
	db, err := setupSubAccountTestDB()
	require.NoError(t, err)
	
	subAccountRepo := repositories.NewSubAccountRepository(db)
	userRepo := repositories.NewUserRepository(db)
	exchangeRepo := repositories.NewExchangeRepository(db)
	userFactory := helpers.NewUserFactory()
	
	// Create test user and exchange
	testUser := userFactory.WithEmail("delete@example.com")
	if testUser.ID == uuid.Nil {
		testUser.ID = uuid.New()
	}
	err = userRepo.Create(context.Background(), testUser)
	require.NoError(t, err)
	
	testExchange := &models.Exchange{
		ID:        uuid.New(),
		UserID:    testUser.ID,
		Name:      "Test Exchange",
		Type:      "binance",
		APIKey:    "test_key",
		APISecret: "test_secret",
		Status:    "active",
	}
	err = exchangeRepo.Create(context.Background(), testExchange)
	require.NoError(t, err)
	
	// Test successful deletion (zero balance)
	t.Run("successful_delete_zero_balance", func(t *testing.T) {
		// Create sub-account with zero balance
		zeroBalanceAccount := &models.SubAccount{
			ID:         uuid.New(),
			UserID:     testUser.ID,
			ExchangeID: testExchange.ID,
			Name:       "Deletable Account",
			Symbol:     "XRP",
			Balance:    0.0,
		}
		err = subAccountRepo.Create(context.Background(), zeroBalanceAccount)
		require.NoError(t, err)
		
		// Execute test
		err = subAccountRepo.Delete(context.Background(), zeroBalanceAccount.ID)
		
		// Verify results
		require.NoError(t, err)
		
		// Verify sub-account was soft deleted
		var deletedSubAccount TestSubAccountModel
		err = db.Table("sub_accounts").Unscoped().Where("id = ?", zeroBalanceAccount.ID).First(&deletedSubAccount).Error
		require.NoError(t, err)
		assert.NotNil(t, deletedSubAccount.DeletedAt) // Should be soft deleted
	})
	
	// Test deletion with non-zero balance
	t.Run("delete_with_non_zero_balance", func(t *testing.T) {
		// Create sub-account with non-zero balance
		nonZeroBalanceAccount := &models.SubAccount{
			ID:         uuid.New(),
			UserID:     testUser.ID,
			ExchangeID: testExchange.ID,
			Name:       "Non-deletable Account",
			Symbol:     "ETH",
			Balance:    5.0,
		}
		err = subAccountRepo.Create(context.Background(), nonZeroBalanceAccount)
		require.NoError(t, err)
		
		// Execute test - should fail
		err = subAccountRepo.Delete(context.Background(), nonZeroBalanceAccount.ID)
		
		// Verify error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete sub-account with non-zero balance")
		
		// Verify sub-account was not deleted
		var existingSubAccount TestSubAccountModel
		err = db.Table("sub_accounts").Where("id = ?", nonZeroBalanceAccount.ID).First(&existingSubAccount).Error
		require.NoError(t, err)
		assert.Nil(t, existingSubAccount.DeletedAt) // Should not be deleted
	})
	
	// Test delete non-existent sub-account
	t.Run("delete_non_existent_subaccount", func(t *testing.T) {
		nonExistentID := uuid.New()
		
		// Execute test
		err := subAccountRepo.Delete(context.Background(), nonExistentID)
		
		// Should return error since we can't find the sub-account to check balance
		require.Error(t, err)
	})
}

// Performance test for sub-account repository operations
func TestSubAccountRepository_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}
	
	// Create test database
	db, err := setupSubAccountTestDB()
	require.NoError(t, err)
	
	subAccountRepo := repositories.NewSubAccountRepository(db)
	userRepo := repositories.NewUserRepository(db)
	exchangeRepo := repositories.NewExchangeRepository(db)
	userFactory := helpers.NewUserFactory()
	
	// Create test user and exchange
	testUser := userFactory.WithEmail("performance@example.com")
	if testUser.ID == uuid.Nil {
		testUser.ID = uuid.New()
	}
	err = userRepo.Create(context.Background(), testUser)
	require.NoError(t, err)
	
	testExchange := &models.Exchange{
		ID:        uuid.New(),
		UserID:    testUser.ID,
		Name:      "Performance Exchange",
		Type:      "binance",
		APIKey:    "perf_key",
		APISecret: "perf_secret",
		Status:    "active",
	}
	err = exchangeRepo.Create(context.Background(), testExchange)
	require.NoError(t, err)
	
	t.Run("bulk_subaccount_create_performance", func(t *testing.T) {
		timer := helpers.NewPerformanceTimer()
		timer.Start()
		
		// Create 50 sub-accounts
		for i := 0; i < 50; i++ {
			subAccount := &models.SubAccount{
				ID:         uuid.New(),
				UserID:     testUser.ID,
				ExchangeID: testExchange.ID,
				Name:       "Performance Account " + string(rune(i+48)),
				Symbol:     "PERF" + string(rune(i+48)),
				Balance:    float64(i),
			}
			err := subAccountRepo.Create(context.Background(), subAccount)
			require.NoError(t, err)
		}
		
		timer.Stop()
		
		// Check performance within reasonable bounds
		assert.Less(t, timer.Duration().Milliseconds(), int64(2000),
			"50 sub-account creations should complete within 2 seconds")
	})
	
	t.Run("subaccount_lookup_performance", func(t *testing.T) {
		// Create a test sub-account first
		testSubAccount := &models.SubAccount{
			ID:         uuid.New(),
			UserID:     testUser.ID,
			ExchangeID: testExchange.ID,
			Name:       "Lookup Performance Account",
			Symbol:     "LOOKUP",
			Balance:    1.0,
		}
		err := subAccountRepo.Create(context.Background(), testSubAccount)
		require.NoError(t, err)
		
		timer := helpers.NewPerformanceTimer()
		timer.Start()
		
		// Perform 500 lookups
		for i := 0; i < 500; i++ {
			_, err := subAccountRepo.GetByID(context.Background(), testSubAccount.ID)
			require.NoError(t, err)
		}
		
		timer.Stop()
		
		// Check performance within reasonable bounds
		assert.Less(t, timer.Duration().Milliseconds(), int64(1000),
			"500 sub-account lookups should complete within 1 second")
	})
}