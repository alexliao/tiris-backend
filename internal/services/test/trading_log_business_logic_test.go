package test

import (
	"testing"
)

// TestTradingLogService_BusinessLogic_Integration tests business logic integration
func TestTradingLogService_BusinessLogic_Integration(t *testing.T) {
	// Note: Comprehensive integration tests have been moved to trading_log_integration_test.go
	// This test is kept for backwards compatibility but redirects to the new tests

	// Skip integration tests in unit test package - they belong in internal/integration
	t.Skip("Integration test moved to internal/integration package - run make test-integration-docker")

	t.Log("Comprehensive integration tests are available in trading_log_integration_test.go")
	t.Log("Run: go test -v ./internal/services/test -run 'TestTradingLogService_Integration'")
}
