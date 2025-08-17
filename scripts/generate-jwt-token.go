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
		fmt.Fprintf(os.Stderr, "Error: user-id, username, and email are required\n")
		fmt.Fprintf(os.Stderr, "\nUsage examples:\n")
		fmt.Fprintf(os.Stderr, "  %s --user-id 123e4567-e89b-12d3-a456-426614174000 --username john --email john@example.com\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --user-id 123e4567-e89b-12d3-a456-426614174000 --username jane --email jane@example.com --duration 24h\n", os.Args[0])
		os.Exit(1)
	}

	// Parse user ID
	parsedUserID, err := uuid.Parse(*userID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Invalid user ID format '%s'\n", *userID)
		fmt.Fprintf(os.Stderr, "Expected: A valid UUID (e.g., 123e4567-e89b-12d3-a456-426614174000)\n")
		fmt.Fprintf(os.Stderr, "Got: %s\n", *userID)
		os.Exit(1)
	}

	// Parse duration
	tokenDuration, err := time.ParseDuration(*duration)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Invalid duration format '%s'\n", *duration)
		fmt.Fprintf(os.Stderr, "Expected: Go duration format (e.g., 1h, 24h, 8760h, 365d)\n")
		fmt.Fprintf(os.Stderr, "Got: %s\n", *duration)
		os.Exit(1)
	}

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		// Try to load from current directory first, then parent
		if err := godotenv.Load("../.env"); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not load .env file: %v\n", err)
			fmt.Fprintf(os.Stderr, "Environment variables must be set manually\n")
		}
	}

	// Get JWT secrets from environment
	jwtSecret := os.Getenv("JWT_SECRET")
	refreshSecret := os.Getenv("REFRESH_SECRET")
	
	if jwtSecret == "" {
		fmt.Fprintf(os.Stderr, "Error: JWT_SECRET environment variable is required\n")
		fmt.Fprintf(os.Stderr, "Solutions:\n")
		fmt.Fprintf(os.Stderr, "  1. Create a .env file with JWT_SECRET=your_secret_key\n")
		fmt.Fprintf(os.Stderr, "  2. Set environment variable: export JWT_SECRET=your_secret_key\n")
		fmt.Fprintf(os.Stderr, "  3. Copy from .env.example: cp .env.example .env\n")
		os.Exit(1)
	}
	if refreshSecret == "" {
		fmt.Fprintf(os.Stderr, "Error: REFRESH_SECRET environment variable is required\n")
		fmt.Fprintf(os.Stderr, "Solutions:\n")
		fmt.Fprintf(os.Stderr, "  1. Create a .env file with REFRESH_SECRET=your_refresh_secret\n")
		fmt.Fprintf(os.Stderr, "  2. Set environment variable: export REFRESH_SECRET=your_refresh_secret\n")
		fmt.Fprintf(os.Stderr, "  3. Copy from .env.example: cp .env.example .env\n")
		os.Exit(1)
	}

	// Create JWT manager
	jwtManager := auth.NewJWTManager(jwtSecret, refreshSecret, tokenDuration, 24*time.Hour*7) // 7 days refresh

	// Generate token
	token, err := jwtManager.GenerateToken(parsedUserID, *username, *email, *role)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to generate JWT token: %v\n", err)
		fmt.Fprintf(os.Stderr, "This could be due to:\n")
		fmt.Fprintf(os.Stderr, "  - Invalid JWT secrets\n")
		fmt.Fprintf(os.Stderr, "  - System time issues\n")
		fmt.Fprintf(os.Stderr, "  - Memory constraints\n")
		os.Exit(1)
	}

	// Validate generated token
	if token == "" {
		fmt.Fprintf(os.Stderr, "Error: Generated token is empty\n")
		os.Exit(1)
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