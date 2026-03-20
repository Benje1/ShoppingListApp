package pantry

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// StartExpiryScheduler runs the pantry expiry job every hour.
// It marks items as 'expiring_soon' (within 2 days) or 'expired' (past date).
// Call this once from main after the DB pool is ready.
func StartExpiryScheduler(db *pgxpool.Pool) {
	go func() {
		// Run once immediately on startup to catch anything that expired while the
		// server was down, then tick every hour.
		runJob(db)
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			runJob(db)
		}
	}()
}

func runJob(db *pgxpool.Pool) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	n, err := RunExpiryJob(ctx, db)
	if err != nil {
		fmt.Printf("[pantry expiry] error: %v\n", err)
		return
	}
	if n > 0 {
		fmt.Printf("[pantry expiry] updated %d item(s)\n", n)
	}
}
