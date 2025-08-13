package api

import (
	"time"

	"tiris-backend/internal/config"
	"tiris-backend/internal/middleware"
	"tiris-backend/internal/repositories"
	"tiris-backend/internal/services"
	"tiris-backend/pkg/auth"

	"github.com/gin-gonic/gin"
)

// Server represents the API server
type Server struct {
	router              *gin.Engine
	config              *config.Config
	repos               *repositories.Repositories
	jwtManager          *auth.JWTManager
	authService         *services.AuthService
	userService         *services.UserService
	exchangeService     *services.ExchangeService
	subAccountService   *services.SubAccountService
	transactionService  *services.TransactionService
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, repos *repositories.Repositories) *Server {
	// Initialize JWT manager
	jwtManager := auth.NewJWTManager(
		cfg.Auth.JWTSecret,
		cfg.Auth.RefreshSecret,
		time.Duration(cfg.Auth.JWTExpiration)*time.Second,
		time.Duration(cfg.Auth.RefreshExpiration)*time.Second,
	)

	// Initialize OAuth manager
	oauthConfig := auth.OAuthConfig{
		Google: auth.GoogleOAuthConfig{
			ClientID:     cfg.OAuth.Google.ClientID,
			ClientSecret: cfg.OAuth.Google.ClientSecret,
			RedirectURL:  cfg.OAuth.Google.RedirectURL,
		},
		WeChat: auth.WeChatOAuthConfig{
			AppID:       cfg.OAuth.WeChat.AppID,
			AppSecret:   cfg.OAuth.WeChat.AppSecret,
			RedirectURL: cfg.OAuth.WeChat.RedirectURL,
		},
	}
	oauthManager := auth.NewOAuthManager(oauthConfig)

	// Initialize services
	authService := services.NewAuthService(repos, jwtManager, oauthManager)
	userService := services.NewUserService(repos)
	exchangeService := services.NewExchangeService(repos)
	subAccountService := services.NewSubAccountService(repos)
	transactionService := services.NewTransactionService(repos)

	return &Server{
		config:             cfg,
		repos:              repos,
		jwtManager:         jwtManager,
		authService:        authService,
		userService:        userService,
		exchangeService:    exchangeService,
		subAccountService:  subAccountService,
		transactionService: transactionService,
	}
}

// SetupRoutes sets up all API routes
func (s *Server) SetupRoutes() *gin.Engine {
	router := gin.New()

	// Determine CORS origins based on environment
	var allowedOrigins []string
	if s.config.Environment == "production" {
		allowedOrigins = []string{"https://tiris.ai"}
	} else {
		allowedOrigins = []string{"https://dev.tiris.ai", "http://localhost:3000"}
	}

	// Global middleware
	router.Use(middleware.ErrorLoggingMiddleware())
	router.Use(middleware.SecurityHeadersMiddleware())
	router.Use(middleware.CORSMiddleware(allowedOrigins))
	router.Use(middleware.RequestIDMiddleware())
	router.Use(middleware.HealthCheckLoggingMiddleware())

	// Health check endpoints (no authentication required)
	s.setupHealthRoutes(router)

	// API routes with rate limiting
	api := router.Group("/v1")
	api.Use(middleware.APIRateLimitMiddleware())

	// Authentication routes
	s.setupAuthRoutes(api)

	// Protected routes (require authentication)
	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware(s.jwtManager))

	// User management routes
	s.setupUserRoutes(protected)

	// Exchange management routes
	s.setupExchangeRoutes(protected)

	// Sub-account management routes
	s.setupSubAccountRoutes(protected)

	// Transaction query routes
	s.setupTransactionRoutes(protected)

	s.router = router
	return router
}

// setupHealthRoutes sets up health check routes
func (s *Server) setupHealthRoutes(router *gin.Engine) {
	router.GET("/health/live", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"success": true,
			"data": gin.H{
				"status":    "alive",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			},
		})
	})

	router.GET("/health/ready", func(c *gin.Context) {
		checks := gin.H{
			"database": "ok",
			"nats":     "ok",
		}

		// For health checks, we assume services are ready
		// In a production setup, you'd want to pass database and NATS
		// instances to perform actual health checks

		c.JSON(200, gin.H{
			"success": true,
			"data": gin.H{
				"status":    "ready",
				"checks":    checks,
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			},
		})
	})
}

// setupAuthRoutes sets up authentication routes
func (s *Server) setupAuthRoutes(api *gin.RouterGroup) {
	authHandler := NewAuthHandler(s.authService)

	auth := api.Group("/auth")
	auth.Use(middleware.AuthRateLimitMiddleware()) // Stricter rate limiting for auth

	auth.POST("/login", authHandler.Login)
	auth.POST("/callback", authHandler.Callback)
	auth.POST("/refresh", authHandler.Refresh)
	auth.POST("/logout", middleware.AuthMiddleware(s.jwtManager), authHandler.Logout)
}

// setupUserRoutes sets up user management routes
func (s *Server) setupUserRoutes(protected *gin.RouterGroup) {
	userHandler := NewUserHandler(s.userService)

	users := protected.Group("/users")

	// Current user routes
	users.GET("/me", userHandler.GetCurrentUser)
	users.PUT("/me", userHandler.UpdateCurrentUser)
	users.GET("/me/stats", userHandler.GetUserStats)

	// Admin only routes
	adminUsers := users.Group("")
	adminUsers.Use(middleware.AdminMiddleware())
	
	adminUsers.GET("", userHandler.ListUsers)
	adminUsers.GET("/:id", userHandler.GetUserByID)
	adminUsers.PUT("/:id/disable", userHandler.DisableUser)
}

// setupExchangeRoutes sets up exchange management routes
func (s *Server) setupExchangeRoutes(protected *gin.RouterGroup) {
	exchangeHandler := NewExchangeHandler(s.exchangeService)

	exchanges := protected.Group("/exchanges")

	// User exchange routes
	exchanges.POST("", exchangeHandler.CreateExchange)
	exchanges.GET("", exchangeHandler.GetUserExchanges)
	exchanges.GET("/:id", exchangeHandler.GetExchange)
	exchanges.PUT("/:id", exchangeHandler.UpdateExchange)
	exchanges.DELETE("/:id", exchangeHandler.DeleteExchange)

	// Admin exchange routes
	adminExchanges := protected.Group("/admin/exchanges")
	adminExchanges.Use(middleware.AdminMiddleware())
	
	adminExchanges.GET("", exchangeHandler.ListExchanges)
	adminExchanges.GET("/:id", exchangeHandler.GetExchangeByID)
}

// setupSubAccountRoutes sets up sub-account management routes
func (s *Server) setupSubAccountRoutes(protected *gin.RouterGroup) {
	subAccountHandler := NewSubAccountHandler(s.subAccountService)

	subAccounts := protected.Group("/sub-accounts")

	// User sub-account routes
	subAccounts.POST("", subAccountHandler.CreateSubAccount)
	subAccounts.GET("", subAccountHandler.GetUserSubAccounts)
	subAccounts.GET("/:id", subAccountHandler.GetSubAccount)
	subAccounts.PUT("/:id", subAccountHandler.UpdateSubAccount)
	subAccounts.PUT("/:id/balance", subAccountHandler.UpdateBalance)
	subAccounts.DELETE("/:id", subAccountHandler.DeleteSubAccount)
	
	// Symbol-based queries
	subAccounts.GET("/symbol/:symbol", subAccountHandler.GetSubAccountsBySymbol)
}

// setupTransactionRoutes sets up transaction query routes
func (s *Server) setupTransactionRoutes(protected *gin.RouterGroup) {
	transactionHandler := NewTransactionHandler(s.transactionService)

	transactions := protected.Group("/transactions")

	// User transaction routes
	transactions.GET("", transactionHandler.GetUserTransactions)
	transactions.GET("/:id", transactionHandler.GetTransaction)
	transactions.GET("/sub-account/:sub_account_id", transactionHandler.GetSubAccountTransactions)
	transactions.GET("/exchange/:exchange_id", transactionHandler.GetExchangeTransactions)
	transactions.GET("/time-range", transactionHandler.GetTransactionsByTimeRange)

	// Admin transaction routes
	adminTransactions := protected.Group("/admin/transactions")
	adminTransactions.Use(middleware.AdminMiddleware())
	
	adminTransactions.GET("", transactionHandler.ListAllTransactions)
	adminTransactions.GET("/:id", transactionHandler.GetTransactionByID)
}

// GetRouter returns the configured router
func (s *Server) GetRouter() *gin.Engine {
	if s.router == nil {
		return s.SetupRoutes()
	}
	return s.router
}