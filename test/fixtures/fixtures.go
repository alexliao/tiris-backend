package fixtures

import (
	"time"

	"tiris-backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// UserFixtures provides test user data
var UserFixtures = struct {
	ValidUser        *models.User
	AdminUser        *models.User
	InactiveUser     *models.User
	UserWithoutLogin *models.User
}{
	ValidUser: &models.User{
		ID:       uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
		Username: "testuser",
		Email:    "testuser@example.com",
		Avatar:   stringPtr("https://example.com/avatar.png"),
		Settings: datatypes.JSON(`{"theme": "dark"}`),
		Info:     datatypes.JSON(`{"last_login": "2023-01-01T00:00:00Z"}`),
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	},
	AdminUser: &models.User{
		ID:       uuid.MustParse("123e4567-e89b-12d3-a456-426614174001"),
		Username: "admin",
		Email:    "admin@example.com",
		Avatar:   stringPtr("https://example.com/admin.png"),
		Settings: datatypes.JSON(`{"role": "admin", "theme": "light"}`),
		Info:     datatypes.JSON(`{"permissions": ["read", "write", "admin"]}`),
		CreatedAt: time.Now().Add(-48 * time.Hour),
		UpdatedAt: time.Now().Add(-30 * time.Minute),
	},
	InactiveUser: &models.User{
		ID:       uuid.MustParse("123e4567-e89b-12d3-a456-426614174002"),
		Username: "inactive",
		Email:    "inactive@example.com",
		Avatar:   nil,
		Settings: datatypes.JSON(`{}`),
		Info:     datatypes.JSON(`{"status": "inactive"}`),
		CreatedAt: time.Now().Add(-96 * time.Hour),
		UpdatedAt: time.Now().Add(-72 * time.Hour),
	},
	UserWithoutLogin: &models.User{
		ID:       uuid.MustParse("123e4567-e89b-12d3-a456-426614174003"),
		Username: "newuser",
		Email:    "newuser@example.com",
		Avatar:   nil,
		Settings: datatypes.JSON(`{"theme": "dark"}`),
		Info:     datatypes.JSON(`{}`),
		CreatedAt: time.Now().Add(-1 * time.Hour),
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	},
}

// ExchangeFixtures provides test exchange data
var ExchangeFixtures = struct {
	BinanceExchange *models.Exchange
	OKXExchange     *models.Exchange
	TestExchange    *models.Exchange
}{
	BinanceExchange: &models.Exchange{
		ID:        uuid.MustParse("223e4567-e89b-12d3-a456-426614174000"),
		UserID:    UserFixtures.ValidUser.ID,
		Name:      "binance",
		Type:      "spot",
		APIKey:    "test_binance_api_key",
		APISecret: "test_binance_secret",
		Status:    "active",
		Info:      datatypes.JSON(`{"sandbox": true}`),
		CreatedAt: time.Now().Add(-12 * time.Hour),
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	},
	OKXExchange: &models.Exchange{
		ID:        uuid.MustParse("223e4567-e89b-12d3-a456-426614174001"),
		UserID:    UserFixtures.ValidUser.ID,
		Name:      "okx",
		Type:      "spot",
		APIKey:    "test_okx_api_key",
		APISecret: "test_okx_secret",
		Status:    "active",
		Info:      datatypes.JSON(`{"sandbox": true, "passphrase": "test_passphrase"}`),
		CreatedAt: time.Now().Add(-6 * time.Hour),
		UpdatedAt: time.Now().Add(-30 * time.Minute),
	},
	TestExchange: &models.Exchange{
		ID:        uuid.MustParse("223e4567-e89b-12d3-a456-426614174002"),
		UserID:    UserFixtures.AdminUser.ID,
		Name:      "test_exchange",
		Type:      "futures",
		APIKey:    "test_api_key",
		APISecret: "test_secret_key",
		Status:    "inactive",
		Info:      datatypes.JSON(`{"sandbox": false}`),
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
		ExchangeID: ExchangeFixtures.BinanceExchange.ID,
		Name:       "spot",
		Symbol:     "USDT",
		Balance:    1000.0,
		Info:       datatypes.JSON(`{"type": "spot"}`),
		CreatedAt:  time.Now().Add(-6 * time.Hour),
		UpdatedAt:  time.Now().Add(-30 * time.Minute),
	},
	FuturesAccount: &models.SubAccount{
		ID:         uuid.MustParse("323e4567-e89b-12d3-a456-426614174001"),
		UserID:     UserFixtures.ValidUser.ID,
		ExchangeID: ExchangeFixtures.BinanceExchange.ID,
		Name:       "futures",
		Symbol:     "USDT",
		Balance:    5000.0,
		Info:       datatypes.JSON(`{"type": "futures"}`),
		CreatedAt:  time.Now().Add(-4 * time.Hour),
		UpdatedAt:  time.Now().Add(-15 * time.Minute),
	},
	MarginAccount: &models.SubAccount{
		ID:         uuid.MustParse("323e4567-e89b-12d3-a456-426614174002"),
		UserID:     UserFixtures.ValidUser.ID,
		ExchangeID: ExchangeFixtures.OKXExchange.ID,
		Name:       "margin",
		Symbol:     "USDT",
		Balance:    2000.0,
		Info:       datatypes.JSON(`{"type": "margin"}`),
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
		ExchangeID:     ExchangeFixtures.BinanceExchange.ID,
		SubAccountID:   SubAccountFixtures.SpotAccount.ID,
		Timestamp:      time.Now().Add(-2 * time.Hour),
		Direction:      "credit",
		Reason:         "deposit",
		Amount:         1000.0,
		ClosingBalance: 1000.0,
		Info:           datatypes.JSON(`{"txid": "0x123456789abcdef"}`),
	},
	WithdrawTransaction: &models.Transaction{
		ID:             uuid.MustParse("423e4567-e89b-12d3-a456-426614174001"),
		UserID:         UserFixtures.ValidUser.ID,
		ExchangeID:     ExchangeFixtures.BinanceExchange.ID,
		SubAccountID:   SubAccountFixtures.SpotAccount.ID,
		Timestamp:      time.Now().Add(-1 * time.Hour),
		Direction:      "debit",
		Reason:         "withdrawal",
		Amount:         100.0,
		ClosingBalance: 900.0,
		Info:           datatypes.JSON(`{"address": "0xabcdef123456789"}`),
	},
	TradeTransaction: &models.Transaction{
		ID:             uuid.MustParse("423e4567-e89b-12d3-a456-426614174002"),
		UserID:         UserFixtures.ValidUser.ID,
		ExchangeID:     ExchangeFixtures.BinanceExchange.ID,
		SubAccountID:   SubAccountFixtures.FuturesAccount.ID,
		Timestamp:      time.Now().Add(-30 * time.Minute),
		Direction:      "credit",
		Reason:         "trade_profit",
		Amount:         50.0,
		ClosingBalance: 5050.0,
		Info:           datatypes.JSON(`{"trade_id": "12345", "symbol": "BTCUSDT"}`),
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
		ExchangeID:    ExchangeFixtures.BinanceExchange.ID,
		SubAccountID:  &SubAccountFixtures.SpotAccount.ID,
		TransactionID: &TransactionFixtures.TradeTransaction.ID,
		Timestamp:     time.Now().Add(-1 * time.Hour),
		Type:          "trade",
		Source:        "manual",
		Message:       "Manual trade executed: Buy 0.1 BTC at $45000",
		Info:          datatypes.JSON(`{"symbol": "BTCUSDT", "side": "buy", "amount": 0.1, "price": 45000.0}`),
	},
	BotLog: &models.TradingLog{
		ID:           uuid.MustParse("523e4567-e89b-12d3-a456-426614174001"),
		UserID:       UserFixtures.ValidUser.ID,
		ExchangeID:   ExchangeFixtures.BinanceExchange.ID,
		SubAccountID: &SubAccountFixtures.FuturesAccount.ID,
		Timestamp:    time.Now().Add(-30 * time.Minute),
		Type:         "strategy",
		Source:       "bot",
		Message:      "Grid strategy triggered: Position opened",
		Info:         datatypes.JSON(`{"strategy": "grid", "symbol": "ETHUSDT", "action": "open_position"}`),
	},
	ErrorLog: &models.TradingLog{
		ID:           uuid.MustParse("523e4567-e89b-12d3-a456-426614174002"),
		UserID:       UserFixtures.ValidUser.ID,
		ExchangeID:   ExchangeFixtures.OKXExchange.ID,
		SubAccountID: nil,
		Timestamp:    time.Now().Add(-15 * time.Minute),
		Type:         "error",
		Source:       "bot",
		Message:      "API rate limit exceeded",
		Info:         datatypes.JSON(`{"error_code": "429", "endpoint": "/api/v5/trade/order"}`),
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
		Info:           datatypes.JSON(`{"scope": "email profile"}`),
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
		Info:           datatypes.JSON(`{"openid": "wechat_openid"}`),
		CreatedAt:      time.Now().Add(-6 * time.Hour),
		UpdatedAt:      time.Now().Add(-30 * time.Minute),
	},
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func timePtr(t time.Time) *time.Time {
	return &t
}