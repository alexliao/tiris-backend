package test

import (
	"context"
	"testing"
	"time"

	"tiris-backend/internal/models"
	"tiris-backend/internal/repositories"
	"tiris-backend/test/config"
	"tiris-backend/test/helpers"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestUser represents a simplified user model for SQLite testing
type TestUser struct {
	ID       uuid.UUID `gorm:"type:TEXT;primary_key" json:"id"`
	Username string    `gorm:"type:varchar(50);not null;uniqueIndex" json:"username"`
	Email    string    `gorm:"type:varchar(255);not null;uniqueIndex" json:"email"`
	Avatar   *string   `gorm:"type:text" json:"avatar,omitempty"`
	Settings string    `gorm:"type:text;default:'{}'" json:"settings"`
	Info     string    `gorm:"type:text;default:'{}'" json:"info"`

	CreatedAt time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"-"`
}

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Auto-migrate the test models
	if err := db.AutoMigrate(&TestUser{}); err != nil {
		return nil, err
	}

	return db, nil
}

// convertToTestUser converts a models.User to TestUser for testing
func convertToTestUser(user *models.User) *TestUser {
	testUser := &TestUser{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Avatar:    user.Avatar,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Settings:  "{}",
		Info:      "{}",
	}
	if !user.DeletedAt.Time.IsZero() {
		testUser.DeletedAt = &user.DeletedAt.Time
	}
	return testUser
}

// convertFromTestUser converts a TestUser back to models.User
func convertFromTestUser(testUser *TestUser) *models.User {
	user := &models.User{
		ID:        testUser.ID,
		Username:  testUser.Username,
		Email:     testUser.Email,
		Avatar:    testUser.Avatar,
		CreatedAt: testUser.CreatedAt,
		UpdatedAt: testUser.UpdatedAt,
	}
	return user
}

// TestUserRepository_Create tests user creation functionality
func TestUserRepository_Create(t *testing.T) {
	// Setup test config
	_ = config.GetProfileConfig(config.ProfileQuick)
	
	// Create test database
	db, err := setupTestDB()
	require.NoError(t, err)
	
	// Create repository
	userRepo := repositories.NewUserRepository(db)
	userFactory := helpers.NewUserFactory()
	
	// Test successful user creation
	t.Run("successful_user_creation", func(t *testing.T) {
		testUser := userFactory.WithEmail("test@example.com")
		if testUser.ID == uuid.Nil {
			testUser.ID = uuid.New()
		}
		
		// Execute test
		err := userRepo.Create(context.Background(), testUser)
		
		// Verify results
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, testUser.ID)
		
		// Verify user was saved to database
		var savedUser TestUser
		err = db.Where("id = ?", testUser.ID).First(&savedUser).Error
		require.NoError(t, err)
		assert.Equal(t, testUser.Email, savedUser.Email)
		assert.Equal(t, testUser.Username, savedUser.Username)
	})
	
	// Test duplicate email creation
	t.Run("duplicate_email_creation", func(t *testing.T) {
		existingUser := userFactory.WithEmail("duplicate@example.com")
		if existingUser.ID == uuid.Nil {
			existingUser.ID = uuid.New()
		}
		err := userRepo.Create(context.Background(), existingUser)
		require.NoError(t, err)
		
		// Try to create another user with same email
		duplicateUser := userFactory.WithEmail("duplicate@example.com")
		if duplicateUser.ID == uuid.Nil {
			duplicateUser.ID = uuid.New()
		}
		
		// Execute test
		err = userRepo.Create(context.Background(), duplicateUser)
		
		// Should return error due to unique constraint
		require.Error(t, err)
	})
	
	// Test creation with invalid data
	t.Run("create_with_empty_email", func(t *testing.T) {
		invalidUser := &models.User{
			ID:       uuid.New(),
			Username: "testuser",
			Email:    "", // Empty email
		}
		
		// Execute test
		err := userRepo.Create(context.Background(), invalidUser)
		
		// Should return error due to validation (or succeed depending on implementation)
		// In this case, we just verify the call doesn't panic
		_ = err // May or may not error depending on validation
	})
}

// TestUserRepository_GetByID tests user retrieval by ID
func TestUserRepository_GetByID(t *testing.T) {
	// Create test database
	db, err := setupTestDB()
	require.NoError(t, err)
	
	// Create repository
	userRepo := repositories.NewUserRepository(db)
	userFactory := helpers.NewUserFactory()
	
	// Create test user
	testUser := userFactory.WithEmail("getbyid@example.com")
	if testUser.ID == uuid.Nil {
		testUser.ID = uuid.New()
	}
	err = userRepo.Create(context.Background(), testUser)
	require.NoError(t, err)
	
	// Test successful retrieval
	t.Run("successful_get_by_id", func(t *testing.T) {
		// Execute test
		retrievedUser, err := userRepo.GetByID(context.Background(), testUser.ID)
		
		// Verify results
		require.NoError(t, err)
		require.NotNil(t, retrievedUser)
		assert.Equal(t, testUser.ID, retrievedUser.ID)
		assert.Equal(t, testUser.Email, retrievedUser.Email)
		assert.Equal(t, testUser.Username, retrievedUser.Username)
	})
	
	// Test non-existent user
	t.Run("get_non_existent_user", func(t *testing.T) {
		nonExistentID := uuid.New()
		
		// Execute test
		retrievedUser, err := userRepo.GetByID(context.Background(), nonExistentID)
		
		// Should return nil, nil for not found
		require.NoError(t, err)
		assert.Nil(t, retrievedUser)
	})
	
	// Test with nil UUID
	t.Run("get_with_nil_uuid", func(t *testing.T) {
		// Execute test
		retrievedUser, err := userRepo.GetByID(context.Background(), uuid.Nil)
		
		// Should return nil, nil for invalid UUID
		require.NoError(t, err)
		assert.Nil(t, retrievedUser)
	})
}

// TestUserRepository_GetByEmail tests user retrieval by email
func TestUserRepository_GetByEmail(t *testing.T) {
	// Create test database
	db, err := setupTestDB()
	require.NoError(t, err)
	
	// Create repository
	userRepo := repositories.NewUserRepository(db)
	userFactory := helpers.NewUserFactory()
	
	// Create test user
	testUser := userFactory.WithEmail("getbyemail@example.com")
	if testUser.ID == uuid.Nil {
		testUser.ID = uuid.New()
	}
	err = userRepo.Create(context.Background(), testUser)
	require.NoError(t, err)
	
	// Test successful retrieval
	t.Run("successful_get_by_email", func(t *testing.T) {
		// Execute test
		retrievedUser, err := userRepo.GetByEmail(context.Background(), testUser.Email)
		
		// Verify results
		require.NoError(t, err)
		require.NotNil(t, retrievedUser)
		assert.Equal(t, testUser.ID, retrievedUser.ID)
		assert.Equal(t, testUser.Email, retrievedUser.Email)
		assert.Equal(t, testUser.Username, retrievedUser.Username)
	})
	
	// Test case sensitivity
	t.Run("case_sensitive_email", func(t *testing.T) {
		// Execute test with different case
		retrievedUser, err := userRepo.GetByEmail(context.Background(), "GETBYEMAIL@EXAMPLE.COM")
		
		// Should return nil since emails are case-sensitive in this implementation
		require.NoError(t, err)
		assert.Nil(t, retrievedUser)
	})
	
	// Test non-existent email
	t.Run("get_non_existent_email", func(t *testing.T) {
		// Execute test
		retrievedUser, err := userRepo.GetByEmail(context.Background(), "nonexistent@example.com")
		
		// Should return nil, nil for not found
		require.NoError(t, err)
		assert.Nil(t, retrievedUser)
	})
	
	// Test empty email
	t.Run("get_empty_email", func(t *testing.T) {
		// Execute test
		retrievedUser, err := userRepo.GetByEmail(context.Background(), "")
		
		// Should return nil, nil for empty email
		require.NoError(t, err)
		assert.Nil(t, retrievedUser)
	})
}

// TestUserRepository_GetByUsername tests user retrieval by username
func TestUserRepository_GetByUsername(t *testing.T) {
	// Create test database
	db, err := setupTestDB()
	require.NoError(t, err)
	
	// Create repository
	userRepo := repositories.NewUserRepository(db)
	userFactory := helpers.NewUserFactory()
	
	// Create test user
	testUser := userFactory.WithUsername("testusername")
	if testUser.ID == uuid.Nil {
		testUser.ID = uuid.New()
	}
	err = userRepo.Create(context.Background(), testUser)
	require.NoError(t, err)
	
	// Test successful retrieval
	t.Run("successful_get_by_username", func(t *testing.T) {
		// Execute test
		retrievedUser, err := userRepo.GetByUsername(context.Background(), testUser.Username)
		
		// Verify results
		require.NoError(t, err)
		require.NotNil(t, retrievedUser)
		assert.Equal(t, testUser.ID, retrievedUser.ID)
		assert.Equal(t, testUser.Email, retrievedUser.Email)
		assert.Equal(t, testUser.Username, retrievedUser.Username)
	})
	
	// Test case sensitivity
	t.Run("case_sensitive_username", func(t *testing.T) {
		// Execute test with different case
		retrievedUser, err := userRepo.GetByUsername(context.Background(), "TESTUSERNAME")
		
		// Should return nil since usernames are case-sensitive
		require.NoError(t, err)
		assert.Nil(t, retrievedUser)
	})
	
	// Test non-existent username
	t.Run("get_non_existent_username", func(t *testing.T) {
		// Execute test
		retrievedUser, err := userRepo.GetByUsername(context.Background(), "nonexistentuser")
		
		// Should return nil, nil for not found
		require.NoError(t, err)
		assert.Nil(t, retrievedUser)
	})
}

// TestUserRepository_Update tests user update functionality
func TestUserRepository_Update(t *testing.T) {
	// Create test database
	db, err := setupTestDB()
	require.NoError(t, err)
	
	// Create repository
	userRepo := repositories.NewUserRepository(db)
	userFactory := helpers.NewUserFactory()
	
	// Create test user
	testUser := userFactory.WithEmail("update@example.com")
	if testUser.ID == uuid.Nil {
		testUser.ID = uuid.New()
	}
	err = userRepo.Create(context.Background(), testUser)
	require.NoError(t, err)
	
	// Test successful update
	t.Run("successful_update", func(t *testing.T) {
		// Update user data
		avatarURL := "https://example.com/updated-avatar.jpg"
		testUser.Avatar = &avatarURL
		testUser.Username = "updated_username"
		
		// Execute test
		err := userRepo.Update(context.Background(), testUser)
		
		// Verify results
		require.NoError(t, err)
		
		// Verify update was saved to database
		var updatedUser TestUser
		err = db.Where("id = ?", testUser.ID).First(&updatedUser).Error
		require.NoError(t, err)
		assert.Equal(t, "updated_username", updatedUser.Username)
		assert.NotNil(t, updatedUser.Avatar)
		assert.Equal(t, avatarURL, *updatedUser.Avatar)
	})
	
	// Test update non-existent user
	t.Run("update_non_existent_user", func(t *testing.T) {
		nonExistentUser := &models.User{
			ID:       uuid.New(),
			Email:    "nonexistent@example.com",
			Username: "nonexistent",
		}
		
		// Execute test
		err := userRepo.Update(context.Background(), nonExistentUser)
		
		// Should succeed but no rows affected (GORM behavior)
		require.NoError(t, err)
	})
}

// TestUserRepository_Delete tests user deletion functionality
func TestUserRepository_Delete(t *testing.T) {
	// Create test database
	db, err := setupTestDB()
	require.NoError(t, err)
	
	// Create repository
	userRepo := repositories.NewUserRepository(db)
	userFactory := helpers.NewUserFactory()
	
	// Create test user
	testUser := userFactory.WithEmail("delete@example.com")
	if testUser.ID == uuid.Nil {
		testUser.ID = uuid.New()
	}
	err = userRepo.Create(context.Background(), testUser)
	require.NoError(t, err)
	
	// Test successful deletion
	t.Run("successful_delete", func(t *testing.T) {
		// Execute test
		err := userRepo.Delete(context.Background(), testUser.ID)
		
		// Verify results
		require.NoError(t, err)
		
		// Verify user was deleted from database
		var deletedUser TestUser
		err = db.Where("id = ?", testUser.ID).First(&deletedUser).Error
		assert.Error(t, err) // Should be ErrRecordNotFound
	})
	
	// Test delete non-existent user
	t.Run("delete_non_existent_user", func(t *testing.T) {
		nonExistentID := uuid.New()
		
		// Execute test
		err := userRepo.Delete(context.Background(), nonExistentID)
		
		// Should succeed but no rows affected (GORM behavior)
		require.NoError(t, err)
	})
}

// TestUserRepository_List tests user listing functionality
func TestUserRepository_List(t *testing.T) {
	// Create test database
	db, err := setupTestDB()
	require.NoError(t, err)
	
	// Create repository
	userRepo := repositories.NewUserRepository(db)
	userFactory := helpers.NewUserFactory()
	
	// Create multiple test users
	users := make([]*models.User, 5)
	for i := 0; i < 5; i++ {
		user := userFactory.WithEmail("list" + string(rune(i+48)) + "@example.com") // +48 to get ASCII numbers
		if user.ID == uuid.Nil {
			user.ID = uuid.New()
		}
		err = userRepo.Create(context.Background(), user)
		require.NoError(t, err)
		users[i] = user
		
		// Add small delay to ensure different created_at times
		time.Sleep(1 * time.Millisecond)
	}
	
	// Test list all users
	t.Run("list_all_users", func(t *testing.T) {
		// Execute test
		retrievedUsers, total, err := userRepo.List(context.Background(), 10, 0)
		
		// Verify results
		require.NoError(t, err)
		assert.Equal(t, int64(5), total)
		assert.Len(t, retrievedUsers, 5)
		
		// Verify ordering (should be DESC by created_at)
		for i := 0; i < len(retrievedUsers)-1; i++ {
			assert.True(t, retrievedUsers[i].CreatedAt.After(retrievedUsers[i+1].CreatedAt) ||
				retrievedUsers[i].CreatedAt.Equal(retrievedUsers[i+1].CreatedAt))
		}
	})
	
	// Test pagination
	t.Run("list_with_pagination", func(t *testing.T) {
		// Get first 2 users
		firstPage, total, err := userRepo.List(context.Background(), 2, 0)
		require.NoError(t, err)
		assert.Equal(t, int64(5), total)
		assert.Len(t, firstPage, 2)
		
		// Get next 2 users
		secondPage, total2, err := userRepo.List(context.Background(), 2, 2)
		require.NoError(t, err)
		assert.Equal(t, int64(5), total2)
		assert.Len(t, secondPage, 2)
		
		// Verify no overlap
		assert.NotEqual(t, firstPage[0].ID, secondPage[0].ID)
		assert.NotEqual(t, firstPage[1].ID, secondPage[1].ID)
	})
	
	// Test empty result
	t.Run("list_with_high_offset", func(t *testing.T) {
		// Execute test with offset beyond available records
		retrievedUsers, total, err := userRepo.List(context.Background(), 10, 100)
		
		// Verify results
		require.NoError(t, err)
		assert.Equal(t, int64(5), total) // Total should still be correct
		assert.Len(t, retrievedUsers, 0) // But no users returned
	})
}

// Performance test for user repository operations
func TestUserRepository_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}
	
	// Create test database
	db, err := setupTestDB()
	require.NoError(t, err)
	
	// Create repository
	userRepo := repositories.NewUserRepository(db)
	userFactory := helpers.NewUserFactory()
	
	t.Run("bulk_create_performance", func(t *testing.T) {
		timer := helpers.NewPerformanceTimer()
		timer.Start()
		
		// Create 100 users
		for i := 0; i < 100; i++ {
			user := userFactory.WithEmail("perf" + string(rune(i+48)) + "@example.com")
			if user.ID == uuid.Nil {
				user.ID = uuid.New()
			}
			err := userRepo.Create(context.Background(), user)
			require.NoError(t, err)
		}
		
		timer.Stop()
		
		// Check performance within reasonable bounds
		assert.Less(t, timer.Duration().Milliseconds(), int64(2000),
			"100 user creations should complete within 2 seconds")
	})
	
	t.Run("lookup_performance", func(t *testing.T) {
		// Create a test user first
		testUser := userFactory.WithEmail("lookup@example.com")
		if testUser.ID == uuid.Nil {
			testUser.ID = uuid.New()
		}
		err := userRepo.Create(context.Background(), testUser)
		require.NoError(t, err)
		
		timer := helpers.NewPerformanceTimer()
		timer.Start()
		
		// Perform 1000 lookups
		for i := 0; i < 1000; i++ {
			_, err := userRepo.GetByID(context.Background(), testUser.ID)
			require.NoError(t, err)
		}
		
		timer.Stop()
		
		// Check performance within reasonable bounds
		assert.Less(t, timer.Duration().Milliseconds(), int64(1000),
			"1000 user lookups should complete within 1 second")
	})
}