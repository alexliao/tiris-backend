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