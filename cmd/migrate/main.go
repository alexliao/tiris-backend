package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"tiris-backend/internal/config"
	"tiris-backend/internal/database"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database connection
	db, err := database.Initialize(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close(db)

	command := os.Args[1]

	switch command {
	case "up":
		err = database.RunMigrations(db.DB, cfg.Database)
		if err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
		fmt.Println("Migrations completed successfully")

	case "down":
		steps := 1
		if len(os.Args) > 2 {
			steps, err = strconv.Atoi(os.Args[2])
			if err != nil {
				log.Fatalf("Invalid steps argument: %v", err)
			}
		}
		err = database.RollbackMigrations(db.DB, steps)
		if err != nil {
			log.Fatalf("Rollback failed: %v", err)
		}

	case "version":
		version, dirty, err := database.GetMigrationVersion(db.DB)
		if err != nil {
			log.Fatalf("Failed to get migration version: %v", err)
		}
		if dirty {
			fmt.Printf("Current migration version: %d (dirty)\n", version)
		} else {
			fmt.Printf("Current migration version: %d\n", version)
		}

	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: migrate <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  up                 Run all pending migrations")
	fmt.Println("  down [steps]       Rollback migrations (default: 1 step)")
	fmt.Println("  version            Show current migration version")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  migrate up")
	fmt.Println("  migrate down")
	fmt.Println("  migrate down 3")
	fmt.Println("  migrate version")
}
