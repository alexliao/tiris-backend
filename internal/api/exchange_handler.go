package api

import (
	"net/http"
	"strconv"

	"tiris-backend/internal/middleware"
	"tiris-backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ExchangeHandler handles exchange management endpoints
type ExchangeHandler struct {
	exchangeService *services.ExchangeService
}

// NewExchangeHandler creates a new exchange handler
func NewExchangeHandler(exchangeService *services.ExchangeService) *ExchangeHandler {
	return &ExchangeHandler{
		exchangeService: exchangeService,
	}
}

// CreateExchange creates a new exchange configuration
// @Summary Create new exchange
// @Description Creates a new exchange configuration for the authenticated user
// @Tags Exchanges
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body services.CreateExchangeRequest true "Create exchange request"
// @Success 201 {object} services.ExchangeResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /exchanges [post]
func (h *ExchangeHandler) CreateExchange(c *gin.Context) {
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

	var req services.CreateExchangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, CreateErrorResponse(
			"INVALID_REQUEST",
			"Invalid request format",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	exchange, err := h.exchangeService.CreateExchange(c.Request.Context(), userID, &req)
	if err != nil {
		if err.Error() == "exchange name already exists" {
			c.JSON(http.StatusConflict, CreateErrorResponse(
				"EXCHANGE_NAME_EXISTS",
				"Exchange name already exists",
				err.Error(),
				getTraceID(c),
			))
			return
		}
		if err.Error() == "maximum number of exchanges reached (10)" {
			c.JSON(http.StatusBadRequest, CreateErrorResponse(
				"EXCHANGE_LIMIT_REACHED",
				"Maximum number of exchanges reached",
				err.Error(),
				getTraceID(c),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"EXCHANGE_CREATE_FAILED",
			"Failed to create exchange",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusCreated, CreateSuccessResponse(exchange, getTraceID(c)))
}

// GetUserExchanges retrieves all exchanges for the current user
// @Summary Get user exchanges
// @Description Retrieves all exchange configurations for the authenticated user
// @Tags Exchanges
// @Produce json
// @Security BearerAuth
// @Success 200 {array} services.ExchangeResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /exchanges [get]
func (h *ExchangeHandler) GetUserExchanges(c *gin.Context) {
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

	exchanges, err := h.exchangeService.GetUserExchanges(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"EXCHANGES_GET_FAILED",
			"Failed to get exchanges",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	response := map[string]interface{}{
		"exchanges": exchanges,
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(response, getTraceID(c)))
}

// GetExchange retrieves a specific exchange by ID
// @Summary Get exchange by ID
// @Description Retrieves a specific exchange configuration by ID (must belong to authenticated user)
// @Tags Exchanges
// @Produce json
// @Security BearerAuth
// @Param id path string true "Exchange ID"
// @Success 200 {object} services.ExchangeResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /exchanges/{id} [get]
func (h *ExchangeHandler) GetExchange(c *gin.Context) {
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

	exchangeIDStr := c.Param("id")
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

	exchange, err := h.exchangeService.GetExchange(c.Request.Context(), userID, exchangeID)
	if err != nil {
		if err.Error() == "exchange not found" {
			c.JSON(http.StatusNotFound, CreateErrorResponse(
				"EXCHANGE_NOT_FOUND",
				"Exchange not found",
				err.Error(),
				getTraceID(c),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"EXCHANGE_GET_FAILED",
			"Failed to get exchange",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(exchange, getTraceID(c)))
}

// UpdateExchange updates an existing exchange
// @Summary Update exchange
// @Description Updates an existing exchange configuration (must belong to authenticated user)
// @Tags Exchanges
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Exchange ID"
// @Param request body services.UpdateExchangeRequest true "Update exchange request"
// @Success 200 {object} services.ExchangeResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /exchanges/{id} [put]
func (h *ExchangeHandler) UpdateExchange(c *gin.Context) {
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

	exchangeIDStr := c.Param("id")
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

	var req services.UpdateExchangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, CreateErrorResponse(
			"INVALID_REQUEST",
			"Invalid request format",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	exchange, err := h.exchangeService.UpdateExchange(c.Request.Context(), userID, exchangeID, &req)
	if err != nil {
		if err.Error() == "exchange not found" {
			c.JSON(http.StatusNotFound, CreateErrorResponse(
				"EXCHANGE_NOT_FOUND",
				"Exchange not found",
				err.Error(),
				getTraceID(c),
			))
			return
		}
		if err.Error() == "exchange name already exists" {
			c.JSON(http.StatusConflict, CreateErrorResponse(
				"EXCHANGE_NAME_EXISTS",
				"Exchange name already exists",
				err.Error(),
				getTraceID(c),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"EXCHANGE_UPDATE_FAILED",
			"Failed to update exchange",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(exchange, getTraceID(c)))
}

// DeleteExchange deletes an exchange
// @Summary Delete exchange
// @Description Deletes an exchange configuration (must belong to authenticated user)
// @Tags Exchanges
// @Produce json
// @Security BearerAuth
// @Param id path string true "Exchange ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /exchanges/{id} [delete]
func (h *ExchangeHandler) DeleteExchange(c *gin.Context) {
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

	exchangeIDStr := c.Param("id")
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

	err = h.exchangeService.DeleteExchange(c.Request.Context(), userID, exchangeID)
	if err != nil {
		if err.Error() == "exchange not found" {
			c.JSON(http.StatusNotFound, CreateErrorResponse(
				"EXCHANGE_NOT_FOUND",
				"Exchange not found",
				err.Error(),
				getTraceID(c),
			))
			return
		}
		if err.Error() == "cannot delete exchange with existing sub-accounts" {
			c.JSON(http.StatusConflict, CreateErrorResponse(
				"EXCHANGE_HAS_SUBACCOUNTS",
				"Cannot delete exchange with existing sub-accounts",
				err.Error(),
				getTraceID(c),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"EXCHANGE_DELETE_FAILED",
			"Failed to delete exchange",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	response := map[string]interface{}{
		"message": "Exchange deleted successfully",
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(response, getTraceID(c)))
}

// ListExchanges lists all exchanges (admin only)
// @Summary List all exchanges
// @Description Lists all exchange configurations with pagination (admin only)
// @Tags Exchanges
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Number of exchanges to return" default(100)
// @Param offset query int false "Number of exchanges to skip" default(0)
// @Success 200 {object} PaginatedResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /admin/exchanges [get]
func (h *ExchangeHandler) ListExchanges(c *gin.Context) {
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

	exchanges, total, err := h.exchangeService.ListExchanges(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"EXCHANGES_LIST_FAILED",
			"Failed to list exchanges",
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
		"exchanges": exchanges,
	}

	c.JSON(http.StatusOK, CreatePaginatedResponse(response, pagination, getTraceID(c)))
}

// GetExchangeByID retrieves exchange by ID (admin only)
// @Summary Get exchange by ID
// @Description Retrieves an exchange by ID (admin only)
// @Tags Exchanges
// @Produce json
// @Security BearerAuth
// @Param id path string true "Exchange ID"
// @Success 200 {object} services.ExchangeResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /admin/exchanges/{id} [get]
func (h *ExchangeHandler) GetExchangeByID(c *gin.Context) {
	exchangeIDStr := c.Param("id")
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

	exchange, err := h.exchangeService.GetExchangeByID(c.Request.Context(), exchangeID)
	if err != nil {
		if err.Error() == "exchange not found" {
			c.JSON(http.StatusNotFound, CreateErrorResponse(
				"EXCHANGE_NOT_FOUND",
				"Exchange not found",
				err.Error(),
				getTraceID(c),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"EXCHANGE_GET_FAILED",
			"Failed to get exchange",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(exchange, getTraceID(c)))
}
