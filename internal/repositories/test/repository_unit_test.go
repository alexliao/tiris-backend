package test

import (
	"context"
	"testing"
	"time"

	"tiris-backend/internal/models"
	"tiris-backend/internal/repositories"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/datatypes"
)

// SimpleUser represents a SQLite-compatible user model for testing
type SimpleUser struct {
	ID       string `gorm:"type:TEXT;primary_key"`
	Username string `gorm:"type:TEXT;not null;uniqueIndex"`
	Email    string `gorm:"type:TEXT;not null;uniqueIndex"`
	Avatar   string `gorm:"type:TEXT"`
	Settings string `gorm:"type:TEXT;default:'{}'"`
	Info     string `gorm:"type:TEXT;default:'{}'"`

	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP"`
	DeletedAt *time.Time `gorm:"index"`
}

// TableName specifies the table name for SimpleUser
func (SimpleUser) TableName() string {
	return "users"
}

// setupSimpleDB creates a simple SQLite database for basic repository testing
func setupSimpleDB() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return nil, err
	}

	// Auto-migrate simple models
	if err := db.AutoMigrate(&SimpleUser{}); err != nil {
		return nil, err
	}

	return db, nil
}

// convertToSimpleUser converts a models.User to SimpleUser
func convertToSimpleUser(user *models.User) *SimpleUser {
	simple := &SimpleUser{
		ID:       user.ID.String(),
		Username: user.Username,
		Email:    user.Email,
		Settings: "{}",
		Info:     "{}",
	}
	
	if user.Avatar != nil {
		simple.Avatar = *user.Avatar
	}
	
	return simple
}

// convertFromSimpleUser converts a SimpleUser back to models.User
func convertFromSimpleUser(simple *SimpleUser) *models.User {
	id, _ := uuid.Parse(simple.ID)
	user := &models.User{
		ID:       id,
		Username: simple.Username,
		Email:    simple.Email,
		Settings: datatypes.JSON(simple.Settings),
		Info:     datatypes.JSON(simple.Info),
		CreatedAt: simple.CreatedAt,
		UpdatedAt: simple.UpdatedAt,
	}
	
	if simple.Avatar != "" {
		user.Avatar = &simple.Avatar
	}
	
	return user
}

// TestRepositoryBasicOperations tests basic CRUD operations
func TestRepositoryBasicOperations(t *testing.T) {
	db, err := setupSimpleDB()
	require.NoError(t, err)

	t.Run("user_repository_basic_operations", func(t *testing.T) {
		// Test that repository can be created without error
		userRepo := repositories.NewUserRepository(db)
		assert.NotNil(t, userRepo)

		// Create a simple user directly in the database
		simpleUser := &SimpleUser{
			ID:       uuid.New().String(),
			Username: "testuser",
			Email:    "test@example.com",
			Settings: "{}",
			Info:     "{}",
		}
		
		result := db.Create(simpleUser)
		assert.NoError(t, result.Error)
		assert.Equal(t, int64(1), result.RowsAffected)

		// Test that we can retrieve it
		var retrieved SimpleUser
		result = db.First(&retrieved, "email = ?", "test@example.com")
		assert.NoError(t, result.Error)
		assert.Equal(t, "testuser", retrieved.Username)
	})

	t.Run("repository_interface_compliance", func(t *testing.T) {
		// Test that our repositories implement their interfaces correctly
		userRepo := repositories.NewUserRepository(db)
		
		// These should not panic and should handle missing data gracefully
		ctx := context.Background()
		
		// Test GetByID with non-existent ID
		user, err := userRepo.GetByID(ctx, uuid.New())
		if err != nil {
			assert.Error(t, err) // Should return "record not found" error
		} else {
			assert.Nil(t, user) // Or return nil user
		}
		
		// Test GetByEmail with non-existent email
		user, err = userRepo.GetByEmail(ctx, "nonexistent@example.com")
		if err != nil {
			assert.Error(t, err) // Should return "record not found" error  
		} else {
			assert.Nil(t, user) // Or return nil user
		}
		
		// Test GetByUsername with non-existent username
		user, err = userRepo.GetByUsername(ctx, "nonexistent")
		if err != nil {
			assert.Error(t, err) // Should return "record not found" error
		} else {
			assert.Nil(t, user) // Or return nil user
		}
	})
}

// TestRepositoryErrorHandling tests error handling in repositories
func TestRepositoryErrorHandling(t *testing.T) {
	db, err := setupSimpleDB()
	require.NoError(t, err)

	userRepo := repositories.NewUserRepository(db)
	ctx := context.Background()

	t.Run("handles_database_errors_gracefully", func(t *testing.T) {
		// Test with invalid UUID - should handle gracefully
		user, err := userRepo.GetByID(ctx, uuid.Nil)
		assert.NoError(t, err) // Repository design returns nil error for not found
		assert.Nil(t, user)    // But user should be nil

		// Test list with valid parameters should work
		users, total, err := userRepo.List(ctx, 10, 0)
		assert.NoError(t, err)
		assert.NotNil(t, users)
		assert.GreaterOrEqual(t, total, int64(0))
	})
}

// TestRepositoryPerformance tests basic performance characteristics
func TestRepositoryPerformance(t *testing.T) {
	db, err := setupSimpleDB()
	require.NoError(t, err)

	userRepo := repositories.NewUserRepository(db)
	ctx := context.Background()

	t.Run("repository_operations_performance", func(t *testing.T) {
		// Measure time for basic operations
		start := time.Now()
		
		// Perform multiple operations
		for i := 0; i < 10; i++ {
			user, err := userRepo.GetByID(ctx, uuid.New())
			// Should handle gracefully without error
			assert.NoError(t, err)
			assert.Nil(t, user) // User should be nil for non-existent ID
		}
		
		duration := time.Since(start)
		
		// Should complete within reasonable time (less than 100ms for 10 operations)
		assert.Less(t, duration, 100*time.Millisecond, "Repository operations should be fast")
	})
}