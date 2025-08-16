package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"tiris-backend/pkg/auth"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

// UserData represents the data needed to generate a JWT token
type UserData struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

func main() {
	// Command line flags
	var (
		userID   = flag.String("user-id", "", "User UUID (required)")
		username = flag.String("username", "", "Username (required)")
		email    = flag.String("email", "", "Email (required)")
		role     = flag.String("role", "user", "User role (default: user)")
		duration = flag.String("duration", "1h", "Token duration (e.g., 1h, 24h, 365d)")
		output   = flag.String("output", "token", "Output format: token, json, or curl")
	)
	flag.Parse()

	// Validate required fields
	if *userID == "" || *username == "" || *email == "" {
		fmt.Println("Error: user-id, username, and email are required")
		flag.Usage()
		os.Exit(1)
	}

	// Parse user ID
	parsedUserID, err := uuid.Parse(*userID)
	if err != nil {
		log.Fatalf("Invalid user ID format: %v", err)
	}

	// Parse duration
	tokenDuration, err := time.ParseDuration(*duration)
	if err != nil {
		log.Fatalf("Invalid duration format: %v", err)
	}

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		// .env file is optional, continue without it
	}

	// Get JWT secrets from environment
	jwtSecret := os.Getenv("JWT_SECRET")
	refreshSecret := os.Getenv("REFRESH_SECRET")
	
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}
	if refreshSecret == "" {
		log.Fatal("REFRESH_SECRET environment variable is required")
	}

	// Create JWT manager
	jwtManager := auth.NewJWTManager(jwtSecret, refreshSecret, tokenDuration, 24*time.Hour*7) // 7 days refresh

	// Generate token
	token, err := jwtManager.GenerateToken(parsedUserID, *username, *email, *role)
	if err != nil {
		log.Fatalf("Failed to generate token: %v", err)
	}

	// Output based on format
	switch *output {
	case "token":
		fmt.Println(token)
	case "json":
		result := map[string]interface{}{
			"access_token": token,
			"token_type":   "Bearer",
			"expires_in":   int64(tokenDuration.Seconds()),
			"user": UserData{
				UserID:   *userID,
				Username: *username,
				Email:    *email,
				Role:     *role,
			},
		}
		jsonBytes, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			log.Fatalf("Failed to marshal JSON: %v", err)
		}
		fmt.Println(string(jsonBytes))
	case "curl":
		fmt.Printf("curl -H \"Authorization: Bearer %s\" http://localhost:8080/v1/users/me\n", token)
	default:
		log.Fatalf("Invalid output format: %s (must be token, json, or curl)", *output)
	}
}