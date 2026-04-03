package meals

// meal_plan_repeating_service.go
//
// Business logic + HTTP route handlers for the four-slot meal plan
// introduced in migration 009 (repeating_cook, temp_cook, repeating_meal, temp_meal).
//
// Routes added to the existing /meals router:
//
//   GET  /meals/plan/v2          — full week plan with all four slots
//   POST /meals/plan/v2/set      — upsert a day (all four slots + effective values)
//   POST /meals/plan/v2/rollover — promote repeating → effective, clear all temp fields

import (
	"context"
	"fmt"
	"net/http"

	"weekly-shopping-app/authentication"
	"weekly-shopping-app/internal/api/httpx"

	sqlc "weekly-shopping-app/database/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ── Business logic ────────────────────────────────────────────────────────────

// getMealPlanFullV2 loads the week plan and maps the four-slot rows into
// MealPlanDayResponseV2 values.
func getMealPlanFullV2(ctx context.Context, db *pgxpool.Pool, userID, householdID int32) ([]MealPlanDayResponseV2, error) {
	q := sqlc.New(db)
	rows, err := q.GetMealPlanFullV2(ctx, sqlc.GetMealPlanFullV2Params{
		HouseholdID: pgtype.Int4{Int32: householdID, Valid: householdID != 0},
		UserID:      pgtype.Int4{Int32: userID, Valid: true}},
	)
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

// setMealPlanDayV2 upserts a single day with all four slots.
// The effective meal_id / cook_user_id are derived from the temp-beats-repeating
// priority rule so the existing shopping-list logic keeps working unchanged.
func setMealPlanDayV2(ctx context.Context, db *pgxpool.Pool, userID int32, input SetMealPlanInputV2) (*MealPlanDayResponseV2, error) {
	q := sqlc.New(db)
	hid, uid := planScope(userID, input.HouseholdID, input.Scope)

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

	_, err := q.SetMealPlanDayV2(ctx, sqlc.SetMealPlanDayV2Params{
		DayName:             input.DayName,
		MealID:              nullableInt4(effectiveMealID),
		CookUserID:          nullableInt4(effectiveCookID),
		RepeatingMealID:     nullableInt4(input.RepeatingMealID),
		RepeatingCookUserID: nullableInt4(input.RepeatingCookUserID),
		TempMealID:          nullableInt4(input.TempMealID),
		TempCookUserID:      nullableInt4(input.TempCookUserID),
		HouseholdID:         hid,
		UserID:              uid,
	})
	if err != nil {
		return nil, err
	}

	// If an effective meal was resolved, sync ingredients to shopping list
	if effectiveMealID != 0 {
		if addErr := addMealIngredientsToShoppingList(ctx, db, effectiveMealID, hid, uid); addErr != nil {
			fmt.Printf("[meal plan v2] warning: could not sync ingredients: %v\n", addErr)
		}
	}

	// Re-fetch the full day to return enriched names
	days, err := getMealPlanFullV2(ctx, db, userID, input.HouseholdID)
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
func rolloverWeek(ctx context.Context, db *pgxpool.Pool, userID int32, input ClearTempInput) ([]MealPlanDayResponseV2, error) {
	q := sqlc.New(db)
	hid, uid := planScope(userID, input.HouseholdID, input.Scope)

	if input.DayName != "" {
		// Single-day rollover
		if err := q.ClearTempOverridesForDay(ctx, sqlc.ClearTempOverridesForDayParams{DayName: input.DayName, HouseholdID: hid, UserID: uid}); err != nil {
			return nil, err
		}
	} else {
		// Full-week rollover
		if err := q.ClearAllTempOverrides(ctx, sqlc.ClearAllTempOverridesParams{HouseholdID: hid, UserID: uid}); err != nil {
			return nil, err
		}
	}

	return getMealPlanFullV2(ctx, db, userID, input.HouseholdID)
}

// ── HTTP route registration ───────────────────────────────────────────────────

// RegisterRepeatingPlanRoutes adds the v2 plan endpoints to an existing meals router.
// Call this from registerPlanAndCookRoutes (or RegisterMealRoutes) after the v1 routes.
func RegisterRepeatingPlanRoutes(r *Router, db *pgxpool.Pool) {

	// GET /meals/plan/v2 — full week with all four slots
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[struct{}]{
		Path: "/plan/v2", Method: "GET", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
			return func(req *http.Request, _ struct{}) (any, error) {
				sess, err := authentication.SessionFromContext(req)
				if err != nil {
					return nil, err
				}
				return getMealPlanFullV2(req.Context(), db, sess.UserID, sess.FirstHouseholdID())
			}
		},
	})

	// POST /meals/plan/v2/set — upsert a day with all four slots
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
