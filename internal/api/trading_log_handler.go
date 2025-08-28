package api

import (
	"net/http"
	"time"

	"tiris-backend/internal/middleware"
	"tiris-backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TradingLogHandler handles trading log management endpoints
type TradingLogHandler struct {
	tradingLogService *services.TradingLogService
}

// NewTradingLogHandler creates a new trading log handler
func NewTradingLogHandler(tradingLogService *services.TradingLogService) *TradingLogHandler {
	return &TradingLogHandler{
		tradingLogService: tradingLogService,
	}
}

// CreateTradingLog creates a new trading log entry
// @Summary Create trading log
// @Description Creates a new trading log entry for the authenticated user. 
// @Description 
// @Description **Important**: The 'info' field structure must match the 'type' field:
// @Description 
// @Description **Business Logic Types** (trigger automatic financial calculations):
// @Description 
// @Description **For long/short/stop_loss types** - Required fields in 'info':
// @Description - stock_account_id (string): Sub-account UUID for the asset (e.g., ETH account)
// @Description - currency_account_id (string): Sub-account UUID for the currency (e.g., USDT account)  
// @Description - price (number): Price per unit (must be positive, up to 8 decimal places)
// @Description - volume (number): Quantity traded (must be positive, up to 8 decimal places)
// @Description - stock (string): Asset symbol, 1-20 characters (e.g., "ETH")
// @Description - currency (string): Currency symbol, 1-20 characters (e.g., "USDT")
// @Description - fee (number): Trading fee (must be non-negative, up to 8 decimal places)
// @Description 
// @Description **For deposit/withdraw types** - Required fields in 'info':
// @Description - account_id (string): Target sub-account UUID for the operation
// @Description - amount (number): Amount to deposit/withdraw (must be positive, up to 8 decimal places)
// @Description - currency (string): Currency symbol, 1-20 characters (e.g., "USDT")
// @Description 
// @Description **Request Examples**:
// @Description 
// @Description **Long Position Example:**
// @Description <pre><code>{
// @Description ⠀⠀"trading_id": "453f0347-3959-49de-8e3f-1cf7c8e0827c",
// @Description ⠀⠀"type": "long", 
// @Description ⠀⠀"source": "bot",
// @Description ⠀⠀"message": "ETH long position opened",
// @Description ⠀⠀"info": {
// @Description ⠀⠀⠀⠀"stock_account_id": "eth-account-uuid",
// @Description ⠀⠀⠀⠀"currency_account_id": "usdt-account-uuid", 
// @Description ⠀⠀⠀⠀"price": 3000.00,
// @Description ⠀⠀⠀⠀"volume": 2.0,
// @Description ⠀⠀⠀⠀"stock": "ETH",
// @Description ⠀⠀⠀⠀"currency": "USDT",
// @Description ⠀⠀⠀⠀"fee": 12.00
// @Description ⠀⠀}
// @Description }</code></pre>
// @Description 
// @Description **Deposit Example:**
// @Description <pre><code>{
// @Description ⠀⠀"trading_id": "453f0347-3959-49de-8e3f-1cf7c8e0827c",
// @Description ⠀⠀"type": "deposit",
// @Description ⠀⠀"source": "api", 
// @Description ⠀⠀"message": "USDT deposit to account",
// @Description ⠀⠀"info": {
// @Description ⠀⠀⠀⠀"account_id": "usdt-account-uuid",
// @Description ⠀⠀⠀⠀"amount": 1000.00,
// @Description ⠀⠀⠀⠀"currency": "USDT"
// @Description ⠀⠀}
// @Description }</code></pre>
// @Description 
// @Description **Other Types**: Can use any object structure in the 'info' field
// @Tags TradingLogs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body services.CreateTradingLogRequest true "Create trading log request"
// @Success 201 {object} services.TradingLogResponse
// @Failure 400 {object} ErrorResponse "Bad Request - Invalid request format, missing required fields, or incorrect 'info' structure for the specified 'type'. Common validation errors: Missing required 'info' fields for business logic types, Invalid data types or values in 'info' fields, Non-existent sub-account IDs referenced in 'info' fields"
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse "Not Found - Trading ID or sub-account IDs referenced in 'info' field do not exist"
// @Failure 422 {object} ErrorResponse "Unprocessable Entity - Business logic validation failed (e.g., insufficient balance for withdraw operations)"
// @Failure 500 {object} ErrorResponse
// @Router /trading-logs [post]
func (h *TradingLogHandler) CreateTradingLog(c *gin.Context) {
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

	var req services.CreateTradingLogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, CreateErrorResponse(
			"INVALID_REQUEST",
			"Invalid request format",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	tradingLog, err := h.tradingLogService.CreateTradingLog(c.Request.Context(), userID, &req)
	if err != nil {
		if err.Error() == "trading not found" {
			c.JSON(http.StatusNotFound, CreateErrorResponse(
				"TRADING_NOT_FOUND",
				"Trading not found",
				err.Error(),
				getTraceID(c),
			))
			return
		}
		if err.Error() == "sub-account not found" {
			c.JSON(http.StatusNotFound, CreateErrorResponse(
				"SUBACCOUNT_NOT_FOUND",
				"Sub-account not found",
				err.Error(),
				getTraceID(c),
			))
			return
		}
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
			"TRADING_LOG_CREATE_FAILED",
			"Failed to create trading log",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusCreated, CreateSuccessResponse(tradingLog, getTraceID(c)))
}

// GetUserTradingLogs retrieves all trading logs for the current user
// @Summary Get user trading logs
// @Description Retrieves trading log history for the authenticated user with filtering and pagination
// @Tags TradingLogs
// @Produce json
// @Security BearerAuth
// @Param type query string false "Filter by log type"
// @Param source query string false "Filter by source" Enums(manual, bot)
// @Param start_date query string false "Start date (RFC3339 format)"
// @Param end_date query string false "End date (RFC3339 format)"
// @Param limit query int false "Number of logs to return" default(100)
// @Param offset query int false "Number of logs to skip" default(0)
// @Success 200 {object} services.TradingLogQueryResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /trading-logs [get]
func (h *TradingLogHandler) GetUserTradingLogs(c *gin.Context) {
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

	var req services.TradingLogQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, CreateErrorResponse(
			"INVALID_QUERY_PARAMS",
			"Invalid query parameters",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	tradingLogs, err := h.tradingLogService.GetUserTradingLogs(c.Request.Context(), userID, &req)
	if err != nil {
		if err.Error() == "start date cannot be after end date" {
			c.JSON(http.StatusBadRequest, CreateErrorResponse(
				"INVALID_DATE_RANGE",
				"Invalid date range",
				err.Error(),
				getTraceID(c),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"TRADING_LOGS_QUERY_FAILED",
			"Failed to query trading logs",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(tradingLogs, getTraceID(c)))
}

// GetSubAccountTradingLogs retrieves trading logs for a specific sub-account
// @Summary Get sub-account trading logs
// @Description Retrieves trading log history for a specific sub-account (must belong to authenticated user)
// @Tags TradingLogs
// @Produce json
// @Security BearerAuth
// @Param sub_account_id path string true "Sub-account ID"
// @Param type query string false "Filter by log type"
// @Param source query string false "Filter by source" Enums(manual, bot)
// @Param start_date query string false "Start date (RFC3339 format)"
// @Param end_date query string false "End date (RFC3339 format)"
// @Param limit query int false "Number of logs to return" default(100)
// @Param offset query int false "Number of logs to skip" default(0)
// @Success 200 {object} services.TradingLogQueryResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /trading-logs/sub-account/{sub_account_id} [get]
func (h *TradingLogHandler) GetSubAccountTradingLogs(c *gin.Context) {
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

	var req services.TradingLogQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, CreateErrorResponse(
			"INVALID_QUERY_PARAMS",
			"Invalid query parameters",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	tradingLogs, err := h.tradingLogService.GetSubAccountTradingLogs(c.Request.Context(), userID, subAccountID, &req)
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
		if err.Error() == "start date cannot be after end date" {
			c.JSON(http.StatusBadRequest, CreateErrorResponse(
				"INVALID_DATE_RANGE",
				"Invalid date range",
				err.Error(),
				getTraceID(c),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"TRADING_LOGS_QUERY_FAILED",
			"Failed to query sub-account trading logs",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(tradingLogs, getTraceID(c)))
}

// GetTradingLogs retrieves trading logs for a specific trading
// @Summary Get trading logs
// @Description Retrieves trading log history for a specific trading (must belong to authenticated user)
// @Tags TradingLogs
// @Produce json
// @Security BearerAuth
// @Param trading_id path string true "Trading ID"
// @Param type query string false "Filter by log type"
// @Param source query string false "Filter by source" Enums(manual, bot)
// @Param start_date query string false "Start date (RFC3339 format)"
// @Param end_date query string false "End date (RFC3339 format)"
// @Param limit query int false "Number of logs to return" default(100)
// @Param offset query int false "Number of logs to skip" default(0)
// @Success 200 {object} services.TradingLogQueryResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /trading-logs/trading/{trading_id} [get]
func (h *TradingLogHandler) GetTradingLogs(c *gin.Context) {
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

	tradingIDStr := c.Param("trading_id")
	tradingID, err := uuid.Parse(tradingIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, CreateErrorResponse(
			"INVALID_TRADING_ID",
			"Invalid trading ID format",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	var req services.TradingLogQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, CreateErrorResponse(
			"INVALID_QUERY_PARAMS",
			"Invalid query parameters",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	tradingLogs, err := h.tradingLogService.GetTradingLogs(c.Request.Context(), userID, tradingID, &req)
	if err != nil {
		if err.Error() == "trading not found" {
			c.JSON(http.StatusNotFound, CreateErrorResponse(
				"TRADING_NOT_FOUND",
				"Trading not found",
				err.Error(),
				getTraceID(c),
			))
			return
		}
		if err.Error() == "start date cannot be after end date" {
			c.JSON(http.StatusBadRequest, CreateErrorResponse(
				"INVALID_DATE_RANGE",
				"Invalid date range",
				err.Error(),
				getTraceID(c),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"TRADING_LOGS_QUERY_FAILED",
			"Failed to query trading logs",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(tradingLogs, getTraceID(c)))
}

// GetTradingLog retrieves a specific trading log by ID
// @Summary Get trading log by ID
// @Description Retrieves a specific trading log by ID (must belong to authenticated user)
// @Tags TradingLogs
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trading Log ID"
// @Success 200 {object} services.TradingLogResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /trading-logs/{id} [get]
func (h *TradingLogHandler) GetTradingLog(c *gin.Context) {
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

	tradingLogIDStr := c.Param("id")
	tradingLogID, err := uuid.Parse(tradingLogIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, CreateErrorResponse(
			"INVALID_TRADING_LOG_ID",
			"Invalid trading log ID format",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	tradingLog, err := h.tradingLogService.GetTradingLog(c.Request.Context(), userID, tradingLogID)
	if err != nil {
		if err.Error() == "trading log not found" {
			c.JSON(http.StatusNotFound, CreateErrorResponse(
				"TRADING_LOG_NOT_FOUND",
				"Trading log not found",
				err.Error(),
				getTraceID(c),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"TRADING_LOG_GET_FAILED",
			"Failed to get trading log",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(tradingLog, getTraceID(c)))
}

// GetTradingLogsByTimeRange retrieves trading logs within a specific time range
// @Summary Get trading logs by time range
// @Description Retrieves trading logs within a specific time range for the authenticated user
// @Tags TradingLogs
// @Produce json
// @Security BearerAuth
// @Param start_time query string true "Start time (RFC3339 format)"
// @Param end_time query string true "End time (RFC3339 format)"
// @Param type query string false "Filter by log type"
// @Param source query string false "Filter by source" Enums(manual, bot)
// @Param limit query int false "Number of logs to return" default(100)
// @Param offset query int false "Number of logs to skip" default(0)
// @Success 200 {object} services.TradingLogQueryResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /trading-logs/time-range [get]
func (h *TradingLogHandler) GetTradingLogsByTimeRange(c *gin.Context) {
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

	var req services.TradingLogQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, CreateErrorResponse(
			"INVALID_QUERY_PARAMS",
			"Invalid query parameters",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	tradingLogs, err := h.tradingLogService.GetTradingLogsByTimeRange(c.Request.Context(), userID, startTime, endTime, &req)
	if err != nil {
		if err.Error() == "start time cannot be after end time" {
			c.JSON(http.StatusBadRequest, CreateErrorResponse(
				"INVALID_TIME_RANGE",
				"Invalid time range",
				err.Error(),
				getTraceID(c),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"TRADING_LOGS_QUERY_FAILED",
			"Failed to query trading logs by time range",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(tradingLogs, getTraceID(c)))
}

// DeleteTradingLog deletes a trading log
// @Summary Delete trading log
// @Description Deletes a trading log entry (must belong to authenticated user and be manual)
// @Tags TradingLogs
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trading Log ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /trading-logs/{id} [delete]
func (h *TradingLogHandler) DeleteTradingLog(c *gin.Context) {
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

	tradingLogIDStr := c.Param("id")
	tradingLogID, err := uuid.Parse(tradingLogIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, CreateErrorResponse(
			"INVALID_TRADING_LOG_ID",
			"Invalid trading log ID format",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	err = h.tradingLogService.DeleteTradingLog(c.Request.Context(), userID, tradingLogID)
	if err != nil {
		if err.Error() == "trading log not found" {
			c.JSON(http.StatusNotFound, CreateErrorResponse(
				"TRADING_LOG_NOT_FOUND",
				"Trading log not found",
				err.Error(),
				getTraceID(c),
			))
			return
		}
		if err.Error() == "cannot delete bot-generated trading logs" {
			c.JSON(http.StatusConflict, CreateErrorResponse(
				"CANNOT_DELETE_BOT_LOG",
				"Cannot delete bot-generated trading logs",
				err.Error(),
				getTraceID(c),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"TRADING_LOG_DELETE_FAILED",
			"Failed to delete trading log",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	response := map[string]interface{}{
		"message": "Trading log deleted successfully",
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(response, getTraceID(c)))
}

// ListAllTradingLogs lists all trading logs with filtering (admin only)
// @Summary List all trading logs
// @Description Lists all trading logs with filtering and pagination (admin only)
// @Tags TradingLogs
// @Produce json
// @Security BearerAuth
// @Param type query string false "Filter by log type"
// @Param source query string false "Filter by source" Enums(manual, bot)
// @Param start_date query string false "Start date (RFC3339 format)"
// @Param end_date query string false "End date (RFC3339 format)"
// @Param limit query int false "Number of logs to return" default(100)
// @Param offset query int false "Number of logs to skip" default(0)
// @Success 200 {object} services.TradingLogQueryResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /admin/trading-logs [get]
func (h *TradingLogHandler) ListAllTradingLogs(c *gin.Context) {
	var req services.TradingLogQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, CreateErrorResponse(
			"INVALID_QUERY_PARAMS",
			"Invalid query parameters",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	tradingLogs, err := h.tradingLogService.ListAllTradingLogs(c.Request.Context(), &req)
	if err != nil {
		if err.Error() == "start date cannot be after end date" {
			c.JSON(http.StatusBadRequest, CreateErrorResponse(
				"INVALID_DATE_RANGE",
				"Invalid date range",
				err.Error(),
				getTraceID(c),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"TRADING_LOGS_LIST_FAILED",
			"Failed to list trading logs",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(tradingLogs, getTraceID(c)))
}

// GetTradingLogByID retrieves trading log by ID (admin only)
// @Summary Get trading log by ID
// @Description Retrieves a trading log by ID (admin only)
// @Tags TradingLogs
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trading Log ID"
// @Success 200 {object} services.TradingLogResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /admin/trading-logs/{id} [get]
func (h *TradingLogHandler) GetTradingLogByID(c *gin.Context) {
	tradingLogIDStr := c.Param("id")
	tradingLogID, err := uuid.Parse(tradingLogIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, CreateErrorResponse(
			"INVALID_TRADING_LOG_ID",
			"Invalid trading log ID format",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	tradingLog, err := h.tradingLogService.GetTradingLogByID(c.Request.Context(), tradingLogID)
	if err != nil {
		if err.Error() == "trading log not found" {
			c.JSON(http.StatusNotFound, CreateErrorResponse(
				"TRADING_LOG_NOT_FOUND",
				"Trading log not found",
				err.Error(),
				getTraceID(c),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"TRADING_LOG_GET_FAILED",
			"Failed to get trading log",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(tradingLog, getTraceID(c)))
}
