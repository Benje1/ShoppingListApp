package meals

// week_scheduler.go
//
// Runs a weekly rollover job that promotes repeating meal/cook assignments into
// the effective slots and clears one-off temp overrides — giving every scope
// (household or personal) a fresh week based on their standing plan.
//
// The job fires at the next Monday midnight (local server time) and then every
// 7 days thereafter. On startup it also checks whether the current week has
// already been rolled over; if not, it runs immediately so a server restart
// never leaves the plan stale.
//
// Call meals.StartWeekScheduler(pool) from main after the DB pool is ready,
// alongside pantry.StartExpiryScheduler.

import (
	"context"
	"time"

	"weekly-shopping-app/internal/logger"

	"github.com/jackc/pgx/v5/pgxpool"
)

// StartWeekScheduler starts the background goroutine that rolls over the meal
// plan every Monday. It runs once immediately on startup if this week has not
// yet been rolled over, then ticks every 7 days aligned to the next Monday.
func StartWeekScheduler(db *pgxpool.Pool) {
	go func() {
		// On startup, roll over immediately if this week hasn't been done yet.
		if needsRollover(db) {
			runWeeklyRollover(db)
		}

		// Sleep until the next Monday 00:00 local time, then tick weekly.
		for {
			time.Sleep(durationUntilNextMonday())
			runWeeklyRollover(db)
		}
	}()
}

// durationUntilNextMonday returns how long until the coming Monday at midnight
// in the local server timezone. If today is Monday and it is exactly midnight,
// it returns 7 days so we don't spin immediately after a fresh rollover.
func durationUntilNextMonday() time.Duration {
	now := time.Now()
	daysUntilMonday := (int(time.Monday) - int(now.Weekday()) + 7) % 7
	if daysUntilMonday == 0 {
		daysUntilMonday = 7 // already Monday — wait for next one
	}
	nextMonday := time.Date(now.Year(), now.Month(), now.Day()+daysUntilMonday,
		0, 0, 0, 0, now.Location())
	return time.Until(nextMonday)
}

// needsRollover reports whether the meal plan has any rows whose effective
// meal_id or cook_user_id still reflects a previous week. We detect this by
// checking whether any row was last updated before the start of the current
// ISO week (Monday 00:00 local time). If every row was touched this week we
// consider the rollover already done.
func needsRollover(db *pgxpool.Pool) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start of the current ISO week (Monday 00:00 local time).
	now := time.Now()
	daysFromMonday := (int(now.Weekday()) - int(time.Monday) + 7) % 7
	weekStart := time.Date(now.Year(), now.Month(), now.Day()-daysFromMonday,
		0, 0, 0, 0, now.Location())

	// If any row exists and was last updated before this Monday, we need a rollover.
	const q = `
		SELECT EXISTS (
			SELECT 1 FROM meal_plan
			WHERE (repeating_meal_id IS NOT NULL OR repeating_cook_user_id IS NOT NULL)
			  AND updated_at < $1
		)`
	var needed bool
	if err := db.QueryRow(ctx, q, weekStart).Scan(&needed); err != nil {
		// On error, default to running the job — better to roll over unnecessarily
		// than to leave the plan stale.
		logger.Warn("week scheduler: could not check rollover status, running anyway", "err", err)
		return true
	}
	return needed
}

// runWeeklyRollover promotes every scope's repeating values into the effective
// meal_id / cook_user_id columns and clears all temp overrides in one UPDATE.
func runWeeklyRollover(db *pgxpool.Pool) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	const q = `
		UPDATE meal_plan
		SET temp_meal_id      = NULL,
		    temp_cook_user_id = NULL,
		    meal_id           = COALESCE(repeating_meal_id, meal_id),
		    cook_user_id      = COALESCE(repeating_cook_user_id, cook_user_id),
		    updated_at        = now()
		WHERE repeating_meal_id IS NOT NULL
		   OR repeating_cook_user_id IS NOT NULL`

	tag, err := db.Exec(ctx, q)
	if err != nil {
		logger.Error("week scheduler: rollover failed", "err", err)
		return
	}
	logger.Info("week scheduler: weekly rollover complete", "rows_updated", tag.RowsAffected())
}
