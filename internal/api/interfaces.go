package api

import (
	"context"

	"tiris-backend/internal/services"

	"github.com/google/uuid"
)

// TradingServiceInterface defines the interface for trading service operations
type TradingServiceInterface interface {
	CreateTrading(ctx context.Context, userID uuid.UUID, req *services.CreateTradingRequest) (*services.TradingResponse, error)
	GetUserTradings(ctx context.Context, userID uuid.UUID) ([]*services.TradingResponse, error)
	GetTrading(ctx context.Context, userID, tradingID uuid.UUID) (*services.TradingResponse, error)
	UpdateTrading(ctx context.Context, userID, tradingID uuid.UUID, req *services.UpdateTradingRequest) (*services.TradingResponse, error)
	DeleteTrading(ctx context.Context, userID, tradingID uuid.UUID) error
	ListTradings(ctx context.Context, limit, offset int) ([]*services.TradingResponse, int64, error)
	GetTradingByID(ctx context.Context, tradingID uuid.UUID) (*services.TradingResponse, error)
}

// SubAccountServiceInterface defines the interface for sub-account service operations
type SubAccountServiceInterface interface {
	CreateSubAccount(ctx context.Context, userID uuid.UUID, req *services.CreateSubAccountRequest) (*services.SubAccountResponse, error)
	GetUserSubAccounts(ctx context.Context, userID uuid.UUID, tradingID *uuid.UUID) ([]*services.SubAccountResponse, error)
	GetSubAccount(ctx context.Context, userID, subAccountID uuid.UUID) (*services.SubAccountResponse, error)
	UpdateSubAccount(ctx context.Context, userID, subAccountID uuid.UUID, req *services.UpdateSubAccountRequest) (*services.SubAccountResponse, error)
	UpdateBalance(ctx context.Context, userID, subAccountID uuid.UUID, req *services.UpdateBalanceRequest) (*services.SubAccountResponse, error)
	DeleteSubAccount(ctx context.Context, userID, subAccountID uuid.UUID) error
	GetSubAccountsBySymbol(ctx context.Context, userID uuid.UUID, symbol string) ([]*services.SubAccountResponse, error)
}