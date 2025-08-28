package api

import (
	"net/http"
	"time"

	"tiris-backend/internal/middleware"
	"tiris-backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TransactionHandler handles transaction query endpoints
type TransactionHandler struct {
	transactionService *services.TransactionService
}

// NewTransactionHandler creates a new transaction handler
func NewTransactionHandler(transactionService *services.TransactionService) *TransactionHandler {
	return &TransactionHandler{
		transactionService: transactionService,
	}
}

// GetUserTransactions retrieves all transactions for the current user
// @Summary Get user transactions
// @Description Retrieves transaction history for the authenticated user with filtering and pagination
// @Tags Transactions
// @Produce json
// @Security BearerAuth
// @Param direction query string false "Filter by direction" Enums(debit, credit)
// @Param reason query string false "Filter by reason"
// @Param start_date query string false "Start date (RFC3339 format)"
// @Param end_date query string false "End date (RFC3339 format)"
// @Param min_amount query number false "Minimum amount filter"
// @Param max_amount query number false "Maximum amount filter"
// @Param limit query int false "Number of transactions to return" default(100)
// @Param offset query int false "Number of transactions to skip" default(0)
// @Success 200 {object} services.TransactionQueryResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /transactions [get]
func (h *TransactionHandler) GetUserTransactions(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, CreateErrorResponse(
			"AUTH_REQUIRED",
			"Authentication required",
			"",
			getTraceID(c),
		))
		return
	}

	var req services.TransactionQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, CreateErrorResponse(
			"INVALID_QUERY_PARAMS",
			"Invalid query parameters",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	transactions, err := h.transactionService.GetUserTransactions(c.Request.Context(), userID, &req)
	if err != nil {
		if err.Error() == "start date cannot be after end date" ||
			err.Error() == "min amount cannot be greater than max amount" {
			c.JSON(http.StatusBadRequest, CreateErrorResponse(
				"INVALID_FILTER_RANGE",
				"Invalid filter range",
				err.Error(),
				getTraceID(c),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"TRANSACTIONS_QUERY_FAILED",
			"Failed to query transactions",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(transactions, getTraceID(c)))
}

// GetSubAccountTransactions retrieves transactions for a specific sub-account
// @Summary Get sub-account transactions
// @Description Retrieves transaction history for a specific sub-account (must belong to authenticated user)
// @Tags Transactions
// @Produce json
// @Security BearerAuth
// @Param sub_account_id path string true "Sub-account ID"
// @Param direction query string false "Filter by direction" Enums(debit, credit)
// @Param reason query string false "Filter by reason"
// @Param start_date query string false "Start date (RFC3339 format)"
// @Param end_date query string false "End date (RFC3339 format)"
// @Param min_amount query number false "Minimum amount filter"
// @Param max_amount query number false "Maximum amount filter"
// @Param limit query int false "Number of transactions to return" default(100)
// @Param offset query int false "Number of transactions to skip" default(0)
// @Success 200 {object} services.TransactionQueryResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /transactions/sub-account/{sub_account_id} [get]
func (h *TransactionHandler) GetSubAccountTransactions(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, CreateErrorResponse(
			"AUTH_REQUIRED",
			"Authentication required",
			"",
			getTraceID(c),
		))
		return
	}

	subAccountIDStr := c.Param("sub_account_id")
	subAccountID, err := uuid.Parse(subAccountIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, CreateErrorResponse(
			"INVALID_SUBACCOUNT_ID",
			"Invalid sub-account ID format",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	var req services.TransactionQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, CreateErrorResponse(
			"INVALID_QUERY_PARAMS",
			"Invalid query parameters",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	transactions, err := h.transactionService.GetSubAccountTransactions(c.Request.Context(), userID, subAccountID, &req)
	if err != nil {
		if err.Error() == "sub-account not found" {
			c.JSON(http.StatusNotFound, CreateErrorResponse(
				"SUBACCOUNT_NOT_FOUND",
				"Sub-account not found",
				err.Error(),
				getTraceID(c),
			))
			return
		}
		if err.Error() == "start date cannot be after end date" ||
			err.Error() == "min amount cannot be greater than max amount" {
			c.JSON(http.StatusBadRequest, CreateErrorResponse(
				"INVALID_FILTER_RANGE",
				"Invalid filter range",
				err.Error(),
				getTraceID(c),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"TRANSACTIONS_QUERY_FAILED",
			"Failed to query sub-account transactions",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(transactions, getTraceID(c)))
}

// GetExchangeTransactions retrieves transactions for a specific exchange
// @Summary Get exchange transactions
// @Description Retrieves transaction history for a specific exchange (must belong to authenticated user)
// @Tags Transactions
// @Produce json
// @Security BearerAuth
// @Param exchange_id path string true "Exchange ID"
// @Param direction query string false "Filter by direction" Enums(debit, credit)
// @Param reason query string false "Filter by reason"
// @Param start_date query string false "Start date (RFC3339 format)"
// @Param end_date query string false "End date (RFC3339 format)"
// @Param min_amount query number false "Minimum amount filter"
// @Param max_amount query number false "Maximum amount filter"
// @Param limit query int false "Number of transactions to return" default(100)
// @Param offset query int false "Number of transactions to skip" default(0)
// @Success 200 {object} services.TransactionQueryResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /transactions/exchange/{exchange_id} [get]
func (h *TransactionHandler) GetExchangeTransactions(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, CreateErrorResponse(
			"AUTH_REQUIRED",
			"Authentication required",
			"",
			getTraceID(c),
		))
		return
	}

	exchangeIDStr := c.Param("exchange_id")
	exchangeID, err := uuid.Parse(exchangeIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, CreateErrorResponse(
			"INVALID_EXCHANGE_ID",
			"Invalid exchange ID format",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	var req services.TransactionQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, CreateErrorResponse(
			"INVALID_QUERY_PARAMS",
			"Invalid query parameters",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	transactions, err := h.transactionService.GetExchangeTransactions(c.Request.Context(), userID, exchangeID, &req)
	if err != nil {
		if err.Error() == "trading platform not found" {
			c.JSON(http.StatusNotFound, CreateErrorResponse(
				"EXCHANGE_NOT_FOUND",
				"Exchange not found",
				err.Error(),
				getTraceID(c),
			))
			return
		}
		if err.Error() == "start date cannot be after end date" ||
			err.Error() == "min amount cannot be greater than max amount" {
			c.JSON(http.StatusBadRequest, CreateErrorResponse(
				"INVALID_FILTER_RANGE",
				"Invalid filter range",
				err.Error(),
				getTraceID(c),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"TRANSACTIONS_QUERY_FAILED",
			"Failed to query exchange transactions",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(transactions, getTraceID(c)))
}

// GetTransaction retrieves a specific transaction by ID
// @Summary Get transaction by ID
// @Description Retrieves a specific transaction by ID (must belong to authenticated user)
// @Tags Transactions
// @Produce json
// @Security BearerAuth
// @Param id path string true "Transaction ID"
// @Success 200 {object} services.TransactionResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /transactions/{id} [get]
func (h *TransactionHandler) GetTransaction(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, CreateErrorResponse(
			"AUTH_REQUIRED",
			"Authentication required",
			"",
			getTraceID(c),
		))
		return
	}

	transactionIDStr := c.Param("id")
	transactionID, err := uuid.Parse(transactionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, CreateErrorResponse(
			"INVALID_TRANSACTION_ID",
			"Invalid transaction ID format",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	transaction, err := h.transactionService.GetTransaction(c.Request.Context(), userID, transactionID)
	if err != nil {
		if err.Error() == "transaction not found" {
			c.JSON(http.StatusNotFound, CreateErrorResponse(
				"TRANSACTION_NOT_FOUND",
				"Transaction not found",
				err.Error(),
				getTraceID(c),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"TRANSACTION_GET_FAILED",
			"Failed to get transaction",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(transaction, getTraceID(c)))
}

// GetTransactionsByTimeRange retrieves transactions within a specific time range
// @Summary Get transactions by time range
// @Description Retrieves transactions within a specific time range for the authenticated user
// @Tags Transactions
// @Produce json
// @Security BearerAuth
// @Param start_time query string true "Start time (RFC3339 format)"
// @Param end_time query string true "End time (RFC3339 format)"
// @Param direction query string false "Filter by direction" Enums(debit, credit)
// @Param reason query string false "Filter by reason"
// @Param min_amount query number false "Minimum amount filter"
// @Param max_amount query number false "Maximum amount filter"
// @Param limit query int false "Number of transactions to return" default(100)
// @Param offset query int false "Number of transactions to skip" default(0)
// @Success 200 {object} services.TransactionQueryResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /transactions/time-range [get]
func (h *TransactionHandler) GetTransactionsByTimeRange(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, CreateErrorResponse(
			"AUTH_REQUIRED",
			"Authentication required",
			"",
			getTraceID(c),
		))
		return
	}

	// Parse required time range parameters
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")

	if startTimeStr == "" || endTimeStr == "" {
		c.JSON(http.StatusBadRequest, CreateErrorResponse(
			"MISSING_TIME_RANGE",
			"Start time and end time are required",
			"",
			getTraceID(c),
		))
		return
	}

	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, CreateErrorResponse(
			"INVALID_START_TIME",
			"Invalid start time format (use RFC3339)",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	endTime, err := time.Parse(time.RFC3339, endTimeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, CreateErrorResponse(
			"INVALID_END_TIME",
			"Invalid end time format (use RFC3339)",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	var req services.TransactionQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, CreateErrorResponse(
			"INVALID_QUERY_PARAMS",
			"Invalid query parameters",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	transactions, err := h.transactionService.GetTransactionsByTimeRange(c.Request.Context(), userID, startTime, endTime, &req)
	if err != nil {
		if err.Error() == "start time cannot be after end time" ||
			err.Error() == "min amount cannot be greater than max amount" {
			c.JSON(http.StatusBadRequest, CreateErrorResponse(
				"INVALID_FILTER_RANGE",
				"Invalid filter range",
				err.Error(),
				getTraceID(c),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"TRANSACTIONS_QUERY_FAILED",
			"Failed to query transactions by time range",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(transactions, getTraceID(c)))
}

// ListAllTransactions lists all transactions with filtering (admin only)
// @Summary List all transactions
// @Description Lists all transactions with filtering and pagination (admin only)
// @Tags Transactions
// @Produce json
// @Security BearerAuth
// @Param direction query string false "Filter by direction" Enums(debit, credit)
// @Param reason query string false "Filter by reason"
// @Param start_date query string false "Start date (RFC3339 format)"
// @Param end_date query string false "End date (RFC3339 format)"
// @Param min_amount query number false "Minimum amount filter"
// @Param max_amount query number false "Maximum amount filter"
// @Param limit query int false "Number of transactions to return" default(100)
// @Param offset query int false "Number of transactions to skip" default(0)
// @Success 200 {object} services.TransactionQueryResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /admin/transactions [get]
func (h *TransactionHandler) ListAllTransactions(c *gin.Context) {
	var req services.TransactionQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, CreateErrorResponse(
			"INVALID_QUERY_PARAMS",
			"Invalid query parameters",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	transactions, err := h.transactionService.ListAllTransactions(c.Request.Context(), &req)
	if err != nil {
		if err.Error() == "start date cannot be after end date" ||
			err.Error() == "min amount cannot be greater than max amount" {
			c.JSON(http.StatusBadRequest, CreateErrorResponse(
				"INVALID_FILTER_RANGE",
				"Invalid filter range",
				err.Error(),
				getTraceID(c),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"TRANSACTIONS_LIST_FAILED",
			"Failed to list transactions",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(transactions, getTraceID(c)))
}

// GetTransactionByID retrieves transaction by ID (admin only)
// @Summary Get transaction by ID
// @Description Retrieves a transaction by ID (admin only)
// @Tags Transactions
// @Produce json
// @Security BearerAuth
// @Param id path string true "Transaction ID"
// @Success 200 {object} services.TransactionResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /admin/transactions/{id} [get]
func (h *TransactionHandler) GetTransactionByID(c *gin.Context) {
	transactionIDStr := c.Param("id")
	transactionID, err := uuid.Parse(transactionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, CreateErrorResponse(
			"INVALID_TRANSACTION_ID",
			"Invalid transaction ID format",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	transaction, err := h.transactionService.GetTransactionByID(c.Request.Context(), transactionID)
	if err != nil {
		if err.Error() == "transaction not found" {
			c.JSON(http.StatusNotFound, CreateErrorResponse(
				"TRANSACTION_NOT_FOUND",
				"Transaction not found",
				err.Error(),
				getTraceID(c),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"TRANSACTION_GET_FAILED",
			"Failed to get transaction",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(transaction, getTraceID(c)))
}
