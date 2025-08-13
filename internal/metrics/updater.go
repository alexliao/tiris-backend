package metrics

import (
	"context"
	"log"
	"time"

	"tiris-backend/internal/repositories"
)

// MetricsUpdater periodically updates business metrics
type MetricsUpdater struct {
	metrics *Metrics
	repos   *repositories.Repositories
	ticker  *time.Ticker
	done    chan bool
}

// NewMetricsUpdater creates a new metrics updater
func NewMetricsUpdater(metrics *Metrics, repos *repositories.Repositories, interval time.Duration) *MetricsUpdater {
	return &MetricsUpdater{
		metrics: metrics,
		repos:   repos,
		ticker:  time.NewTicker(interval),
		done:    make(chan bool),
	}
}

// Start begins the metrics update loop
func (u *MetricsUpdater) Start() {
	go func() {
		// Update metrics immediately on start
		u.updateMetrics()

		for {
			select {
			case <-u.ticker.C:
				u.updateMetrics()
			case <-u.done:
				return
			}
		}
	}()
}

// Stop stops the metrics update loop
func (u *MetricsUpdater) Stop() {
	u.ticker.Stop()
	u.done <- true
}

// updateMetrics updates all gauge metrics with current values
func (u *MetricsUpdater) updateMetrics() {
	ctx := context.Background()

	// Update user count
	userCount, err := u.getUserCount(ctx)
	if err != nil {
		log.Printf("Failed to get user count for metrics: %v", err)
	} else {
		u.metrics.UsersTotal.Set(float64(userCount))
	}

	// Update exchange count
	exchangeCount, err := u.getExchangeCount(ctx)
	if err != nil {
		log.Printf("Failed to get exchange count for metrics: %v", err)
	} else {
		u.metrics.ExchangesTotal.Set(float64(exchangeCount))
	}

	// Update sub-account count
	subAccountCount, err := u.getSubAccountCount(ctx)
	if err != nil {
		log.Printf("Failed to get sub-account count for metrics: %v", err)
	} else {
		u.metrics.SubAccountsTotal.Set(float64(subAccountCount))
	}
}

// getUserCount gets the total number of users
func (u *MetricsUpdater) getUserCount(ctx context.Context) (int64, error) {
	// We'll use a repository method that can count users
	// This is a simplified approach - in a real system you'd want a dedicated count method
	users, total, err := u.repos.User.List(ctx, 1, 0) // Get just one user to get the total count
	_ = users // We don't need the actual users, just the count
	
	if err != nil {
		return 0, err
	}
	
	return total, nil
}

// getExchangeCount gets the total number of exchanges
func (u *MetricsUpdater) getExchangeCount(ctx context.Context) (int64, error) {
	// For now, we'll return 0 and implement this when the repository has the needed methods
	// TODO: Add List method to ExchangeRepository for proper counting
	return 0, nil
}

// getSubAccountCount gets the total number of sub-accounts
func (u *MetricsUpdater) getSubAccountCount(ctx context.Context) (int64, error) {
	// Similar to exchanges, we'll need proper counting methods
	// For now, return 0
	// TODO: Add counting methods to SubAccountRepository
	return 0, nil
}