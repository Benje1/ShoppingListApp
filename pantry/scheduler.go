package pantry

import (
	"context"
	"time"

	"weekly-shopping-app/internal/logger"

	"github.com/jackc/pgx/v5/pgxpool"
)

// StartExpiryScheduler runs the pantry expiry job every eight hours.
// It marks items as 'expiring_soon' (within 2 days) or 'expired' (past date).
// Call this once from main after the DB pool is ready. It stops when ctx is cancelled.
func StartExpiryScheduler(ctx context.Context, db *pgxpool.Pool) {
	go func() {
		// Run once immediately on startup to catch anything that expired while the
		// server was down, then tick every eight hours.
		runJob(db)
		ticker := time.NewTicker(8 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runJob(db)
			}
		}
	}()
}

func runJob(db *pgxpool.Pool) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	n, err := RunExpiryJob(ctx, db)
	if err != nil {
		logger.Error("pantry expiry job failed", "err", err)
		return
	}
	if n > 0 {
		logger.Info("pantry expiry job completed", "items_updated", n)
	}
}
