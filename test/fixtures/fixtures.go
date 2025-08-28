package fixtures

import (
	"time"

	"tiris-backend/internal/models"

	"github.com/google/uuid"
)

// FixedTime is a consistent time for tests
var FixedTime = time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

// UserFixtures provides test user data
var UserFixtures = struct {
	ValidUser        *models.User
	AdminUser        *models.User
	InactiveUser     *models.User
	UserWithoutLogin *models.User
}{
	ValidUser: &models.User{
		ID:        uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
		Username:  "testuser",
		Email:     "testuser@example.com",
		Avatar:    stringPtr("https://example.com/avatar.png"),
		Settings:  models.JSON{"theme": "dark"},
		Info:      models.JSON{"last_login": "2023-01-01T00:00:00Z"},
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	},
	AdminUser: &models.User{
		ID:        uuid.MustParse("123e4567-e89b-12d3-a456-426614174001"),
		Username:  "admin",
		Email:     "admin@example.com",
		Avatar:    stringPtr("https://example.com/admin.png"),
		Settings:  models.JSON{"role": "admin", "theme": "light"},
		Info:      models.JSON{"permissions": []string{"read", "write", "admin"}},
		CreatedAt: time.Now().Add(-48 * time.Hour),
		UpdatedAt: time.Now().Add(-30 * time.Minute),
	},
	InactiveUser: &models.User{
		ID:        uuid.MustParse("123e4567-e89b-12d3-a456-426614174002"),
		Username:  "inactive",
		Email:     "inactive@example.com",
		Avatar:    nil,
		Settings:  models.JSON{},
		Info:      models.JSON{"status": "inactive"},
		CreatedAt: time.Now().Add(-96 * time.Hour),
		UpdatedAt: time.Now().Add(-72 * time.Hour),
	},
	UserWithoutLogin: &models.User{
		ID:        uuid.MustParse("123e4567-e89b-12d3-a456-426614174003"),
		Username:  "newuser",
		Email:     "newuser@example.com",
		Avatar:    nil,
		Settings:  models.JSON{"theme": "dark"},
		Info:      models.JSON{},
		CreatedAt: time.Now().Add(-1 * time.Hour),
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	},
}

// TradingFixtures provides test trading data
var TradingFixtures = struct {
	BinanceTrading *models.Trading
	OKXTrading     *models.Trading
	TestTrading    *models.Trading
}{
	BinanceTrading: &models.Trading{
		ID:        uuid.MustParse("223e4567-e89b-12d3-a456-426614174000"),
		UserID:    UserFixtures.ValidUser.ID,
		Name:      "binance",
		Type:      "spot",
		APIKey:    "test_binance_api_key",
		APISecret: "test_binance_secret",
		Status:    "active",
		Info:      models.JSON{"sandbox": true},
		CreatedAt: time.Now().Add(-12 * time.Hour),
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	},
	OKXTrading: &models.Trading{
		ID:        uuid.MustParse("223e4567-e89b-12d3-a456-426614174001"),
		UserID:    UserFixtures.ValidUser.ID,
		Name:      "okx",
		Type:      "spot",
		APIKey:    "test_okx_api_key",
		APISecret: "test_okx_secret",
		Status:    "active",
		Info:      models.JSON{"sandbox": true, "passphrase": "test_passphrase"},
		CreatedAt: time.Now().Add(-6 * time.Hour),
		UpdatedAt: time.Now().Add(-30 * time.Minute),
	},
	TestTrading: &models.Trading{
		ID:        uuid.MustParse("223e4567-e89b-12d3-a456-426614174002"),
		UserID:    UserFixtures.AdminUser.ID,
		Name:      "test_trading",
		Type:      "futures",
		APIKey:    "test_api_key",
		APISecret: "test_secret_key",
		Status:    "inactive",
		Info:      models.JSON{"sandbox": false},
		CreatedAt: time.Now().Add(-3 * time.Hour),
		UpdatedAt: time.Now().Add(-15 * time.Minute),
	},
}

// SubAccountFixtures provides test sub-account data
var SubAccountFixtures = struct {
	SpotAccount    *models.SubAccount
	FuturesAccount *models.SubAccount
	MarginAccount  *models.SubAccount
}{
	SpotAccount: &models.SubAccount{
		ID:         uuid.MustParse("323e4567-e89b-12d3-a456-426614174000"),
		UserID:     UserFixtures.ValidUser.ID,
		TradingID: TradingFixtures.BinanceTrading.ID,
		Name:       "spot",
		Symbol:     "USDT",
		Balance:    1000.0,
		Info:       models.JSON{"type": "spot"},
		CreatedAt:  time.Now().Add(-6 * time.Hour),
		UpdatedAt:  time.Now().Add(-30 * time.Minute),
	},
	FuturesAccount: &models.SubAccount{
		ID:         uuid.MustParse("323e4567-e89b-12d3-a456-426614174001"),
		UserID:     UserFixtures.ValidUser.ID,
		TradingID: TradingFixtures.BinanceTrading.ID,
		Name:       "futures",
		Symbol:     "USDT",
		Balance:    5000.0,
		Info:       models.JSON{"type": "futures"},
		CreatedAt:  time.Now().Add(-4 * time.Hour),
		UpdatedAt:  time.Now().Add(-15 * time.Minute),
	},
	MarginAccount: &models.SubAccount{
		ID:         uuid.MustParse("323e4567-e89b-12d3-a456-426614174002"),
		UserID:     UserFixtures.ValidUser.ID,
		TradingID: TradingFixtures.OKXTrading.ID,
		Name:       "margin",
		Symbol:     "USDT",
		Balance:    2000.0,
		Info:       models.JSON{"type": "margin"},
		CreatedAt:  time.Now().Add(-2 * time.Hour),
		UpdatedAt:  time.Now().Add(-5 * time.Minute),
	},
}

// TransactionFixtures provides test transaction data
var TransactionFixtures = struct {
	DepositTransaction  *models.Transaction
	WithdrawTransaction *models.Transaction
	TradeTransaction    *models.Transaction
}{
	DepositTransaction: &models.Transaction{
		ID:             uuid.MustParse("423e4567-e89b-12d3-a456-426614174000"),
		UserID:         UserFixtures.ValidUser.ID,
		TradingID:     TradingFixtures.BinanceTrading.ID,
		SubAccountID:   SubAccountFixtures.SpotAccount.ID,
		Timestamp:      time.Now().Add(-2 * time.Hour),
		Direction:      "credit",
		Reason:         "deposit",
		Amount:         1000.0,
		ClosingBalance: 1000.0,
		Info:           models.JSON{"txid": "0x123456789abcdef"},
	},
	WithdrawTransaction: &models.Transaction{
		ID:             uuid.MustParse("423e4567-e89b-12d3-a456-426614174001"),
		UserID:         UserFixtures.ValidUser.ID,
		TradingID:     TradingFixtures.BinanceTrading.ID,
		SubAccountID:   SubAccountFixtures.SpotAccount.ID,
		Timestamp:      time.Now().Add(-1 * time.Hour),
		Direction:      "debit",
		Reason:         "withdrawal",
		Amount:         100.0,
		ClosingBalance: 900.0,
		Info:           models.JSON{"address": "0xabcdef123456789"},
	},
	TradeTransaction: &models.Transaction{
		ID:             uuid.MustParse("423e4567-e89b-12d3-a456-426614174002"),
		UserID:         UserFixtures.ValidUser.ID,
		TradingID:     TradingFixtures.BinanceTrading.ID,
		SubAccountID:   SubAccountFixtures.FuturesAccount.ID,
		Timestamp:      time.Now().Add(-30 * time.Minute),
		Direction:      "credit",
		Reason:         "trade_profit",
		Amount:         50.0,
		ClosingBalance: 5050.0,
		Info:           models.JSON{"trade_id": "12345", "symbol": "BTCUSDT"},
	},
}

// TradingLogFixtures provides test trading log data
var TradingLogFixtures = struct {
	ManualLog *models.TradingLog
	BotLog    *models.TradingLog
	ErrorLog  *models.TradingLog
}{
	ManualLog: &models.TradingLog{
		ID:            uuid.MustParse("523e4567-e89b-12d3-a456-426614174000"),
		UserID:        UserFixtures.ValidUser.ID,
		TradingID:    TradingFixtures.BinanceTrading.ID,
		SubAccountID:  &SubAccountFixtures.SpotAccount.ID,
		TransactionID: &TransactionFixtures.TradeTransaction.ID,
		Timestamp:     time.Now().Add(-1 * time.Hour),
		Type:          "trade",
		Source:        "manual",
		Message:       "Manual trade executed: Buy 0.1 BTC at $45000",
		Info:          models.JSON{"symbol": "BTCUSDT", "side": "buy", "amount": 0.1, "price": 45000.0},
	},
	BotLog: &models.TradingLog{
		ID:           uuid.MustParse("523e4567-e89b-12d3-a456-426614174001"),
		UserID:       UserFixtures.ValidUser.ID,
		TradingID:   TradingFixtures.BinanceTrading.ID,
		SubAccountID: &SubAccountFixtures.FuturesAccount.ID,
		Timestamp:    time.Now().Add(-30 * time.Minute),
		Type:         "strategy",
		Source:       "bot",
		Message:      "Grid strategy triggered: Position opened",
		Info:         models.JSON{"strategy": "grid", "symbol": "ETHUSDT", "action": "open_position"},
	},
	ErrorLog: &models.TradingLog{
		ID:           uuid.MustParse("523e4567-e89b-12d3-a456-426614174002"),
		UserID:       UserFixtures.ValidUser.ID,
		TradingID:   TradingFixtures.OKXTrading.ID,
		SubAccountID: nil,
		Timestamp:    time.Now().Add(-15 * time.Minute),
		Type:         "error",
		Source:       "bot",
		Message:      "API rate limit exceeded",
		Info:         models.JSON{"error_code": "429", "endpoint": "/api/v5/trade/order"},
	},
}

// OAuthTokenFixtures provides test OAuth token data
var OAuthTokenFixtures = struct {
	GoogleToken *models.OAuthToken
	WeChatToken *models.OAuthToken
}{
	GoogleToken: &models.OAuthToken{
		ID:             uuid.MustParse("623e4567-e89b-12d3-a456-426614174000"),
		UserID:         UserFixtures.ValidUser.ID,
		Provider:       "google",
		ProviderUserID: "google_user_123",
		AccessToken:    "google_access_token",
		RefreshToken:   stringPtr("google_refresh_token"),
		ExpiresAt:      timePtr(time.Now().Add(1 * time.Hour)),
		Info:           models.JSON{"scope": "email profile"},
		CreatedAt:      time.Now().Add(-12 * time.Hour),
		UpdatedAt:      time.Now().Add(-1 * time.Hour),
	},
	WeChatToken: &models.OAuthToken{
		ID:             uuid.MustParse("623e4567-e89b-12d3-a456-426614174001"),
		UserID:         UserFixtures.AdminUser.ID,
		Provider:       "wechat",
		ProviderUserID: "wechat_user_456",
		AccessToken:    "wechat_access_token",
		RefreshToken:   nil,
		ExpiresAt:      timePtr(time.Now().Add(2 * time.Hour)),
		Info:           models.JSON{"openid": "wechat_openid"},
		CreatedAt:      time.Now().Add(-6 * time.Hour),
		UpdatedAt:      time.Now().Add(-30 * time.Minute),
	},
}

// CreateUser creates a new user with randomized values for testing
func CreateUser() *models.User {
	return &models.User{
		ID:        uuid.New(),
		Username:  "testuser_" + uuid.New().String()[:8],
		Email:     "test_" + uuid.New().String()[:8] + "@example.com",
		Avatar:    nil,
		Settings:  models.JSON{"theme": "dark"},
		Info:      models.JSON{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// CreateTrading creates a new trading with randomized values for testing
func CreateTrading(userID uuid.UUID) *models.Trading {
	return &models.Trading{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      "test_trading_" + uuid.New().String()[:8],
		Type:      "spot",
		APIKey:    "test_api_key",
		APISecret: "test_api_secret",
		Status:    "active",
		Info:      models.JSON{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// CreateSubAccount creates a new sub-account with randomized values for testing
func CreateSubAccount(userID, tradingID uuid.UUID) *models.SubAccount {
	return &models.SubAccount{
		ID:         uuid.New(),
		UserID:     userID,
		TradingID: tradingID,
		Name:       "test_subaccount_" + uuid.New().String()[:8],
		Symbol:     "USDT",
		Balance:    1000.0,
		Info:       models.JSON{},
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func timePtr(t time.Time) *time.Time {
	return &t
}