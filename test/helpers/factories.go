package helpers

import (
	"fmt"
	"math/rand"
	"time"

	"tiris-backend/internal/models"

	"github.com/google/uuid"
)

// TestDataFactory provides methods to create test data dynamically
type TestDataFactory struct {
	counter int64
}

// NewTestDataFactory creates a new test data factory
func NewTestDataFactory() *TestDataFactory {
	return &TestDataFactory{
		counter: time.Now().UnixNano(),
	}
}

// nextID generates a unique identifier for tests
func (f *TestDataFactory) nextID() int64 {
	f.counter++
	return f.counter
}

// UserFactory creates User test data
type UserFactory struct {
	factory *TestDataFactory
}

// NewUserFactory creates a new user factory
func NewUserFactory() *UserFactory {
	return &UserFactory{
		factory: NewTestDataFactory(),
	}
}

// Build creates a basic user with default values
func (f *UserFactory) Build() *models.User {
	id := f.factory.nextID()
	return &models.User{
		ID:        uuid.New(),
		Username:  fmt.Sprintf("testuser_%d", id),
		Email:     fmt.Sprintf("testuser_%d@example.com", id),
		Avatar:    nil,
		Settings:  models.JSON(map[string]interface{}{"theme": "dark"}),
		Info:      models.JSON(map[string]interface{}{}),
		CreatedAt: time.Now().Add(-time.Duration(rand.Intn(72)) * time.Hour),
		UpdatedAt: time.Now().Add(-time.Duration(rand.Intn(24)) * time.Hour),
	}
}

// WithID sets a specific ID
func (f *UserFactory) WithID(id uuid.UUID) *models.User {
	user := f.Build()
	user.ID = id
	return user
}

// WithEmail sets a specific email
func (f *UserFactory) WithEmail(email string) *models.User {
	user := f.Build()
	user.Email = email
	return user
}

// WithUsername sets a specific username
func (f *UserFactory) WithUsername(username string) *models.User {
	user := f.Build()
	user.Username = username
	return user
}

// WithAvatar sets an avatar URL
func (f *UserFactory) WithAvatar(avatarURL string) *models.User {
	user := f.Build()
	user.Avatar = &avatarURL
	return user
}

// WithSettings sets user settings
func (f *UserFactory) WithSettings(settings map[string]interface{}) *models.User {
	user := f.Build()
	user.Settings = models.JSON(settings)
	return user
}

// WithInfo sets user info
func (f *UserFactory) WithInfo(info map[string]interface{}) *models.User {
	user := f.Build()
	user.Info = models.JSON(info)
	return user
}

// AdminUser creates a user with admin settings
func (f *UserFactory) AdminUser() *models.User {
	user := f.Build()
	user.Username = fmt.Sprintf("admin_%d", f.factory.nextID())
	user.Email = fmt.Sprintf("admin_%d@example.com", f.factory.counter)
	adminSettings := map[string]interface{}{
		"role":  "admin",
		"theme": "light",
	}
	user.Settings = models.JSON(adminSettings)
	adminInfo := map[string]interface{}{
		"permissions": []string{"read", "write", "admin"},
	}
	user.Info = models.JSON(adminInfo)
	return user
}

// TradingFactory creates Trading test data
type TradingFactory struct {
	factory *TestDataFactory
}

// NewTradingFactory creates a new trading factory
func NewTradingFactory() *TradingFactory {
	return &TradingFactory{
		factory: NewTestDataFactory(),
	}
}

// Build creates a basic exchange with default values
func (f *TradingFactory) Build() *models.Trading {
	id := f.factory.nextID()
	return &models.Trading{
		ID:        uuid.New(),
		UserID:    uuid.New(), // Will be overridden in WithUserID
		Name:      fmt.Sprintf("trading_%d", id),
		Type:      "spot",
		APIKey:    fmt.Sprintf("test_api_key_%d", id),
		APISecret: fmt.Sprintf("test_api_secret_%d", id),
		Status:    "active",
		Info:      models.JSON(map[string]interface{}{"sandbox": true}),
		CreatedAt: time.Now().Add(-time.Duration(rand.Intn(48)) * time.Hour),
		UpdatedAt: time.Now().Add(-time.Duration(rand.Intn(12)) * time.Hour),
	}
}

// WithUserID sets the user ID
func (f *TradingFactory) WithUserID(userID uuid.UUID) *models.Trading {
	trading := f.Build()
	trading.UserID = userID
	return trading
}

// WithName sets the exchange name
func (f *TradingFactory) WithName(name string) *models.Trading {
	trading := f.Build()
	trading.Name = name
	return trading
}

// WithType sets the trading type
func (f *TradingFactory) WithType(tradingType string) *models.Trading {
	trading := f.Build()
	trading.Type = tradingType
	return trading
}

// WithCredentials sets API credentials
func (f *TradingFactory) WithCredentials(apiKey, apiSecret string) *models.Trading {
	trading := f.Build()
	trading.APIKey = apiKey
	trading.APISecret = apiSecret
	return trading
}

// WithStatus sets the exchange status
func (f *TradingFactory) WithStatus(status string) *models.Trading {
	trading := f.Build()
	trading.Status = status
	return trading
}

// WithInfo sets exchange info
func (f *TradingFactory) WithInfo(info map[string]interface{}) *models.Trading {
	trading := f.Build()
	trading.Info = models.JSON(info)
	return trading
}

// BinanceTrading creates a Binance trading platform
func (f *TradingFactory) BinanceTrading() *models.Trading {
	trading := f.Build()
	trading.Name = "binance"
	trading.Type = "spot"
	binanceInfo := map[string]interface{}{
		"sandbox":   true,
		"base_url":  "https://testnet.binance.vision",
		"rate_limit": 1200,
	}
	trading.Info = models.JSON(binanceInfo)
	return trading
}

// SubAccountFactory creates SubAccount test data
type SubAccountFactory struct {
	factory *TestDataFactory
}

// NewSubAccountFactory creates a new sub-account factory
func NewSubAccountFactory() *SubAccountFactory {
	return &SubAccountFactory{
		factory: NewTestDataFactory(),
	}
}

// Build creates a basic sub-account with default values
func (f *SubAccountFactory) Build() *models.SubAccount {
	id := f.factory.nextID()
	return &models.SubAccount{
		ID:         uuid.New(),
		UserID:     uuid.New(), // Will be overridden
		TradingID: uuid.New(), // Will be overridden
		Name:       fmt.Sprintf("account_%d", id),
		Symbol:     "USDT",
		Balance:    1000.0 + float64(rand.Intn(9000)), // Random balance between 1000-10000
		Info:       models.JSON(map[string]interface{}{"type": "spot"}),
		CreatedAt:  time.Now().Add(-time.Duration(rand.Intn(24)) * time.Hour),
		UpdatedAt:  time.Now().Add(-time.Duration(rand.Intn(6)) * time.Hour),
	}
}

// WithUserID sets the user ID
func (f *SubAccountFactory) WithUserID(userID uuid.UUID) *models.SubAccount {
	account := f.Build()
	account.UserID = userID
	return account
}

// WithTradingID sets the trading ID
func (f *SubAccountFactory) WithTradingID(tradingID uuid.UUID) *models.SubAccount {
	account := f.Build()
	account.TradingID = tradingID
	return account
}

// WithUserAndTrading sets both user and trading IDs
func (f *SubAccountFactory) WithUserAndTrading(userID, tradingID uuid.UUID) *models.SubAccount {
	account := f.Build()
	account.UserID = userID
	account.TradingID = tradingID
	return account
}

// WithBalance sets the balance
func (f *SubAccountFactory) WithBalance(balance float64) *models.SubAccount {
	account := f.Build()
	account.Balance = balance
	return account
}

// WithSymbol sets the symbol
func (f *SubAccountFactory) WithSymbol(symbol string) *models.SubAccount {
	account := f.Build()
	account.Symbol = symbol
	return account
}

// WithName sets the account name
func (f *SubAccountFactory) WithName(name string) *models.SubAccount {
	account := f.Build()
	account.Name = name
	return account
}

// SpotAccount creates a spot trading account
func (f *SubAccountFactory) SpotAccount() *models.SubAccount {
	account := f.Build()
	account.Name = "spot"
	account.Symbol = "USDT"
	spotInfo := map[string]interface{}{
		"type":     "spot",
		"features": []string{"trading", "deposits", "withdrawals"},
	}
	account.Info = models.JSON(spotInfo)
	return account
}

// FuturesAccount creates a futures trading account
func (f *SubAccountFactory) FuturesAccount() *models.SubAccount {
	account := f.Build()
	account.Name = "futures"
	account.Symbol = "USDT"
	account.Balance = 5000.0 + float64(rand.Intn(15000)) // Higher balance for futures
	futuresInfo := map[string]interface{}{
		"type":     "futures",
		"leverage": 10,
		"features": []string{"trading", "margin"},
	}
	account.Info = models.JSON(futuresInfo)
	return account
}

// TransactionFactory creates Transaction test data
type TransactionFactory struct {
	factory *TestDataFactory
}

// NewTransactionFactory creates a new transaction factory
func NewTransactionFactory() *TransactionFactory {
	return &TransactionFactory{
		factory: NewTestDataFactory(),
	}
}

// Build creates a basic transaction with default values
func (f *TransactionFactory) Build() *models.Transaction {
	amount := 100.0 + float64(rand.Intn(900)) // Random amount between 100-1000
	return &models.Transaction{
		ID:             uuid.New(),
		UserID:         uuid.New(), // Will be overridden
		TradingID:     uuid.New(), // Will be overridden
		SubAccountID:   uuid.New(), // Will be overridden
		Timestamp:      time.Now().Add(-time.Duration(rand.Intn(168)) * time.Hour), // Last week
		Direction:      "credit",
		Reason:         "deposit",
		Amount:         amount,
		ClosingBalance: amount,
		Info:           models.JSON(map[string]interface{}{}),
	}
}

// WithUserID sets the user ID
func (f *TransactionFactory) WithUserID(userID uuid.UUID) *models.Transaction {
	tx := f.Build()
	tx.UserID = userID
	return tx
}

// WithTradingID sets the trading ID
func (f *TransactionFactory) WithTradingID(tradingID uuid.UUID) *models.Transaction {
	tx := f.Build()
	tx.TradingID = tradingID
	return tx
}

// WithSubAccountID sets the sub-account ID
func (f *TransactionFactory) WithSubAccountID(subAccountID uuid.UUID) *models.Transaction {
	tx := f.Build()
	tx.SubAccountID = subAccountID
	return tx
}

// WithAmount sets the transaction amount
func (f *TransactionFactory) WithAmount(amount float64) *models.Transaction {
	tx := f.Build()
	tx.Amount = amount
	return tx
}

// WithDirection sets the transaction direction
func (f *TransactionFactory) WithDirection(direction string) *models.Transaction {
	tx := f.Build()
	tx.Direction = direction
	return tx
}

// WithReason sets the transaction reason
func (f *TransactionFactory) WithReason(reason string) *models.Transaction {
	tx := f.Build()
	tx.Reason = reason
	return tx
}

// DepositTransaction creates a deposit transaction
func (f *TransactionFactory) DepositTransaction(amount float64, closingBalance float64) *models.Transaction {
	tx := f.Build()
	tx.Direction = "credit"
	tx.Reason = "deposit"
	tx.Amount = amount
	tx.ClosingBalance = closingBalance
	depositInfo := map[string]interface{}{
		"txid":   fmt.Sprintf("0x%x", rand.Int63()),
		"method": "bank_transfer",
	}
	tx.Info = models.JSON(depositInfo)
	return tx
}

// WithdrawTransaction creates a withdrawal transaction
func (f *TransactionFactory) WithdrawTransaction(amount float64, closingBalance float64) *models.Transaction {
	tx := f.Build()
	tx.Direction = "debit"
	tx.Reason = "withdrawal"
	tx.Amount = amount
	tx.ClosingBalance = closingBalance
	withdrawInfo := map[string]interface{}{
		"address": fmt.Sprintf("0x%x", rand.Int63()),
		"fee":     amount * 0.001, // 0.1% fee
	}
	tx.Info = models.JSON(withdrawInfo)
	return tx
}

// TradeTransaction creates a trade transaction
func (f *TransactionFactory) TradeTransaction(amount float64, closingBalance float64) *models.Transaction {
	tx := f.Build()
	tx.Direction = "credit" // Could be debit for losses
	tx.Reason = "trade_profit"
	tx.Amount = amount
	tx.ClosingBalance = closingBalance
	tradeInfo := map[string]interface{}{
		"trade_id": fmt.Sprintf("T%d", rand.Int63()),
		"symbol":   "BTCUSDT",
		"side":     "buy",
		"price":    45000.0 + float64(rand.Intn(10000)),
	}
	tx.Info = models.JSON(tradeInfo)
	return tx
}

// TradingLogFactory creates TradingLog test data
type TradingLogFactory struct {
	factory *TestDataFactory
}

// NewTradingLogFactory creates a new trading log factory
func NewTradingLogFactory() *TradingLogFactory {
	return &TradingLogFactory{
		factory: NewTestDataFactory(),
	}
}

// Build creates a basic trading log with default values
func (f *TradingLogFactory) Build() *models.TradingLog {
	id := f.factory.nextID()
	return &models.TradingLog{
		ID:           uuid.New(),
		UserID:       uuid.New(), // Will be overridden
		TradingID:   uuid.New(), // Will be overridden
		SubAccountID: nil,
		Timestamp:    time.Now().Add(-time.Duration(rand.Intn(24)) * time.Hour),
		Type:         "trade",
		Source:       "manual",
		Message:      fmt.Sprintf("Test trading log entry %d", id),
		Info:         models.JSON(map[string]interface{}{}),
	}
}

// WithUserID sets the user ID
func (f *TradingLogFactory) WithUserID(userID uuid.UUID) *models.TradingLog {
	log := f.Build()
	log.UserID = userID
	return log
}

// WithTradingID sets the trading ID
func (f *TradingLogFactory) WithTradingID(tradingID uuid.UUID) *models.TradingLog {
	log := f.Build()
	log.TradingID = tradingID
	return log
}

// WithSubAccountID sets the sub-account ID
func (f *TradingLogFactory) WithSubAccountID(subAccountID uuid.UUID) *models.TradingLog {
	log := f.Build()
	log.SubAccountID = &subAccountID
	return log
}

// WithTransactionID sets the transaction ID
func (f *TradingLogFactory) WithTransactionID(transactionID uuid.UUID) *models.TradingLog {
	log := f.Build()
	log.TransactionID = &transactionID
	return log
}

// BotLog creates a bot-generated trading log
func (f *TradingLogFactory) BotLog() *models.TradingLog {
	log := f.Build()
	log.Type = "strategy"
	log.Source = "bot"
	log.Message = "Grid strategy triggered: Position opened"
	botInfo := map[string]interface{}{
		"strategy": "grid",
		"symbol":   "ETHUSDT",
		"action":   "open_position",
		"bot_id":   fmt.Sprintf("bot_%d", f.factory.nextID()),
	}
	log.Info = models.JSON(botInfo)
	return log
}

// ErrorLog creates an error trading log
func (f *TradingLogFactory) ErrorLog() *models.TradingLog {
	log := f.Build()
	log.Type = "error"
	log.Source = "bot"
	log.Message = "API rate limit exceeded"
	errorInfo := map[string]interface{}{
		"error_code": "429",
		"endpoint":   "/api/v5/trade/order",
		"retry_after": 30,
	}
	log.Info = models.JSON(errorInfo)
	return log
}

// CompleteSetupFactory creates a complete user setup with related entities
type CompleteSetupFactory struct {
	UserFactory       *UserFactory
	TradingFactory   *TradingFactory
	SubAccountFactory *SubAccountFactory
	TransactionFactory *TransactionFactory
	TradingLogFactory *TradingLogFactory
}

// NewCompleteSetupFactory creates a new complete setup factory
func NewCompleteSetupFactory() *CompleteSetupFactory {
	return &CompleteSetupFactory{
		UserFactory:        NewUserFactory(),
		TradingFactory:    NewTradingFactory(),
		SubAccountFactory:  NewSubAccountFactory(),
		TransactionFactory: NewTransactionFactory(),
		TradingLogFactory:  NewTradingLogFactory(),
	}
}

// CreateUserWithTrading creates a user with one trading platform
func (f *CompleteSetupFactory) CreateUserWithTrading() (*models.User, *models.Trading) {
	user := f.UserFactory.Build()
	trading := f.TradingFactory.WithUserID(user.ID)
	return user, trading
}

// CreateCompleteUserSetup creates a user with trading platform, sub-account, and initial transaction
func (f *CompleteSetupFactory) CreateCompleteUserSetup() (*models.User, *models.Trading, *models.SubAccount, *models.Transaction) {
	user := f.UserFactory.Build()
	trading := f.TradingFactory.BinanceTrading()
	trading.UserID = user.ID
	subAccount := f.SubAccountFactory.SpotAccount()
	subAccount.UserID = user.ID
	subAccount.TradingID = trading.ID
	transaction := f.TransactionFactory.DepositTransaction(1000.0, 1000.0)
	transaction.UserID = user.ID
	transaction.TradingID = trading.ID
	transaction.SubAccountID = subAccount.ID
	
	return user, trading, subAccount, transaction
}

// CreateTradingScenario creates a full trading scenario with logs and transactions
func (f *CompleteSetupFactory) CreateTradingScenario() (*models.User, *models.Trading, *models.SubAccount, *models.Transaction, *models.TradingLog) {
	user, trading, subAccount, transaction := f.CreateCompleteUserSetup()
	
	tradingLog := f.TradingLogFactory.BotLog()
	tradingLog.UserID = user.ID
	tradingLog.TradingID = trading.ID
	tradingLog.SubAccountID = &subAccount.ID
	tradingLog.TransactionID = &transaction.ID
	
	return user, trading, subAccount, transaction, tradingLog
}