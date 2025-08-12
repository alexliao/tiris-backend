package api

import (
	"net/http"
	"time"

	"tiris-backend/internal/middleware"
	"tiris-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	authService *services.AuthService
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Login initiates OAuth login flow
// @Summary Initiate OAuth login
// @Description Initiates OAuth login flow and returns authorization URL
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body services.LoginRequest true "Login request"
// @Success 200 {object} services.LoginResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req services.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_REQUEST",
				Message: "Invalid request format",
				Details: err.Error(),
			},
			Metadata: ResponseMetadata{
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				TraceID:   getTraceID(c),
			},
		})
		return
	}

	response, err := h.authService.InitiateLogin(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "LOGIN_FAILED",
				Message: "Failed to initiate login",
				Details: err.Error(),
			},
			Metadata: ResponseMetadata{
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				TraceID:   getTraceID(c),
			},
		})
		return
	}

	// Store state in session/cookie for validation (in production, use secure session storage)
	c.SetCookie("oauth_state", response.State, 600, "/", "", false, true) // 10 minutes

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    response,
		Metadata: ResponseMetadata{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			TraceID:   getTraceID(c),
		},
	})
}

// Callback handles OAuth callback
// @Summary Handle OAuth callback
// @Description Handles OAuth callback and returns JWT tokens
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body services.CallbackRequest true "Callback request"
// @Success 200 {object} services.AuthResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/callback [post]
func (h *AuthHandler) Callback(c *gin.Context) {
	var req services.CallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_REQUEST",
				Message: "Invalid request format",
				Details: err.Error(),
			},
			Metadata: ResponseMetadata{
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				TraceID:   getTraceID(c),
			},
		})
		return
	}

	// Get expected state from cookie
	expectedState, err := c.Cookie("oauth_state")
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_STATE",
				Message: "Missing or invalid state parameter",
				Details: "OAuth state not found in session",
			},
			Metadata: ResponseMetadata{
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				TraceID:   getTraceID(c),
			},
		})
		return
	}

	// Clear the state cookie
	c.SetCookie("oauth_state", "", -1, "/", "", false, true)

	response, err := h.authService.HandleCallback(c.Request.Context(), &req, expectedState)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "OAUTH_CALLBACK_FAILED",
				Message: "OAuth callback failed",
				Details: err.Error(),
			},
			Metadata: ResponseMetadata{
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				TraceID:   getTraceID(c),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    response,
		Metadata: ResponseMetadata{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			TraceID:   getTraceID(c),
		},
	})
}

// Refresh refreshes access token using refresh token
// @Summary Refresh access token
// @Description Refreshes access token using refresh token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body services.RefreshRequest true "Refresh request"
// @Success 200 {object} services.AuthResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req services.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_REQUEST",
				Message: "Invalid request format",
				Details: err.Error(),
			},
			Metadata: ResponseMetadata{
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				TraceID:   getTraceID(c),
			},
		})
		return
	}

	response, err := h.authService.RefreshToken(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "TOKEN_REFRESH_FAILED",
				Message: "Failed to refresh token",
				Details: err.Error(),
			},
			Metadata: ResponseMetadata{
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				TraceID:   getTraceID(c),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    response,
		Metadata: ResponseMetadata{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			TraceID:   getTraceID(c),
		},
	})
}

// Logout logs out the current user
// @Summary Logout user
// @Description Logs out the current user and invalidates session
// @Tags Authentication
// @Produce json
// @Security BearerAuth
// @Success 200 {object} SuccessResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "AUTH_REQUIRED",
				Message: "Authentication required",
			},
			Metadata: ResponseMetadata{
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				TraceID:   getTraceID(c),
			},
		})
		return
	}

	err := h.authService.Logout(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "LOGOUT_FAILED",
				Message: "Failed to logout",
				Details: err.Error(),
			},
			Metadata: ResponseMetadata{
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				TraceID:   getTraceID(c),
			},
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data: gin.H{
			"message": "Logged out successfully",
		},
		Metadata: ResponseMetadata{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			TraceID:   getTraceID(c),
		},
	})
}

// getTraceID extracts trace ID from context
func getTraceID(c *gin.Context) string {
	if traceID, exists := c.Get("request_id"); exists {
		if id, ok := traceID.(string); ok {
			return id
		}
	}
	return ""
}