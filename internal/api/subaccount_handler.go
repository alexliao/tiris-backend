package api

import (
	"net/http"

	"tiris-backend/internal/middleware"
	"tiris-backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SubAccountHandler handles sub-account management endpoints
type SubAccountHandler struct {
	subAccountService SubAccountServiceInterface
}

// NewSubAccountHandler creates a new sub-account handler
func NewSubAccountHandler(subAccountService SubAccountServiceInterface) *SubAccountHandler {
	return &SubAccountHandler{
		subAccountService: subAccountService,
	}
}

// CreateSubAccount creates a new sub-account
// @Summary Create new sub-account
// @Description Creates a new sub-account for the authenticated user
// @Tags SubAccounts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body services.CreateSubAccountRequest true "Create sub-account request"
// @Success 201 {object} services.SubAccountResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /sub-accounts [post]
func (h *SubAccountHandler) CreateSubAccount(c *gin.Context) {
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

	var req services.CreateSubAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, CreateErrorResponse(
			"INVALID_REQUEST",
			"Invalid request format",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	subAccount, err := h.subAccountService.CreateSubAccount(c.Request.Context(), userID, &req)
	if err != nil {
		if err.Error() == "trading platform not found" {
			c.JSON(http.StatusNotFound, CreateErrorResponse(
				"TRADING_NOT_FOUND",
				"Trading platform not found",
				err.Error(),
				getTraceID(c),
			))
			return
		}
		if err.Error() == "sub-account name already exists for this trading platform" {
			c.JSON(http.StatusConflict, CreateErrorResponse(
				"SUBACCOUNT_NAME_EXISTS",
				"Sub-account name already exists for this trading platform",
				err.Error(),
				getTraceID(c),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"SUBACCOUNT_CREATE_FAILED",
			"Failed to create sub-account",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusCreated, CreateSuccessResponse(subAccount, getTraceID(c)))
}

// GetUserSubAccounts retrieves all sub-accounts for the current user
// @Summary Get user sub-accounts
// @Description Retrieves all sub-account configurations for the authenticated user, optionally filtered by trading platform
// @Tags SubAccounts
// @Produce json
// @Security BearerAuth
// @Param trading_id query string false "Filter by trading platform ID"
// @Success 200 {array} services.SubAccountResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /sub-accounts [get]
func (h *SubAccountHandler) GetUserSubAccounts(c *gin.Context) {
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

	// Parse optional trading_id filter
	var tradingID *uuid.UUID
	if tradingIDStr := c.Query("trading_id"); tradingIDStr != "" {
		if parsed, err := uuid.Parse(tradingIDStr); err == nil {
			tradingID = &parsed
		} else {
			c.JSON(http.StatusBadRequest, CreateErrorResponse(
				"INVALID_TRADING_ID",
				"Invalid trading platform ID format",
				err.Error(),
				getTraceID(c),
			))
			return
		}
	}

	subAccounts, err := h.subAccountService.GetUserSubAccounts(c.Request.Context(), userID, tradingID)
	if err != nil {
		if err.Error() == "trading platform not found" {
			c.JSON(http.StatusNotFound, CreateErrorResponse(
				"TRADING_NOT_FOUND",
				"Trading platform not found",
				err.Error(),
				getTraceID(c),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"SUBACCOUNTS_GET_FAILED",
			"Failed to get sub-accounts",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	response := map[string]interface{}{
		"sub_accounts": subAccounts,
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(response, getTraceID(c)))
}

// GetSubAccount retrieves a specific sub-account by ID
// @Summary Get sub-account by ID
// @Description Retrieves a specific sub-account configuration by ID (must belong to authenticated user)
// @Tags SubAccounts
// @Produce json
// @Security BearerAuth
// @Param id path string true "Sub-account ID"
// @Success 200 {object} services.SubAccountResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /sub-accounts/{id} [get]
func (h *SubAccountHandler) GetSubAccount(c *gin.Context) {
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

	subAccountIDStr := c.Param("id")
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

	subAccount, err := h.subAccountService.GetSubAccount(c.Request.Context(), userID, subAccountID)
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

		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"SUBACCOUNT_GET_FAILED",
			"Failed to get sub-account",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(subAccount, getTraceID(c)))
}

// UpdateSubAccount updates an existing sub-account
// @Summary Update sub-account
// @Description Updates an existing sub-account configuration (must belong to authenticated user)
// @Tags SubAccounts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Sub-account ID"
// @Param request body services.UpdateSubAccountRequest true "Update sub-account request"
// @Success 200 {object} services.SubAccountResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /sub-accounts/{id} [put]
func (h *SubAccountHandler) UpdateSubAccount(c *gin.Context) {
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

	subAccountIDStr := c.Param("id")
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

	var req services.UpdateSubAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, CreateErrorResponse(
			"INVALID_REQUEST",
			"Invalid request format",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	subAccount, err := h.subAccountService.UpdateSubAccount(c.Request.Context(), userID, subAccountID, &req)
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
		if err.Error() == "sub-account name already exists for this exchange" {
			c.JSON(http.StatusConflict, CreateErrorResponse(
				"SUBACCOUNT_NAME_EXISTS",
				"Sub-account name already exists for this exchange",
				err.Error(),
				getTraceID(c),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"SUBACCOUNT_UPDATE_FAILED",
			"Failed to update sub-account",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(subAccount, getTraceID(c)))
}

// UpdateBalance updates sub-account balance
// @Summary Update sub-account balance
// @Description Updates sub-account balance with proper logging (must belong to authenticated user)
// @Tags SubAccounts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Sub-account ID"
// @Param request body services.UpdateBalanceRequest true "Update balance request"
// @Success 200 {object} services.SubAccountResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /sub-accounts/{id}/balance [put]
func (h *SubAccountHandler) UpdateBalance(c *gin.Context) {
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

	subAccountIDStr := c.Param("id")
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

	var req services.UpdateBalanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, CreateErrorResponse(
			"INVALID_REQUEST",
			"Invalid request format",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	subAccount, err := h.subAccountService.UpdateBalance(c.Request.Context(), userID, subAccountID, &req)
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
		if err.Error() == "insufficient balance" {
			c.JSON(http.StatusBadRequest, CreateErrorResponse(
				"INSUFFICIENT_BALANCE",
				"Insufficient balance",
				err.Error(),
				getTraceID(c),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"BALANCE_UPDATE_FAILED",
			"Failed to update balance",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(subAccount, getTraceID(c)))
}

// DeleteSubAccount deletes a sub-account
// @Summary Delete sub-account
// @Description Deletes a sub-account configuration (must belong to authenticated user)
// @Tags SubAccounts
// @Produce json
// @Security BearerAuth
// @Param id path string true "Sub-account ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /sub-accounts/{id} [delete]
func (h *SubAccountHandler) DeleteSubAccount(c *gin.Context) {
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

	subAccountIDStr := c.Param("id")
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

	err = h.subAccountService.DeleteSubAccount(c.Request.Context(), userID, subAccountID)
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
		if err.Error() == "cannot delete sub-account with positive balance" {
			c.JSON(http.StatusConflict, CreateErrorResponse(
				"SUBACCOUNT_HAS_BALANCE",
				"Cannot delete sub-account with positive balance",
				err.Error(),
				getTraceID(c),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"SUBACCOUNT_DELETE_FAILED",
			"Failed to delete sub-account",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	response := map[string]interface{}{
		"message": "Sub-account deleted successfully",
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(response, getTraceID(c)))
}

// GetSubAccountsBySymbol retrieves sub-accounts by symbol
// @Summary Get sub-accounts by symbol
// @Description Retrieves all sub-accounts for a specific trading symbol (must belong to authenticated user)
// @Tags SubAccounts
// @Produce json
// @Security BearerAuth
// @Param symbol path string true "Trading symbol"
// @Success 200 {array} services.SubAccountResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /sub-accounts/symbol/{symbol} [get]
func (h *SubAccountHandler) GetSubAccountsBySymbol(c *gin.Context) {
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

	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, CreateErrorResponse(
			"INVALID_SYMBOL",
			"Symbol is required",
			"",
			getTraceID(c),
		))
		return
	}

	subAccounts, err := h.subAccountService.GetSubAccountsBySymbol(c.Request.Context(), userID, symbol)
	if err != nil {
		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"SUBACCOUNTS_GET_FAILED",
			"Failed to get sub-accounts by symbol",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	response := map[string]interface{}{
		"sub_accounts": subAccounts,
		"symbol":       symbol,
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(response, getTraceID(c)))
}
