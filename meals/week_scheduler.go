package meals

// week_scheduler.go
//
// Runs a weekly job that ensures two full weeks of meal plan are always
// available: the current week and the week after.
//
// On every server startup the scheduler checks:
//   - Does the current week have rows for every scope that has repeating meals?
//     If not, generate them (handles a missed rollover or a fresh server start).
//   - Does the next week have rows? If not, generate them too.
//
// After the startup check, it sleeps until the next Monday midnight (local
// server time) and repeats, so new next-week rows are always created at the
// start of each new week.
//
// Call meals.StartWeekScheduler(pool) from main alongside the other jobs.

import (
	"context"
	"time"

	"weekly-shopping-app/database"
	"weekly-shopping-app/internal/logger"

	"github.com/jackc/pgx/v5/pgxpool"
)

// StartWeekScheduler starts the background goroutine that keeps two weeks of
// meal plan rows populated for every scope with repeating assignments.
func StartWeekScheduler(db *pgxpool.Pool) {
	go func() {
		runGeneration(db)

		for {
			time.Sleep(durationUntilNextMonday())
			runGeneration(db)
		}
	}()
}

// durationUntilNextMonday returns the time until the coming Monday at midnight
// (local server time). If right now is exactly Monday midnight, returns 7 days
// so the scheduler doesn't fire immediately after a fresh generation.
func durationUntilNextMonday() time.Duration {
	now := time.Now()
	daysUntilMonday := (int(time.Monday) - int(now.Weekday()) + 7) % 7
	if daysUntilMonday == 0 {
		daysUntilMonday = 7
	}
	nextMonday := time.Date(now.Year(), now.Month(), now.Day()+daysUntilMonday,
		0, 0, 0, 0, now.Location())
	return time.Until(nextMonday)
}

// runGeneration ensures the current week and the next week both have rows for
// every scope with repeating assignments. It is safe to call multiple times —
// GenerateNextWeek uses ON CONFLICT DO NOTHING so existing rows are not touched.
func runGeneration(db *pgxpool.Pool) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	thisWeek := database.WeekStart(time.Now())
	nextWeek := thisWeek.AddDate(0, 0, 7)

	scopes, err := database.DistinctScopes(ctx, db)
	if err != nil {
		logger.Error("week scheduler: could not fetch scopes", "err", logger.WithStack(err))
		return
	}
	if len(scopes) == 0 {
		return // nothing to do
	}

	var thisGenerated, nextGenerated int64
	for _, scope := range scopes {
		// Ensure current week exists.
		n, err := database.GenerateNextWeek(ctx, db, database.GenerateWeekParams{
			TargetWeekStart: thisWeek,
			HouseholdID:     scope.HouseholdID,
			UserID:          scope.UserID,
		})
		if err != nil {
			logger.Error("week scheduler: generating this week failed",
				"week", thisWeek.Format("2006-01-02"), "err", logger.WithStack(err))
		}
		thisGenerated += n

		// Ensure next week exists.
		n, err = database.GenerateNextWeek(ctx, db, database.GenerateWeekParams{
			TargetWeekStart: nextWeek,
			HouseholdID:     scope.HouseholdID,
			UserID:          scope.UserID,
		})
		if err != nil {
			logger.Error("week scheduler: generating next week failed",
				"week", nextWeek.Format("2006-01-02"), "err", logger.WithStack(err))
		}
		nextGenerated += n
	}

	if thisGenerated > 0 || nextGenerated > 0 {
		logger.Info("week scheduler: generation complete",
			"this_week", thisWeek.Format("2006-01-02"),
			"next_week", nextWeek.Format("2006-01-02"),
			"this_week_rows", thisGenerated,
			"next_week_rows", nextGenerated,
		)
	}
}
