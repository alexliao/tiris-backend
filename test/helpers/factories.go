package helpers

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"tiris-backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/datatypes"
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
		Settings:  datatypes.JSON(`{"theme": "dark"}`),
		Info:      datatypes.JSON(`{}`),
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
	if settingsJSON, err := json.Marshal(settings); err == nil {
		user.Settings = datatypes.JSON(settingsJSON)
	}
	return user
}

// WithInfo sets user info
func (f *UserFactory) WithInfo(info map[string]interface{}) *models.User {
	user := f.Build()
	if infoJSON, err := json.Marshal(info); err == nil {
		user.Info = datatypes.JSON(infoJSON)
	}
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
	if settingsJSON, err := json.Marshal(adminSettings); err == nil {
		user.Settings = datatypes.JSON(settingsJSON)
	}
	adminInfo := map[string]interface{}{
		"permissions": []string{"read", "write", "admin"},
	}
	if infoJSON, err := json.Marshal(adminInfo); err == nil {
		user.Info = datatypes.JSON(infoJSON)
	}
	return user
}

// ExchangeFactory creates Exchange test data
type ExchangeFactory struct {
	factory *TestDataFactory
}

// NewExchangeFactory creates a new exchange factory
func NewExchangeFactory() *ExchangeFactory {
	return &ExchangeFactory{
		factory: NewTestDataFactory(),
	}
}

// Build creates a basic exchange with default values
func (f *ExchangeFactory) Build() *models.Exchange {
	id := f.factory.nextID()
	return &models.Exchange{
		ID:        uuid.New(),
		UserID:    uuid.New(), // Will be overridden in WithUserID
		Name:      fmt.Sprintf("exchange_%d", id),
		Type:      "spot",
		APIKey:    fmt.Sprintf("test_api_key_%d", id),
		APISecret: fmt.Sprintf("test_api_secret_%d", id),
		Status:    "active",
		Info:      datatypes.JSON(`{"sandbox": true}`),
		CreatedAt: time.Now().Add(-time.Duration(rand.Intn(48)) * time.Hour),
		UpdatedAt: time.Now().Add(-time.Duration(rand.Intn(12)) * time.Hour),
	}
}

// WithUserID sets the user ID
func (f *ExchangeFactory) WithUserID(userID uuid.UUID) *models.Exchange {
	exchange := f.Build()
	exchange.UserID = userID
	return exchange
}

// WithName sets the exchange name
func (f *ExchangeFactory) WithName(name string) *models.Exchange {
	exchange := f.Build()
	exchange.Name = name
	return exchange
}

// WithType sets the exchange type
func (f *ExchangeFactory) WithType(exchangeType string) *models.Exchange {
	exchange := f.Build()
	exchange.Type = exchangeType
	return exchange
}

// WithCredentials sets API credentials
func (f *ExchangeFactory) WithCredentials(apiKey, apiSecret string) *models.Exchange {
	exchange := f.Build()
	exchange.APIKey = apiKey
	exchange.APISecret = apiSecret
	return exchange
}

// WithStatus sets the exchange status
func (f *ExchangeFactory) WithStatus(status string) *models.Exchange {
	exchange := f.Build()
	exchange.Status = status
	return exchange
}

// WithInfo sets exchange info
func (f *ExchangeFactory) WithInfo(info map[string]interface{}) *models.Exchange {
	exchange := f.Build()
	if infoJSON, err := json.Marshal(info); err == nil {
		exchange.Info = datatypes.JSON(infoJSON)
	}
	return exchange
}

// BinanceExchange creates a Binance exchange
func (f *ExchangeFactory) BinanceExchange() *models.Exchange {
	exchange := f.Build()
	exchange.Name = "binance"
	exchange.Type = "spot"
	binanceInfo := map[string]interface{}{
		"sandbox":   true,
		"base_url":  "https://testnet.binance.vision",
		"rate_limit": 1200,
	}
	if infoJSON, err := json.Marshal(binanceInfo); err == nil {
		exchange.Info = datatypes.JSON(infoJSON)
	}
	return exchange
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
		ExchangeID: uuid.New(), // Will be overridden
		Name:       fmt.Sprintf("account_%d", id),
		Symbol:     "USDT",
		Balance:    1000.0 + float64(rand.Intn(9000)), // Random balance between 1000-10000
		Info:       datatypes.JSON(`{"type": "spot"}`),
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

// WithExchangeID sets the exchange ID
func (f *SubAccountFactory) WithExchangeID(exchangeID uuid.UUID) *models.SubAccount {
	account := f.Build()
	account.ExchangeID = exchangeID
	return account
}

// WithUserAndExchange sets both user and exchange IDs
func (f *SubAccountFactory) WithUserAndExchange(userID, exchangeID uuid.UUID) *models.SubAccount {
	account := f.Build()
	account.UserID = userID
	account.ExchangeID = exchangeID
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
	if infoJSON, err := json.Marshal(spotInfo); err == nil {
		account.Info = datatypes.JSON(infoJSON)
	}
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
	if infoJSON, err := json.Marshal(futuresInfo); err == nil {
		account.Info = datatypes.JSON(infoJSON)
	}
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
		ExchangeID:     uuid.New(), // Will be overridden
		SubAccountID:   uuid.New(), // Will be overridden
		Timestamp:      time.Now().Add(-time.Duration(rand.Intn(168)) * time.Hour), // Last week
		Direction:      "credit",
		Reason:         "deposit",
		Amount:         amount,
		ClosingBalance: amount,
		Info:           datatypes.JSON(`{}`),
	}
}

// WithUserID sets the user ID
func (f *TransactionFactory) WithUserID(userID uuid.UUID) *models.Transaction {
	tx := f.Build()
	tx.UserID = userID
	return tx
}

// WithExchangeID sets the exchange ID
func (f *TransactionFactory) WithExchangeID(exchangeID uuid.UUID) *models.Transaction {
	tx := f.Build()
	tx.ExchangeID = exchangeID
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
	if infoJSON, err := json.Marshal(depositInfo); err == nil {
		tx.Info = datatypes.JSON(infoJSON)
	}
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
	if infoJSON, err := json.Marshal(withdrawInfo); err == nil {
		tx.Info = datatypes.JSON(infoJSON)
	}
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
	if infoJSON, err := json.Marshal(tradeInfo); err == nil {
		tx.Info = datatypes.JSON(infoJSON)
	}
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
		ExchangeID:   uuid.New(), // Will be overridden
		SubAccountID: nil,
		Timestamp:    time.Now().Add(-time.Duration(rand.Intn(24)) * time.Hour),
		Type:         "trade",
		Source:       "manual",
		Message:      fmt.Sprintf("Test trading log entry %d", id),
		Info:         datatypes.JSON(`{}`),
	}
}

// WithUserID sets the user ID
func (f *TradingLogFactory) WithUserID(userID uuid.UUID) *models.TradingLog {
	log := f.Build()
	log.UserID = userID
	return log
}

// WithExchangeID sets the exchange ID
func (f *TradingLogFactory) WithExchangeID(exchangeID uuid.UUID) *models.TradingLog {
	log := f.Build()
	log.ExchangeID = exchangeID
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
	if infoJSON, err := json.Marshal(botInfo); err == nil {
		log.Info = datatypes.JSON(infoJSON)
	}
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
	if infoJSON, err := json.Marshal(errorInfo); err == nil {
		log.Info = datatypes.JSON(infoJSON)
	}
	return log
}

// CompleteSetupFactory creates a complete user setup with related entities
type CompleteSetupFactory struct {
	UserFactory       *UserFactory
	ExchangeFactory   *ExchangeFactory
	SubAccountFactory *SubAccountFactory
	TransactionFactory *TransactionFactory
	TradingLogFactory *TradingLogFactory
}

// NewCompleteSetupFactory creates a new complete setup factory
func NewCompleteSetupFactory() *CompleteSetupFactory {
	return &CompleteSetupFactory{
		UserFactory:        NewUserFactory(),
		ExchangeFactory:    NewExchangeFactory(),
		SubAccountFactory:  NewSubAccountFactory(),
		TransactionFactory: NewTransactionFactory(),
		TradingLogFactory:  NewTradingLogFactory(),
	}
}

// CreateUserWithExchange creates a user with one exchange
func (f *CompleteSetupFactory) CreateUserWithExchange() (*models.User, *models.Exchange) {
	user := f.UserFactory.Build()
	exchange := f.ExchangeFactory.WithUserID(user.ID)
	return user, exchange
}

// CreateCompleteUserSetup creates a user with exchange, sub-account, and initial transaction
func (f *CompleteSetupFactory) CreateCompleteUserSetup() (*models.User, *models.Exchange, *models.SubAccount, *models.Transaction) {
	user := f.UserFactory.Build()
	exchange := f.ExchangeFactory.BinanceExchange()
	exchange.UserID = user.ID
	subAccount := f.SubAccountFactory.SpotAccount()
	subAccount.UserID = user.ID
	subAccount.ExchangeID = exchange.ID
	transaction := f.TransactionFactory.DepositTransaction(1000.0, 1000.0)
	transaction.UserID = user.ID
	transaction.ExchangeID = exchange.ID
	transaction.SubAccountID = subAccount.ID
	
	return user, exchange, subAccount, transaction
}

// CreateTradingScenario creates a full trading scenario with logs and transactions
func (f *CompleteSetupFactory) CreateTradingScenario() (*models.User, *models.Exchange, *models.SubAccount, *models.Transaction, *models.TradingLog) {
	user, exchange, subAccount, transaction := f.CreateCompleteUserSetup()
	
	tradingLog := f.TradingLogFactory.BotLog()
	tradingLog.UserID = user.ID
	tradingLog.ExchangeID = exchange.ID
	tradingLog.SubAccountID = &subAccount.ID
	tradingLog.TransactionID = &transaction.ID
	
	return user, exchange, subAccount, transaction, tradingLog
}