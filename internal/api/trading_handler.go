package api

import (
	"net/http"
	"strconv"

	"tiris-backend/internal/middleware"
	"tiris-backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TradingHandler handles trading management endpoints
type TradingHandler struct {
	tradingService TradingServiceInterface
}

// NewTradingHandler creates a new trading handler
func NewTradingHandler(tradingService TradingServiceInterface) *TradingHandler {
	return &TradingHandler{
		tradingService: tradingService,
	}
}

// CreateTrading creates a new trading configuration
// @Summary Create new trading
// @Description Creates a new trading configuration for the authenticated user
// @Tags Tradings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body services.CreateTradingRequest true "Create trading request"
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
		if err.Error() == "trading name already exists" {
			c.JSON(http.StatusConflict, CreateErrorResponse(
				"TRADING_NAME_EXISTS",
				"Trading name already exists",
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
			"Failed to create trading",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusCreated, CreateSuccessResponse(trading, getTraceID(c)))
}

// GetUserTradings retrieves all tradings for the current user
// @Summary Get user tradings
// @Description Retrieves all trading configurations for the authenticated user
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
			"Failed to get tradings",
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

// GetTrading retrieves a specific trading by ID
// @Summary Get trading by ID
// @Description Retrieves a specific trading configuration by ID (must belong to authenticated user)
// @Tags Tradings
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trading ID"
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
			"Invalid trading ID format",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	trading, err := h.tradingService.GetTrading(c.Request.Context(), userID, tradingID)
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

		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"TRADING_GET_FAILED",
			"Failed to get trading",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(trading, getTraceID(c)))
}

// UpdateTrading updates an existing trading
// @Summary Update trading
// @Description Updates an existing trading configuration (must belong to authenticated user)
// @Tags Tradings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trading ID"
// @Param request body services.UpdateTradingRequest true "Update trading request"
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
			"Invalid trading ID format",
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
		if err.Error() == "trading not found" {
			c.JSON(http.StatusNotFound, CreateErrorResponse(
				"TRADING_NOT_FOUND",
				"Trading not found",
				err.Error(),
				getTraceID(c),
			))
			return
		}
		if err.Error() == "trading name already exists" {
			c.JSON(http.StatusConflict, CreateErrorResponse(
				"TRADING_NAME_EXISTS",
				"Trading name already exists",
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
			"Failed to update trading",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(trading, getTraceID(c)))
}

// DeleteTrading deletes a trading
// @Summary Delete trading
// @Description Deletes a trading configuration (must belong to authenticated user)
// @Tags Tradings
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trading ID"
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
			"Invalid trading ID format",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	err = h.tradingService.DeleteTrading(c.Request.Context(), userID, tradingID)
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
		if err.Error() == "cannot delete trading with existing sub-accounts" {
			c.JSON(http.StatusConflict, CreateErrorResponse(
				"TRADING_HAS_SUBACCOUNTS",
				"Cannot delete trading with existing sub-accounts",
				err.Error(),
				getTraceID(c),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"TRADING_DELETE_FAILED",
			"Failed to delete trading",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	response := map[string]interface{}{
		"message": "Trading deleted successfully",
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(response, getTraceID(c)))
}

// ListTradings lists all tradings (admin only)
// @Summary List all tradings
// @Description Lists all trading configurations with pagination (admin only)
// @Tags Tradings
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Number of tradings to return" default(100)
// @Param offset query int false "Number of tradings to skip" default(0)
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
			"Failed to list tradings",
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

// GetTradingByID retrieves trading by ID (admin only)
// @Summary Get trading by ID
// @Description Retrieves a trading by ID (admin only)
// @Tags Tradings
// @Produce json
// @Security BearerAuth
// @Param id path string true "Trading ID"
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
			"Invalid trading ID format",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	trading, err := h.tradingService.GetTradingByID(c.Request.Context(), tradingID)
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

		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"TRADING_GET_FAILED",
			"Failed to get trading",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(trading, getTraceID(c)))
}