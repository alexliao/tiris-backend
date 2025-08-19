package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Environment string
	Server      ServerConfig
	Database    DatabaseConfig
	Auth        AuthConfig
	NATS        NATSConfig
	OAuth       OAuthConfig
}

type ServerConfig struct {
	Port         string
	ReadTimeout  int
	WriteTimeout int
	IdleTimeout  int
}

type DatabaseConfig struct {
	Host         string
	Port         string
	Username     string
	Password     string
	DatabaseName string
	SSLMode      string
	MaxConns     int
	MaxIdleConns int
	MaxLifetime  int
}

type AuthConfig struct {
	JWTSecret         string
	JWTExpiration     int
	RefreshSecret     string
	RefreshExpiration int
}

type NATSConfig struct {
	Enabled     bool
	URL         string
	ClusterID   string
	ClientID    string
	DurableName string
}

type OAuthConfig struct {
	Google GoogleOAuthConfig
	WeChat WeChatOAuthConfig
}

type GoogleOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type WeChatOAuthConfig struct {
	AppID       string
	AppSecret   string
	RedirectURL string
}

func Load() (*Config, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		// .env file is optional
	}

	cfg := &Config{
		Environment: getEnvOrDefault("ENVIRONMENT", "development"),
		Server: ServerConfig{
			Port:         getEnvOrDefault("PORT", "8080"),
			ReadTimeout:  getEnvAsIntOrDefault("READ_TIMEOUT", 15),
			WriteTimeout: getEnvAsIntOrDefault("WRITE_TIMEOUT", 15),
			IdleTimeout:  getEnvAsIntOrDefault("IDLE_TIMEOUT", 60),
		},
		Database: DatabaseConfig{
			Host:         getEnvOrDefault("DB_HOST", "localhost"),
			Port:         getEnvOrDefault("DB_PORT", "5432"),
			Username:     getEnvOrDefault("DB_USER", "postgres"),
			Password:     getEnvOrDefault("DB_PASSWORD", ""),
			DatabaseName: getEnvOrDefault("DB_NAME", "tiris"),
			SSLMode:      getEnvOrDefault("DB_SSL_MODE", "disable"),
			MaxConns:     getEnvAsIntOrDefault("DB_MAX_CONNS", 25),
			MaxIdleConns: getEnvAsIntOrDefault("DB_MAX_IDLE_CONNS", 10),
			MaxLifetime:  getEnvAsIntOrDefault("DB_MAX_LIFETIME", 300),
		},
		Auth: AuthConfig{
			JWTSecret:         getRequiredEnv("JWT_SECRET"),
			JWTExpiration:     getEnvAsIntOrDefault("JWT_EXPIRATION", 3600),
			RefreshSecret:     getRequiredEnv("REFRESH_SECRET"),
			RefreshExpiration: getEnvAsIntOrDefault("REFRESH_EXPIRATION", 604800),
		},
		NATS: NATSConfig{
			Enabled:     getEnvAsBoolOrDefault("NATS_ENABLED", true),
			URL:         getEnvOrDefault("NATS_URL", "nats://localhost:4222"),
			ClusterID:   getEnvOrDefault("NATS_CLUSTER_ID", "tiris-cluster"),
			ClientID:    getEnvOrDefault("NATS_CLIENT_ID", "tiris-backend"),
			DurableName: getEnvOrDefault("NATS_DURABLE_NAME", "tiris-backend-durable"),
		},
		OAuth: OAuthConfig{
			Google: GoogleOAuthConfig{
				ClientID:     getEnvOrDefault("GOOGLE_CLIENT_ID", ""),
				ClientSecret: getEnvOrDefault("GOOGLE_CLIENT_SECRET", ""),
				RedirectURL:  getEnvOrDefault("GOOGLE_REDIRECT_URL", ""),
			},
			WeChat: WeChatOAuthConfig{
				AppID:       getEnvOrDefault("WECHAT_APP_ID", ""),
				AppSecret:   getEnvOrDefault("WECHAT_APP_SECRET", ""),
				RedirectURL: getEnvOrDefault("WECHAT_REDIRECT_URL", ""),
			},
		},
	}

	return cfg, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBoolOrDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getRequiredEnv(key string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	panic(fmt.Sprintf("Required environment variable %s is not set", key))
}
