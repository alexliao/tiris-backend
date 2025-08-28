package api

import (
	"net/http"
	"strconv"

	"tiris-backend/internal/middleware"
	"tiris-backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TradingHandler handles trading platform management endpoints
type TradingHandler struct {
	tradingService TradingServiceInterface
}

// NewTradingHandler creates a new trading handler
func NewTradingHandler(tradingService TradingServiceInterface) *TradingHandler {
	return &TradingHandler{
		tradingService: tradingService,
	}
}

// CreateTrading creates a new trading platform configuration
// @Summary Create new trading platform
// @Description Creates a new trading platform configuration for the authenticated user
// @Tags Tradings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body services.CreateTradingRequest true "Create trading platform request"
// @Success 201 {object} services.TradingResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /tradings [post]
func (h *TradingHandler) CreateTrading(c *gin.Context) {
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

	var req services.CreateTradingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, CreateErrorResponse(
			"INVALID_REQUEST",
			"Invalid request format",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	trading, err := h.tradingService.CreateTrading(c.Request.Context(), userID, &req)
	if err != nil {
		if err.Error() == "trading platform name already exists" {
			c.JSON(http.StatusConflict, CreateErrorResponse(
				"TRADING_NAME_EXISTS",
				"Trading platform name already exists",
				err.Error(),
				getTraceID(c),
			))
			return
		}
		if err.Error() == "api key already exists" {
			c.JSON(http.StatusConflict, CreateErrorResponse(
				"API_KEY_EXISTS",
				"API key already exists",
				err.Error(),
				getTraceID(c),
			))
			return
		}
		if err.Error() == "api secret already exists" {
			c.JSON(http.StatusConflict, CreateErrorResponse(
				"API_SECRET_EXISTS",
				"API secret already exists",
				err.Error(),
				getTraceID(c),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"TRADING_CREATE_FAILED",
			"Failed to create trading platform",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusCreated, CreateSuccessResponse(trading, getTraceID(c)))
}

// GetUserTradings retrieves all trading platforms for the current user
// @Summary Get user trading platforms
// @Description Retrieves all trading platform configurations for the authenticated user
// @Tags Tradings
// @Produce json
// @Security BearerAuth
// @Success 200 {array} services.TradingResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /tradings [get]
func (h *TradingHandler) GetUserTradings(c *gin.Context) {
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

	tradings, err := h.tradingService.GetUserTradings(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"TRADINGS_GET_FAILED",
			"Failed to get trading platforms",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	response := map[string]interface{}{
		"tradings": tradings,
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(response, getTraceID(c)))
}

// GetTrading retrieves a specific trading platform by ID
// @Summary Get trading platform by ID
// @Description Retrieves a specific trading platform configuration by ID (must belong to authenticated user)
// @Tags Tradings
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trading Platform ID"
// @Success 200 {object} services.TradingResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /tradings/{id} [get]
func (h *TradingHandler) GetTrading(c *gin.Context) {
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

	tradingIDStr := c.Param("id")
	tradingID, err := uuid.Parse(tradingIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, CreateErrorResponse(
			"INVALID_TRADING_ID",
			"Invalid trading platform ID format",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	trading, err := h.tradingService.GetTrading(c.Request.Context(), userID, tradingID)
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
			"TRADING_GET_FAILED",
			"Failed to get trading platform",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(trading, getTraceID(c)))
}

// UpdateTrading updates an existing trading platform
// @Summary Update trading platform
// @Description Updates an existing trading platform configuration (must belong to authenticated user)
// @Tags Tradings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trading Platform ID"
// @Param request body services.UpdateTradingRequest true "Update trading platform request"
// @Success 200 {object} services.TradingResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /tradings/{id} [put]
func (h *TradingHandler) UpdateTrading(c *gin.Context) {
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

	tradingIDStr := c.Param("id")
	tradingID, err := uuid.Parse(tradingIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, CreateErrorResponse(
			"INVALID_TRADING_ID",
			"Invalid trading platform ID format",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	var req services.UpdateTradingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, CreateErrorResponse(
			"INVALID_REQUEST",
			"Invalid request format",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	trading, err := h.tradingService.UpdateTrading(c.Request.Context(), userID, tradingID, &req)
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
		if err.Error() == "trading platform name already exists" {
			c.JSON(http.StatusConflict, CreateErrorResponse(
				"TRADING_NAME_EXISTS",
				"Trading platform name already exists",
				err.Error(),
				getTraceID(c),
			))
			return
		}
		if err.Error() == "api key already exists" {
			c.JSON(http.StatusConflict, CreateErrorResponse(
				"API_KEY_EXISTS",
				"API key already exists",
				err.Error(),
				getTraceID(c),
			))
			return
		}
		if err.Error() == "api secret already exists" {
			c.JSON(http.StatusConflict, CreateErrorResponse(
				"API_SECRET_EXISTS",
				"API secret already exists",
				err.Error(),
				getTraceID(c),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"TRADING_UPDATE_FAILED",
			"Failed to update trading platform",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(trading, getTraceID(c)))
}

// DeleteTrading deletes a trading platform
// @Summary Delete trading platform
// @Description Deletes a trading platform configuration (must belong to authenticated user)
// @Tags Tradings
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trading Platform ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /tradings/{id} [delete]
func (h *TradingHandler) DeleteTrading(c *gin.Context) {
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

	tradingIDStr := c.Param("id")
	tradingID, err := uuid.Parse(tradingIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, CreateErrorResponse(
			"INVALID_TRADING_ID",
			"Invalid trading platform ID format",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	err = h.tradingService.DeleteTrading(c.Request.Context(), userID, tradingID)
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
		if err.Error() == "cannot delete trading platform with existing sub-accounts" {
			c.JSON(http.StatusConflict, CreateErrorResponse(
				"TRADING_HAS_SUBACCOUNTS",
				"Cannot delete trading platform with existing sub-accounts",
				err.Error(),
				getTraceID(c),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"TRADING_DELETE_FAILED",
			"Failed to delete trading platform",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	response := map[string]interface{}{
		"message": "Trading platform deleted successfully",
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(response, getTraceID(c)))
}

// ListTradings lists all trading platforms (admin only)
// @Summary List all trading platforms
// @Description Lists all trading platform configurations with pagination (admin only)
// @Tags Tradings
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Number of trading platforms to return" default(100)
// @Param offset query int false "Number of trading platforms to skip" default(0)
// @Success 200 {object} PaginatedResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /admin/tradings [get]
func (h *TradingHandler) ListTradings(c *gin.Context) {
	// Parse pagination parameters
	limit := 100
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 1000 {
			limit = parsed
		}
	}

	offset := 0
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	tradings, total, err := h.tradingService.ListTradings(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"TRADINGS_LIST_FAILED",
			"Failed to list trading platforms",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	// Create pagination metadata
	hasMore := int64(offset+limit) < total
	var nextOffset *int
	if hasMore {
		next := offset + limit
		nextOffset = &next
	}

	pagination := &PaginationMetadata{
		Total:      total,
		Limit:      limit,
		Offset:     offset,
		HasMore:    hasMore,
		NextOffset: nextOffset,
	}

	response := map[string]interface{}{
		"tradings": tradings,
	}

	c.JSON(http.StatusOK, CreatePaginatedResponse(response, pagination, getTraceID(c)))
}

// GetTradingByID retrieves trading platform by ID (admin only)
// @Summary Get trading platform by ID
// @Description Retrieves a trading platform by ID (admin only)
// @Tags Tradings
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trading Platform ID"
// @Success 200 {object} services.TradingResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /admin/tradings/{id} [get]
func (h *TradingHandler) GetTradingByID(c *gin.Context) {
	tradingIDStr := c.Param("id")
	tradingID, err := uuid.Parse(tradingIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, CreateErrorResponse(
			"INVALID_TRADING_ID",
			"Invalid trading platform ID format",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	trading, err := h.tradingService.GetTradingByID(c.Request.Context(), tradingID)
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
			"TRADING_GET_FAILED",
			"Failed to get trading platform",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(trading, getTraceID(c)))
}