package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"tiris-backend/internal/models"
	"tiris-backend/internal/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TradingLogProcessor handles business logic processing for trading logs
type TradingLogProcessor struct {
	repos     *repositories.Repositories
	validator *TradingLogValidator
}

// NewTradingLogProcessor creates a new trading log processor
func NewTradingLogProcessor(repos *repositories.Repositories) *TradingLogProcessor {
	return &TradingLogProcessor{
		repos:     repos,
		validator: NewTradingLogValidator(),
	}
}

// ProcessingResult contains the results of trading log processing
type ProcessingResult struct {
	CreatedTransactions []*models.Transaction
	UpdatedSubAccounts  []*models.SubAccount
	TradingLogRecord    *models.TradingLog
}

// ProcessTradingLog processes a trading log and performs business logic operations
func (p *TradingLogProcessor) ProcessTradingLog(ctx context.Context, db *gorm.DB, userID uuid.UUID, req *CreateTradingLogRequest) (*ProcessingResult, error) {
	// Validate the trading log type
	if err := p.validator.ValidateType(req.Type); err != nil {
		return nil, fmt.Errorf("type validation failed: %w", err)
	}

	// Validate and extract structured info if this is a business logic type
	tradingInfo, err := p.validator.ValidateInfoStructure(req.Info, req.Type)
	if err != nil {
		return nil, fmt.Errorf("info validation failed: %w", err)
	}

	// If this is not a business logic type, create a simple trading log
	if tradingInfo == nil {
		return p.createSimpleTradingLog(ctx, db, userID, req)
	}

	// Process business logic type within a database transaction
	var result *ProcessingResult
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var txErr error
		result, txErr = p.processBusinessLogicType(ctx, tx, userID, req, tradingInfo)
		return txErr
	})

	if err != nil {
		return nil, fmt.Errorf("transaction failed: %w", err)
	}

	return result, nil
}

// processBusinessLogicType handles long, short, and stop_loss trading log types
func (p *TradingLogProcessor) processBusinessLogicType(ctx context.Context, tx *gorm.DB, userID uuid.UUID, req *CreateTradingLogRequest, tradingInfo *TradingLogInfo) (*ProcessingResult, error) {
	// Verify trading ownership
	trading, err := p.repos.Trading.GetByID(ctx, req.TradingID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify trading: %w", err)
	}
	if trading == nil || trading.UserID != userID {
		return nil, fmt.Errorf("trading not found")
	}

	// Verify and get sub-accounts
	stockAccount, err := p.repos.SubAccount.GetByID(ctx, tradingInfo.StockAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get stock account: %w", err)
	}
	if stockAccount == nil || stockAccount.UserID != userID {
		return nil, fmt.Errorf("stock account not found")
	}

	var currencyAccount *models.SubAccount
	// For deposit/withdraw, currencyAccount is not needed
	if req.Type != "deposit" && req.Type != "withdraw" {
		currencyAccount, err = p.repos.SubAccount.GetByID(ctx, tradingInfo.CurrencyAccountID)
		if err != nil {
			return nil, fmt.Errorf("failed to get currency account: %w", err)
		}
		if currencyAccount == nil || currencyAccount.UserID != userID {
			return nil, fmt.Errorf("currency account not found")
		}
	}

	// Create the trading log record first
	tradingLogRecord := &models.TradingLog{
		ID:           uuid.New(),
		UserID:       userID,
		TradingID:   req.TradingID,
		SubAccountID: req.SubAccountID,
		Timestamp:    time.Now().UTC(),
		EventTime:    req.EventTime,
		Type:         req.Type,
		Source:       req.Source,
		Message:      req.Message,
		Info:         models.JSON(req.Info),
	}

	if err := tx.WithContext(ctx).Create(tradingLogRecord).Error; err != nil {
		return nil, fmt.Errorf("failed to create trading log: %w", err)
	}

	// Convert trading log to JSON for transaction info
	tradingLogJSON, err := json.Marshal(tradingLogRecord)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal trading log: %w", err)
	}

	var tradingLogInfo map[string]interface{}
	if err := json.Unmarshal(tradingLogJSON, &tradingLogInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal trading log info: %w", err)
	}

	// Process based on type
	var createdTransactions []*models.Transaction
	var updatedSubAccounts []*models.SubAccount

	switch req.Type {
	case "long":
		transactions, accounts, err := p.ProcessLongPosition(ctx, tradingInfo, stockAccount, currencyAccount, tradingLogInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to process long position: %w", err)
		}
		createdTransactions = transactions
		updatedSubAccounts = accounts

	case "short", "stop_loss":
		reason := "short"
		if req.Type == "stop_loss" {
			reason = "stop_loss"
		}
		transactions, accounts, err := p.ProcessShortPosition(ctx, tradingInfo, stockAccount, currencyAccount, tradingLogInfo, reason)
		if err != nil {
			return nil, fmt.Errorf("failed to process %s position: %w", req.Type, err)
		}
		createdTransactions = transactions
		updatedSubAccounts = accounts

	case "deposit":
		transactions, accounts, err := p.ProcessDeposit(ctx, tx, tradingInfo, stockAccount, tradingLogInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to process deposit: %w", err)
		}
		createdTransactions = transactions
		updatedSubAccounts = accounts

	case "withdraw":
		transactions, accounts, err := p.ProcessWithdraw(ctx, tradingInfo, stockAccount, tradingLogInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to process withdraw: %w", err)
		}
		createdTransactions = transactions
		updatedSubAccounts = accounts

	default:
		return nil, fmt.Errorf("unsupported business logic type: %s", req.Type)
	}

	return &ProcessingResult{
		CreatedTransactions: createdTransactions,
		UpdatedSubAccounts:  updatedSubAccounts,
		TradingLogRecord:    tradingLogRecord,
	}, nil
}

// ProcessLongPosition handles long position business logic
func (p *TradingLogProcessor) ProcessLongPosition(ctx context.Context, tradingInfo *TradingLogInfo, stockAccount, currencyAccount *models.SubAccount, tradingLogInfo map[string]interface{}) ([]*models.Transaction, []*models.SubAccount, error) {
	var transactions []*models.Transaction
	var updatedAccounts []*models.SubAccount

	// Calculate amounts
	totalCost := tradingInfo.Price*tradingInfo.Volume + tradingInfo.Fee

	// Check if currency account has sufficient balance for debit
	if currencyAccount.Balance < totalCost {
		return nil, nil, fmt.Errorf("insufficient balance in currency account: required %.8f, available %.8f",
			totalCost, currencyAccount.Balance)
	}

	// Transaction 1: Credit stock account with volume
	newStockBalance := stockAccount.Balance + tradingInfo.Volume
	stockTransactionID, err := p.repos.SubAccount.UpdateBalance(ctx, stockAccount.ID, newStockBalance, tradingInfo.Volume, "credit", "long", tradingLogInfo)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to update stock account balance: %w", err)
	}

	// Get the created stock transaction
	if stockTransactionID != nil {
		stockTransaction, err := p.repos.Transaction.GetByID(ctx, *stockTransactionID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get stock transaction: %w", err)
		}
		if stockTransaction != nil {
			// Set additional fields for trading transactions
			stockTransaction.Price = &tradingInfo.Price
			stockTransaction.QuoteSymbol = &tradingInfo.Currency
			transactions = append(transactions, stockTransaction)
		}
	}

	// Update stock account record
	stockAccount.Balance = newStockBalance
	updatedAccounts = append(updatedAccounts, stockAccount)

	// Transaction 2: Debit currency account with total cost
	newCurrencyBalance := currencyAccount.Balance - totalCost
	currencyTransactionID, err := p.repos.SubAccount.UpdateBalance(ctx, currencyAccount.ID, newCurrencyBalance, totalCost, "debit", "long", tradingLogInfo)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to update currency account balance: %w", err)
	}

	// Get the created currency transaction
	if currencyTransactionID != nil {
		currencyTransaction, err := p.repos.Transaction.GetByID(ctx, *currencyTransactionID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get currency transaction: %w", err)
		}
		if currencyTransaction != nil {
			// Set additional fields for trading transactions
			currencyTransaction.Price = &tradingInfo.Price
			currencyTransaction.QuoteSymbol = &tradingInfo.Currency
			transactions = append(transactions, currencyTransaction)
		}
	}

	// Update currency account record
	currencyAccount.Balance = newCurrencyBalance
	updatedAccounts = append(updatedAccounts, currencyAccount)

	return transactions, updatedAccounts, nil
}

// ProcessShortPosition handles short and stop_loss position business logic
func (p *TradingLogProcessor) ProcessShortPosition(ctx context.Context, tradingInfo *TradingLogInfo, stockAccount, currencyAccount *models.SubAccount, tradingLogInfo map[string]interface{}, reason string) ([]*models.Transaction, []*models.SubAccount, error) {
	var transactions []*models.Transaction
	var updatedAccounts []*models.SubAccount

	// Calculate amounts
	netProceeds := tradingInfo.Price*tradingInfo.Volume - tradingInfo.Fee

	// Check if stock account has sufficient balance for debit
	if stockAccount.Balance < tradingInfo.Volume {
		return nil, nil, fmt.Errorf("insufficient balance in stock account: required %.8f, available %.8f",
			tradingInfo.Volume, stockAccount.Balance)
	}

	// Transaction 1: Debit stock account with volume
	newStockBalance := stockAccount.Balance - tradingInfo.Volume
	stockTransactionID, err := p.repos.SubAccount.UpdateBalance(ctx, stockAccount.ID, newStockBalance, tradingInfo.Volume, "debit", reason, tradingLogInfo)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to update stock account balance: %w", err)
	}

	// Get the created stock transaction
	if stockTransactionID != nil {
		stockTransaction, err := p.repos.Transaction.GetByID(ctx, *stockTransactionID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get stock transaction: %w", err)
		}
		if stockTransaction != nil {
			// Set additional fields for trading transactions
			stockTransaction.Price = &tradingInfo.Price
			stockTransaction.QuoteSymbol = &tradingInfo.Currency
			transactions = append(transactions, stockTransaction)
		}
	}

	// Update stock account record
	stockAccount.Balance = newStockBalance
	updatedAccounts = append(updatedAccounts, stockAccount)

	// Transaction 2: Credit currency account with net proceeds
	newCurrencyBalance := currencyAccount.Balance + netProceeds
	currencyTransactionID, err := p.repos.SubAccount.UpdateBalance(ctx, currencyAccount.ID, newCurrencyBalance, netProceeds, "credit", reason, tradingLogInfo)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to update currency account balance: %w", err)
	}

	// Get the created currency transaction
	if currencyTransactionID != nil {
		currencyTransaction, err := p.repos.Transaction.GetByID(ctx, *currencyTransactionID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get currency transaction: %w", err)
		}
		if currencyTransaction != nil {
			// Set additional fields for trading transactions
			currencyTransaction.Price = &tradingInfo.Price
			currencyTransaction.QuoteSymbol = &tradingInfo.Currency
			transactions = append(transactions, currencyTransaction)
		}
	}

	// Update currency account record
	currencyAccount.Balance = newCurrencyBalance
	updatedAccounts = append(updatedAccounts, currencyAccount)

	return transactions, updatedAccounts, nil
}

// ProcessDeposit handles deposit business logic
func (p *TradingLogProcessor) ProcessDeposit(ctx context.Context, tx *gorm.DB, tradingInfo *TradingLogInfo, targetAccount *models.SubAccount, tradingLogInfo map[string]interface{}) ([]*models.Transaction, []*models.SubAccount, error) {
	var transactions []*models.Transaction
	var updatedAccounts []*models.SubAccount

	// Calculate new balance after deposit
	depositAmount := tradingInfo.Volume // Amount is stored in Volume field
	newBalance := targetAccount.Balance + depositAmount

	// Convert info to JSON for database function
	infoJSON, err := json.Marshal(tradingLogInfo)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal trading log info: %w", err)
	}
	
	// Create credit transaction for the deposit using transaction context
	var transactionIDStr string
	err = tx.WithContext(ctx).Raw(
		"SELECT update_sub_account_balance(?, ?, ?, ?, ?, ?::jsonb)",
		targetAccount.ID, newBalance, depositAmount, "credit", "deposit", string(infoJSON),
	).Row().Scan(&transactionIDStr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to update target account balance: %w", err)
	}
	
	// Parse the transaction ID
	transactionUUID, err := uuid.Parse(transactionIDStr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse transaction ID: %w", err)
	}
	transactionID := &transactionUUID

	// Get the created transaction
	if transactionID != nil {
		transaction, err := p.repos.Transaction.GetByID(ctx, *transactionID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get deposit transaction: %w", err)
		}
		if transaction != nil {
			// Set additional fields for deposit transactions
			price := 1.0 // Fixed price for deposits
			transaction.Price = &price
			transaction.QuoteSymbol = &tradingInfo.Stock // Currency is stored in Stock field
			transactions = append(transactions, transaction)
		}
	}

	// Update account record
	targetAccount.Balance = newBalance
	updatedAccounts = append(updatedAccounts, targetAccount)

	return transactions, updatedAccounts, nil
}

// ProcessWithdraw handles withdraw business logic
func (p *TradingLogProcessor) ProcessWithdraw(ctx context.Context, tradingInfo *TradingLogInfo, sourceAccount *models.SubAccount, tradingLogInfo map[string]interface{}) ([]*models.Transaction, []*models.SubAccount, error) {
	var transactions []*models.Transaction
	var updatedAccounts []*models.SubAccount

	withdrawAmount := tradingInfo.Volume // Amount is stored in Volume field

	// Check if source account has sufficient balance for withdrawal
	if sourceAccount.Balance < withdrawAmount {
		return nil, nil, fmt.Errorf("insufficient balance in source account: required %.8f, available %.8f",
			withdrawAmount, sourceAccount.Balance)
	}

	// Calculate new balance after withdrawal
	newBalance := sourceAccount.Balance - withdrawAmount

	// Create debit transaction for the withdrawal
	transactionID, err := p.repos.SubAccount.UpdateBalance(ctx, sourceAccount.ID, newBalance, withdrawAmount, "debit", "withdraw", tradingLogInfo)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to update source account balance: %w", err)
	}

	// Get the created transaction
	if transactionID != nil {
		transaction, err := p.repos.Transaction.GetByID(ctx, *transactionID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get withdraw transaction: %w", err)
		}
		if transaction != nil {
			// Set additional fields for withdraw transactions
			price := 1.0 // Fixed price for withdrawals
			transaction.Price = &price
			transaction.QuoteSymbol = &tradingInfo.Stock // Currency is stored in Stock field
			transactions = append(transactions, transaction)
		}
	}

	// Update account record
	sourceAccount.Balance = newBalance
	updatedAccounts = append(updatedAccounts, sourceAccount)

	return transactions, updatedAccounts, nil
}

// createSimpleTradingLog creates a trading log without business logic processing
func (p *TradingLogProcessor) createSimpleTradingLog(ctx context.Context, db *gorm.DB, userID uuid.UUID, req *CreateTradingLogRequest) (*ProcessingResult, error) {
	// Verify trading ownership
	trading, err := p.repos.Trading.GetByID(ctx, req.TradingID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify trading: %w", err)
	}
	if trading == nil || trading.UserID != userID {
		return nil, fmt.Errorf("trading not found")
	}

	// Verify sub-account ownership if provided
	if req.SubAccountID != nil {
		subAccount, err := p.repos.SubAccount.GetByID(ctx, *req.SubAccountID)
		if err != nil {
			return nil, fmt.Errorf("failed to verify sub-account: %w", err)
		}
		if subAccount == nil || subAccount.UserID != userID {
			return nil, fmt.Errorf("sub-account not found")
		}
	}

	// Verify transaction ownership if provided
	if req.TransactionID != nil {
		transaction, err := p.repos.Transaction.GetByID(ctx, *req.TransactionID)
		if err != nil {
			return nil, fmt.Errorf("failed to verify transaction: %w", err)
		}
		if transaction == nil || transaction.UserID != userID {
			return nil, fmt.Errorf("transaction not found")
		}
	}

	// Create info map with metadata
	infoMap := req.Info
	if infoMap == nil {
		infoMap = make(map[string]interface{})
	}
	// Add metadata
	infoMap["created_by"] = "api"
	infoMap["api_version"] = "v1"
	infoMap["trading_type"] = trading.Type

	// Create trading log model
	tradingLog := &models.TradingLog{
		ID:            uuid.New(),
		UserID:        userID,
		TradingID:    req.TradingID,
		SubAccountID:  req.SubAccountID,
		TransactionID: req.TransactionID,
		Timestamp:     time.Now().UTC(),
		EventTime:     req.EventTime,
		Type:          req.Type,
		Source:        req.Source,
		Message:       req.Message,
		Info:          models.JSON(infoMap),
	}

	// Save to database
	if err := p.repos.TradingLog.Create(ctx, tradingLog); err != nil {
		return nil, fmt.Errorf("failed to create trading log: %w", err)
	}

	return &ProcessingResult{
		CreatedTransactions: []*models.Transaction{},
		UpdatedSubAccounts:  []*models.SubAccount{},
		TradingLogRecord:    tradingLog,
	}, nil
}
