package meals

// meal_plan_repeating_service.go
//
// Business logic + HTTP route handlers for the four-slot meal plan
// introduced in migration 009 (repeating_cook, temp_cook, repeating_meal, temp_meal)
// and extended in migration 010 (week_start DATE column).
//
// Routes added to the existing /meals router:
//
//   GET  /meals/plan/v2               — this week's plan with all four slots
//   GET  /meals/plan/v2/next          — next week's plan with all four slots
//   POST /meals/plan/v2/set           — upsert a day for a specific week
//   POST /meals/plan/v2/rollover      — promote repeating → effective, clear temps
//
// The week to read/write is always identified by its Monday date (week_start).
// Callers pass week_offset: 0 = this week, 1 = next week.

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"weekly-shopping-app/authentication"
	"weekly-shopping-app/database"
	"weekly-shopping-app/internal/api/httpx"
	"weekly-shopping-app/internal/logger"

	sqlc "weekly-shopping-app/database/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ── Business logic ────────────────────────────────────────────────────────────

// getMealPlanWeek loads one week's plan and maps the four-slot rows into
// MealPlanDayResponseV2 values. weekStart must be a Monday.
func getMealPlanWeek(ctx context.Context, db *pgxpool.Pool, userID, householdID int32, weekStart time.Time) ([]MealPlanDayResponseV2, error) {
	rows, err := database.GetWeekPlan(ctx, db, database.GetWeekPlanParams{
		WeekStart:   weekStart,
		HouseholdID: pgtype.Int4{Int32: householdID, Valid: householdID != 0},
		UserID:      pgtype.Int4{Int32: userID, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	out := make([]MealPlanDayResponseV2, 0, len(rows))
	for _, r := range rows {
		day := MealPlanDayResponseV2{DayName: r.DayName}

		// ── Effective meal ──
		if r.MealID.Valid {
			id := r.MealID.Int32
			day.MealID = &id
		}
		if r.MealName.Valid {
			day.MealName = r.MealName.String
		}
		if r.MealDescription.Valid {
			day.MealDescription = r.MealDescription.String
		}
		if r.DefaultPortions.Valid {
			day.DefaultPortions = r.DefaultPortions.Int32
		}

		// ── Effective cook ──
		if r.CookName.Valid {
			day.Cook = &CookResponse{
				ID:       r.CookUserID.Int32,
				Name:     r.CookName.String,
				Username: r.CookUsername.String,
			}
		}

		// ── Repeating slots ──
		if r.RepeatingMealID.Valid {
			name := ""
			if r.RepeatingMealName.Valid {
				name = r.RepeatingMealName.String
			}
			day.RepeatingMeal = &MealSlot{ID: r.RepeatingMealID.Int32, Name: name}
		}
		if r.RepeatingCookUserID.Valid {
			name, username := "", ""
			if r.RepeatingCookName.Valid {
				name = r.RepeatingCookName.String
			}
			if r.RepeatingCookUsername.Valid {
				username = r.RepeatingCookUsername.String
			}
			day.RepeatingCook = &CookSlot{ID: r.RepeatingCookUserID.Int32, Name: name, Username: username}
		}

		// ── Temp slots ──
		if r.TempMealID.Valid {
			name := ""
			if r.TempMealName.Valid {
				name = r.TempMealName.String
			}
			day.TempMeal = &MealSlot{ID: r.TempMealID.Int32, Name: name}
		}
		if r.TempCookUserID.Valid {
			name, username := "", ""
			if r.TempCookName.Valid {
				name = r.TempCookName.String
			}
			if r.TempCookUsername.Valid {
				username = r.TempCookUsername.String
			}
			day.TempCook = &CookSlot{ID: r.TempCookUserID.Int32, Name: name, Username: username}
		}

		out = append(out, day)
	}
	return out, nil
}

// setMealPlanDayV2 upserts a single day within the week identified by
// input.WeekOffset (0 = this week, 1 = next week).
func setMealPlanDayV2(ctx context.Context, db *pgxpool.Pool, userID int32, input SetMealPlanInputV2) (*MealPlanDayResponseV2, error) {
	hid, uid := planScope(userID, input.HouseholdID, input.Scope)

	weekStart := database.WeekStart(time.Now())
	if input.WeekOffset == 1 {
		weekStart = weekStart.AddDate(0, 0, 7)
	}

	// Resolve effective meal: explicit override > temp > repeating
	effectiveMealID := input.MealID
	if effectiveMealID == 0 {
		if input.TempMealID != 0 {
			effectiveMealID = input.TempMealID
		} else {
			effectiveMealID = input.RepeatingMealID
		}
	}

	// Resolve effective cook: explicit override > temp > repeating
	effectiveCookID := input.CookUserID
	if effectiveCookID == 0 {
		if input.TempCookUserID != 0 {
			effectiveCookID = input.TempCookUserID
		} else {
			effectiveCookID = input.RepeatingCookUserID
		}
	}

	if err := database.SetWeekPlanDay(ctx, db, database.SetWeekPlanDayParams{
		DayName:             input.DayName,
		WeekStart:           weekStart,
		MealID:              nullableInt4(effectiveMealID),
		CookUserID:          nullableInt4(effectiveCookID),
		RepeatingMealID:     nullableInt4(input.RepeatingMealID),
		RepeatingCookUserID: nullableInt4(input.RepeatingCookUserID),
		TempMealID:          nullableInt4(input.TempMealID),
		TempCookUserID:      nullableInt4(input.TempCookUserID),
		HouseholdID:         hid,
		UserID:              uid,
	}); err != nil {
		return nil, err
	}

	// If an effective meal was resolved, sync ingredients to shopping list.
	// Repeating plans have no user selection for option groups, so none are included.
	if effectiveMealID != 0 {
		if addErr := addMealIngredientsToShoppingList(ctx, db, effectiveMealID, hid, uid, nil); addErr != nil {
			logger.Warn("meal plan v2: could not sync ingredients", "err", addErr)
		}
	}

	// Re-fetch the full day to return enriched names
	days, err := getMealPlanWeek(ctx, db, userID, input.HouseholdID, weekStart)
	if err != nil {
		return nil, err
	}
	for _, d := range days {
		if d.DayName == input.DayName {
			return &d, nil
		}
	}
	return nil, fmt.Errorf("day %q not found after upsert", input.DayName)
}

// rolloverWeek promotes repeating values into the effective slots and clears
// all temp overrides for a given household or user scope.
// It operates on the current week's rows only.
func rolloverWeek(ctx context.Context, db *pgxpool.Pool, userID int32, input ClearTempInput) ([]MealPlanDayResponseV2, error) {
	q := sqlc.New(db)
	hid, uid := planScope(userID, input.HouseholdID, input.Scope)

	if input.DayName != "" {
		if err := q.ClearTempOverridesForDay(ctx, sqlc.ClearTempOverridesForDayParams{
			DayName:     input.DayName,
			HouseholdID: hid,
			UserID:      uid,
		}); err != nil {
			return nil, err
		}
	} else {
		if err := q.ClearAllTempOverrides(ctx, sqlc.ClearAllTempOverridesParams{
			HouseholdID: hid,
			UserID:      uid,
		}); err != nil {
			return nil, err
		}
	}

	weekStart := database.WeekStart(time.Now())
	return getMealPlanWeek(ctx, db, userID, input.HouseholdID, weekStart)
}

// ── HTTP route registration ───────────────────────────────────────────────────

// RegisterRepeatingPlanRoutes adds the v2 plan endpoints to an existing meals router.
func RegisterRepeatingPlanRoutes(r *Router, db *pgxpool.Pool) {

	// GET /meals/plan/v2 — this week's plan with all four slots
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[struct{}]{
		Path: "/plan/v2", Method: "GET", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
			return func(req *http.Request, _ struct{}) (any, error) {
				sess, err := authentication.SessionFromContext(req)
				if err != nil {
					return nil, err
				}
				weekStart := database.WeekStart(time.Now())
				return getMealPlanWeek(req.Context(), db, sess.UserID, sess.FirstHouseholdID(), weekStart)
			}
		},
	})

	// GET /meals/plan/v2/next — next week's plan with all four slots
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[struct{}]{
		Path: "/plan/v2/next", Method: "GET", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
			return func(req *http.Request, _ struct{}) (any, error) {
				sess, err := authentication.SessionFromContext(req)
				if err != nil {
					return nil, err
				}
				weekStart := database.WeekStart(time.Now()).AddDate(0, 0, 7)
				return getMealPlanWeek(req.Context(), db, sess.UserID, sess.FirstHouseholdID(), weekStart)
			}
		},
	})

	// POST /meals/plan/v2/set — upsert a day (week_offset: 0=this week, 1=next week)
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[SetMealPlanInputV2]{
		Path: "/plan/v2/set", Method: "POST", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, SetMealPlanInputV2) (any, error) {
			return func(req *http.Request, input SetMealPlanInputV2) (any, error) {
				sess, err := authentication.SessionFromContext(req)
				if err != nil {
					return nil, err
				}
				return setMealPlanDayV2(req.Context(), db, sess.UserID, input)
			}
		},
	})

	// POST /meals/plan/v2/rollover — clear temps, promote repeating → effective
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[ClearTempInput]{
		Path: "/plan/v2/rollover", Method: "POST", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, ClearTempInput) (any, error) {
			return func(req *http.Request, input ClearTempInput) (any, error) {
				sess, err := authentication.SessionFromContext(req)
				if err != nil {
					return nil, err
				}
				return rolloverWeek(req.Context(), db, sess.UserID, input)
			}
		},
	})
}
