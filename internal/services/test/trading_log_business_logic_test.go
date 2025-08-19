package test

import (
	"testing"

	"tiris-backend/internal/repositories"
	"tiris-backend/internal/services"
	"tiris-backend/test/config"
	"tiris-backend/test/helpers"

	"github.com/stretchr/testify/require"
)

// TestTradingLogService_BusinessLogic_Integration tests business logic integration
func TestTradingLogService_BusinessLogic_Integration(t *testing.T) {
	// Note: Comprehensive integration tests have been moved to trading_log_integration_test.go
	// This test is kept for backwards compatibility but redirects to the new tests
	
	if testing.Short() {
		t.Skip("Integration tests skipped in short mode - run without -short flag")
	}
	
	t.Log("Comprehensive integration tests are available in trading_log_integration_test.go")
	t.Log("Run: go test -v ./internal/services/test -run 'TestTradingLogService_Integration'")
	
	// Basic smoke test to ensure service creation works
	testConfig := config.GetProfileConfig(config.ProfileQuick)
	dbHelper := helpers.NewDatabaseTestHelper(t, testConfig)
	
	repos := repositories.NewRepositories(dbHelper.DB)
	service := services.NewTradingLogService(repos, dbHelper.DB)
	
	// Verify service was created successfully
	require.NotNil(t, service)
}