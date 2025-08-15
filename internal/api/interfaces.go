package api

import (
	"context"

	"tiris-backend/internal/services"

	"github.com/google/uuid"
)

// ExchangeServiceInterface defines the interface for exchange service operations
type ExchangeServiceInterface interface {
	CreateExchange(ctx context.Context, userID uuid.UUID, req *services.CreateExchangeRequest) (*services.ExchangeResponse, error)
	GetUserExchanges(ctx context.Context, userID uuid.UUID) ([]*services.ExchangeResponse, error)
	GetExchange(ctx context.Context, userID, exchangeID uuid.UUID) (*services.ExchangeResponse, error)
	UpdateExchange(ctx context.Context, userID, exchangeID uuid.UUID, req *services.UpdateExchangeRequest) (*services.ExchangeResponse, error)
	DeleteExchange(ctx context.Context, userID, exchangeID uuid.UUID) error
	ListExchanges(ctx context.Context, limit, offset int) ([]*services.ExchangeResponse, int64, error)
	GetExchangeByID(ctx context.Context, exchangeID uuid.UUID) (*services.ExchangeResponse, error)
}

// SubAccountServiceInterface defines the interface for sub-account service operations
type SubAccountServiceInterface interface {
	CreateSubAccount(ctx context.Context, userID uuid.UUID, req *services.CreateSubAccountRequest) (*services.SubAccountResponse, error)
	GetUserSubAccounts(ctx context.Context, userID uuid.UUID, exchangeID *uuid.UUID) ([]*services.SubAccountResponse, error)
	GetSubAccount(ctx context.Context, userID, subAccountID uuid.UUID) (*services.SubAccountResponse, error)
	UpdateSubAccount(ctx context.Context, userID, subAccountID uuid.UUID, req *services.UpdateSubAccountRequest) (*services.SubAccountResponse, error)
	UpdateBalance(ctx context.Context, userID, subAccountID uuid.UUID, req *services.UpdateBalanceRequest) (*services.SubAccountResponse, error)
	DeleteSubAccount(ctx context.Context, userID, subAccountID uuid.UUID) error
	GetSubAccountsBySymbol(ctx context.Context, userID uuid.UUID, symbol string) ([]*services.SubAccountResponse, error)
}