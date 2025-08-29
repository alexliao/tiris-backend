package api

import (
	"net/http"
	"strconv"
	"strings"

	"tiris-backend/internal/middleware"
	"tiris-backend/internal/services"
	"tiris-backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Helper functions for error checking
func isConflictError(err error) bool {
	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "already exists") || 
		   strings.Contains(errMsg, "already in use") ||
		   strings.Contains(errMsg, "duplicate") ||
		   strings.Contains(errMsg, "conflict")
}

func isNotFoundError(err error) bool {
	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "not found") ||
		   strings.Contains(errMsg, "does not exist")
}

// ExchangeBindingHandler handles exchange binding management endpoints
type ExchangeBindingHandler struct {
	exchangeBindingService services.ExchangeBindingService
}

// NewExchangeBindingHandler creates a new exchange binding handler
func NewExchangeBindingHandler(exchangeBindingService services.ExchangeBindingService) *ExchangeBindingHandler {
	return &ExchangeBindingHandler{
		exchangeBindingService: exchangeBindingService,
	}
}

// CreateExchangeBinding creates a new exchange binding
// @Summary Create new exchange binding
// @Description Creates a new exchange binding for the authenticated user
// @Tags Exchange Bindings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.CreateExchangeBindingRequest true "Create exchange binding request"
// @Success 201 {object} models.ExchangeBinding
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /exchange-bindings [post]
func (h *ExchangeBindingHandler) CreateExchangeBinding(c *gin.Context) {
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

	var request models.CreateExchangeBindingRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, CreateErrorResponse(
			"INVALID_REQUEST",
			"Invalid request format",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	// Set the user ID for private bindings
	if request.Type == "private" {
		request.UserID = &userID
	}

	binding, err := h.exchangeBindingService.CreateExchangeBinding(c.Request.Context(), &request)
	if err != nil {
		if isConflictError(err) {
			errorCode := "EXCHANGE_BINDING_EXISTS"
			errorMessage := "Exchange binding already exists"
			
			// Check for specific API credential errors
			errMsg := strings.ToLower(err.Error())
			if strings.Contains(errMsg, "api key already in use") {
				errorCode = "API_KEY_EXISTS"
				errorMessage = "API key already in use"
			} else if strings.Contains(errMsg, "api secret already in use") {
				errorCode = "API_SECRET_EXISTS" 
				errorMessage = "API secret already in use"
			}
			
			c.JSON(http.StatusConflict, CreateErrorResponse(
				errorCode,
				errorMessage,
				err.Error(),
				getTraceID(c),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"INTERNAL_ERROR",
			"Failed to create exchange binding",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusCreated, CreateSuccessResponse(binding, getTraceID(c)))
}

// GetUserExchangeBindings retrieves all exchange bindings for the authenticated user
// @Summary Get user exchange bindings
// @Description Retrieves all exchange bindings for the authenticated user with pagination
// @Tags Exchange Bindings
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Success 200 {object} SuccessResponse{data=[]models.ExchangeBinding}
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /exchange-bindings [get]
func (h *ExchangeBindingHandler) GetUserExchangeBindings(c *gin.Context) {
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

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	params := models.PaginationParams{
		Page:  page,
		Limit: limit,
	}

	bindings, pagination, err := h.exchangeBindingService.GetUserExchangeBindings(c.Request.Context(), userID, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"INTERNAL_ERROR",
			"Failed to retrieve exchange bindings",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	response := map[string]interface{}{
		"items":      bindings,
		"pagination": pagination,
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(response, getTraceID(c)))
}

// GetExchangeBinding retrieves a specific exchange binding by ID
// @Summary Get exchange binding by ID
// @Description Retrieves a specific exchange binding by ID (must be owned by user or be public)
// @Tags Exchange Bindings
// @Produce json
// @Security BearerAuth
// @Param id path string true "Exchange binding ID"
// @Success 200 {object} SuccessResponse{data=models.ExchangeBinding}
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /exchange-bindings/{id} [get]
func (h *ExchangeBindingHandler) GetExchangeBinding(c *gin.Context) {
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

	bindingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, CreateErrorResponse(
			"INVALID_ID",
			"Invalid exchange binding ID",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	// Validate access
	hasAccess, err := h.exchangeBindingService.ValidateExchangeBindingAccess(c.Request.Context(), userID, bindingID)
	if err != nil {
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, CreateErrorResponse(
				"EXCHANGE_BINDING_NOT_FOUND",
				"Exchange binding not found",
				err.Error(),
				getTraceID(c),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"INTERNAL_ERROR",
			"Failed to validate access",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	if !hasAccess {
		c.JSON(http.StatusForbidden, CreateErrorResponse(
			"ACCESS_DENIED",
			"Access denied to this exchange binding",
			"",
			getTraceID(c),
		))
		return
	}

	binding, err := h.exchangeBindingService.GetExchangeBinding(c.Request.Context(), bindingID)
	if err != nil {
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, CreateErrorResponse(
				"EXCHANGE_BINDING_NOT_FOUND",
				"Exchange binding not found",
				err.Error(),
				getTraceID(c),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"INTERNAL_ERROR",
			"Failed to retrieve exchange binding",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(binding, getTraceID(c)))
}

// UpdateExchangeBinding updates an existing exchange binding
// @Summary Update exchange binding
// @Description Updates an existing exchange binding (must be owned by user)
// @Tags Exchange Bindings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Exchange binding ID"
// @Param request body models.UpdateExchangeBindingRequest true "Update exchange binding request"
// @Success 200 {object} SuccessResponse{data=models.ExchangeBinding}
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /exchange-bindings/{id} [put]
func (h *ExchangeBindingHandler) UpdateExchangeBinding(c *gin.Context) {
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

	bindingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, CreateErrorResponse(
			"INVALID_ID",
			"Invalid exchange binding ID",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	var request models.UpdateExchangeBindingRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, CreateErrorResponse(
			"INVALID_REQUEST",
			"Invalid request format",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	// Validate that user owns this binding (only owners can update)
	hasAccess, err := h.exchangeBindingService.ValidateExchangeBindingAccess(c.Request.Context(), userID, bindingID)
	if err != nil {
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, CreateErrorResponse(
				"EXCHANGE_BINDING_NOT_FOUND",
				"Exchange binding not found",
				err.Error(),
				getTraceID(c),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"INTERNAL_ERROR",
			"Failed to validate access",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	if !hasAccess {
		c.JSON(http.StatusForbidden, CreateErrorResponse(
			"ACCESS_DENIED",
			"Access denied to this exchange binding",
			"",
			getTraceID(c),
		))
		return
	}

	binding, err := h.exchangeBindingService.UpdateExchangeBinding(c.Request.Context(), bindingID, &request)
	if err != nil {
		if isConflictError(err) {
			c.JSON(http.StatusConflict, CreateErrorResponse(
				"EXCHANGE_BINDING_EXISTS",
				"Exchange binding name already exists",
				err.Error(),
				getTraceID(c),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"INTERNAL_ERROR",
			"Failed to update exchange binding",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(binding, getTraceID(c)))
}

// DeleteExchangeBinding deletes an exchange binding
// @Summary Delete exchange binding
// @Description Deletes an exchange binding (must be owned by user and not in use)
// @Tags Exchange Bindings
// @Produce json
// @Security BearerAuth
// @Param id path string true "Exchange binding ID"
// @Success 200 {object} SuccessResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /exchange-bindings/{id} [delete]
func (h *ExchangeBindingHandler) DeleteExchangeBinding(c *gin.Context) {
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

	bindingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, CreateErrorResponse(
			"INVALID_ID",
			"Invalid exchange binding ID",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	// Validate that user owns this binding (only owners can delete)
	hasAccess, err := h.exchangeBindingService.ValidateExchangeBindingAccess(c.Request.Context(), userID, bindingID)
	if err != nil {
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, CreateErrorResponse(
				"EXCHANGE_BINDING_NOT_FOUND",
				"Exchange binding not found",
				err.Error(),
				getTraceID(c),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"INTERNAL_ERROR",
			"Failed to validate access",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	if !hasAccess {
		c.JSON(http.StatusForbidden, CreateErrorResponse(
			"ACCESS_DENIED",
			"Access denied to this exchange binding",
			"",
			getTraceID(c),
		))
		return
	}

	err = h.exchangeBindingService.DeleteExchangeBinding(c.Request.Context(), bindingID)
	if err != nil {
		if isConflictError(err) {
			c.JSON(http.StatusConflict, CreateErrorResponse(
				"EXCHANGE_BINDING_IN_USE",
				"Cannot delete exchange binding that is currently in use",
				err.Error(),
				getTraceID(c),
			))
			return
		}

		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"INTERNAL_ERROR",
			"Failed to delete exchange binding",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(map[string]string{
		"message": "Exchange binding deleted successfully",
	}, getTraceID(c)))
}

// GetPublicExchangeBindings retrieves public exchange bindings
// @Summary Get public exchange bindings
// @Description Retrieves all public exchange bindings (optionally filtered by exchange)
// @Tags Exchange Bindings
// @Produce json
// @Security BearerAuth
// @Param exchange query string false "Filter by exchange type"
// @Success 200 {object} SuccessResponse{data=[]models.ExchangeBinding}
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /exchange-bindings/public [get]
func (h *ExchangeBindingHandler) GetPublicExchangeBindings(c *gin.Context) {
	exchange := c.Query("exchange")

	bindings, err := h.exchangeBindingService.GetPublicExchangeBindings(c.Request.Context(), exchange)
	if err != nil {
		c.JSON(http.StatusInternalServerError, CreateErrorResponse(
			"INTERNAL_ERROR",
			"Failed to retrieve public exchange bindings",
			err.Error(),
			getTraceID(c),
		))
		return
	}

	c.JSON(http.StatusOK, CreateSuccessResponse(bindings, getTraceID(c)))
}