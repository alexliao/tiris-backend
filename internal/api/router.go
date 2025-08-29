package api

import (
	"time"

	_ "tiris-backend/docs"
	"tiris-backend/internal/config"
	"tiris-backend/internal/database"
	"tiris-backend/internal/metrics"
	"tiris-backend/internal/middleware"
	"tiris-backend/internal/nats"
	"tiris-backend/internal/repositories"
	"tiris-backend/internal/services"
	"tiris-backend/pkg/auth"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Server represents the API server
type Server struct {
	router             *gin.Engine
	config             *config.Config
	db                 *database.DB
	natsManager        *nats.Manager
	repos              *repositories.Repositories
	jwtManager         *auth.JWTManager
	authService          *services.AuthService
	userService          *services.UserService
	exchangeBindingService services.ExchangeBindingService
	tradingService       *services.TradingService
	subAccountService    *services.SubAccountService
	transactionService   *services.TransactionService
	tradingLogService    *services.TradingLogService
	metrics              *metrics.Metrics
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, repos *repositories.Repositories, db *database.DB, natsManager *nats.Manager) *Server {
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

	// Initialize metrics
	metricsInstance := metrics.NewMetrics()

	// Initialize services
	authService := services.NewAuthService(repos, jwtManager, oauthManager)
	userService := services.NewUserService(repos)
	exchangeBindingService := services.NewExchangeBindingService(repos.ExchangeBinding)
	tradingService := services.NewTradingService(repos, exchangeBindingService)
	subAccountService := services.NewSubAccountService(repos)
	transactionService := services.NewTransactionService(repos)
	tradingLogService := services.NewTradingLogService(repos, db.DB)

	return &Server{
		config:               cfg,
		db:                   db,
		natsManager:          natsManager,
		repos:                repos,
		jwtManager:           jwtManager,
		authService:          authService,
		userService:          userService,
		exchangeBindingService: exchangeBindingService,
		tradingService:       tradingService,
		subAccountService:    subAccountService,
		transactionService:   transactionService,
		tradingLogService:    tradingLogService,
		metrics:              metricsInstance,
	}
}

// SetupRoutes sets up all API routes
func (s *Server) SetupRoutes() *gin.Engine {
	router := gin.New()

	// Determine CORS origins based on environment
	var allowedOrigins []string
	if s.config.Environment == "production" {
		allowedOrigins = []string{"https://backend.tiris.ai"}
	} else {
		allowedOrigins = []string{"https://backend.dev.tiris.ai", "http://localhost:3000"}
	}

	// Global middleware
	router.Use(middleware.ErrorLoggingMiddleware())
	router.Use(middleware.SecurityHeadersMiddleware())
	router.Use(middleware.CORSMiddleware(allowedOrigins))
	router.Use(middleware.RequestIDMiddleware())
	router.Use(middleware.HealthCheckLoggingMiddleware())
	router.Use(s.metrics.HTTPMetricsMiddleware())

	// Health check endpoints (no authentication required)
	s.setupHealthRoutes(router)

	// Metrics endpoint (no authentication required)
	s.setupMetricsRoutes(router)

	// API documentation endpoint (no authentication required)
	s.setupDocsRoutes(router)

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

	// Exchange binding management routes
	s.setupExchangeBindingRoutes(protected)

	// Trading management routes
	s.setupTradingRoutes(protected)

	// Sub-account management routes
	s.setupSubAccountRoutes(protected)

	// Transaction query routes
	s.setupTransactionRoutes(protected)

	// Trading log management routes
	s.setupTradingLogRoutes(protected)

	s.router = router
	return router
}

// setupHealthRoutes sets up health check routes
func (s *Server) setupHealthRoutes(router *gin.Engine) {
	healthHandler := NewHealthHandler(s.db, s.natsManager)

	// Kubernetes liveness probe - simple check that the app is running
	router.GET("/health/live", healthHandler.LivenessProbe)
	
	// Kubernetes readiness probe - check if app can serve traffic
	router.GET("/health/ready", healthHandler.ReadinessProbe)
	
	// Detailed health check with dependency information
	router.GET("/health", healthHandler.HealthCheck)
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

// setupExchangeBindingRoutes sets up exchange binding management routes
func (s *Server) setupExchangeBindingRoutes(protected *gin.RouterGroup) {
	exchangeBindingHandler := NewExchangeBindingHandler(s.exchangeBindingService)

	bindings := protected.Group("/exchange-bindings")

	// User exchange binding routes
	bindings.POST("", exchangeBindingHandler.CreateExchangeBinding)
	bindings.GET("", exchangeBindingHandler.GetUserExchangeBindings)
	bindings.GET("/:id", exchangeBindingHandler.GetExchangeBinding)
	bindings.PUT("/:id", exchangeBindingHandler.UpdateExchangeBinding)
	bindings.DELETE("/:id", exchangeBindingHandler.DeleteExchangeBinding)

	// Public exchange bindings (read-only, but still under protected group for user context)
	publicBindings := protected.Group("/exchange-bindings/public")
	publicBindings.GET("", exchangeBindingHandler.GetPublicExchangeBindings)
}

// setupTradingRoutes sets up trading management routes
func (s *Server) setupTradingRoutes(protected *gin.RouterGroup) {
	tradingHandler := NewTradingHandler(s.tradingService)

	tradings := protected.Group("/tradings")

	// User trading routes
	tradings.POST("", tradingHandler.CreateTrading)
	tradings.GET("", tradingHandler.GetUserTradings)
	tradings.GET("/:id", tradingHandler.GetTrading)
	tradings.PUT("/:id", tradingHandler.UpdateTrading)
	tradings.DELETE("/:id", tradingHandler.DeleteTrading)

	// Admin trading routes
	adminTradings := protected.Group("/admin/tradings")
	adminTradings.Use(middleware.AdminMiddleware())

	adminTradings.GET("", tradingHandler.ListTradings)
	adminTradings.GET("/:id", tradingHandler.GetTradingByID)
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
	transactions.GET("/trading/:trading_id", transactionHandler.GetTradingTransactions)
	transactions.GET("/time-range", transactionHandler.GetTransactionsByTimeRange)

	// Admin transaction routes
	adminTransactions := protected.Group("/admin/transactions")
	adminTransactions.Use(middleware.AdminMiddleware())

	adminTransactions.GET("", transactionHandler.ListAllTransactions)
	adminTransactions.GET("/:id", transactionHandler.GetTransactionByID)
}

// setupTradingLogRoutes sets up trading log management routes
func (s *Server) setupTradingLogRoutes(protected *gin.RouterGroup) {
	tradingLogHandler := NewTradingLogHandler(s.tradingLogService)

	tradingLogs := protected.Group("/trading-logs")

	// User trading log routes with specific rate limiting for creation
	tradingLogsCreation := tradingLogs.Group("")
	tradingLogsCreation.Use(middleware.TradingRateLimitMiddleware())
	tradingLogsCreation.POST("", tradingLogHandler.CreateTradingLog)
	tradingLogs.GET("", tradingLogHandler.GetUserTradingLogs)
	tradingLogs.GET("/:id", tradingLogHandler.GetTradingLog)
	tradingLogs.DELETE("/:id", tradingLogHandler.DeleteTradingLog)
	tradingLogs.GET("/sub-account/:sub_account_id", tradingLogHandler.GetSubAccountTradingLogs)
	tradingLogs.GET("/trading/:trading_id", tradingLogHandler.GetTradingLogs)
	tradingLogs.GET("/time-range", tradingLogHandler.GetTradingLogsByTimeRange)

	// Admin trading log routes
	adminTradingLogs := protected.Group("/admin/trading-logs")
	adminTradingLogs.Use(middleware.AdminMiddleware())

	adminTradingLogs.GET("", tradingLogHandler.ListAllTradingLogs)
	adminTradingLogs.GET("/:id", tradingLogHandler.GetTradingLogByID)
}

// setupMetricsRoutes sets up Prometheus metrics endpoints
func (s *Server) setupMetricsRoutes(router *gin.Engine) {
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
}

// setupDocsRoutes sets up API documentation endpoints
func (s *Server) setupDocsRoutes(router *gin.Engine) {
	// Swagger UI with custom handling for root path
	router.GET("/docs/*any", func(c *gin.Context) {
		path := c.Param("any")
		// Handle root path by redirecting to index.html
		if path == "/" {
			c.Redirect(301, "/docs/index.html")
			return
		}
		// Use default ginSwagger handler for all other paths
		ginSwagger.WrapHandler(swaggerFiles.Handler)(c)
	})
}

// GetRouter returns the configured router
func (s *Server) GetRouter() *gin.Engine {
	if s.router == nil {
		return s.SetupRoutes()
	}
	return s.router
}

// GetMetrics returns the metrics instance
func (s *Server) GetMetrics() *metrics.Metrics {
	return s.metrics
}

// GetJWTManager returns the JWT manager instance
func (s *Server) GetJWTManager() *auth.JWTManager {
	return s.jwtManager
}
