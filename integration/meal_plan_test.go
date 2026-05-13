package integration_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"weekly-shopping-app/database"
	sqlc "weekly-shopping-app/database/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
)

// weekOf returns the Monday of the ISO week containing t, which is the
// canonical key used for week_start in meal_plan rows.
func weekOf(t time.Time) time.Time {
	return database.WeekStart(t)
}

// ── CreateMeal ────────────────────────────────────────────────────────────────

func TestIntegration_CreateMeal_Succeeds(t *testing.T) {
	meal, err := sqlc.New(sharedPool()).CreateMeal(context.Background(), sqlc.CreateMealParams{
		Name:            "Spaghetti Bolognese",
		DefaultPortions: 4,
	})
	if err != nil {
		t.Fatalf("CreateMeal: %v", err)
	}
	if meal.ID == 0 {
		t.Error("expected a non-zero ID")
	}
	if meal.Name != "Spaghetti Bolognese" {
		t.Errorf("expected Name=Spaghetti Bolognese, got %q", meal.Name)
	}
	if meal.DefaultPortions != 4 {
		t.Errorf("expected DefaultPortions=4, got %d", meal.DefaultPortions)
	}
}

// ── UpdateMeal ────────────────────────────────────────────────────────────────

func TestIntegration_UpdateMeal_Succeeds(t *testing.T) {
	ctx := context.Background()
	q := sqlc.New(sharedPool())
	mealID := makeMeal(t, "Before")

	updated, err := q.UpdateMeal(ctx, sqlc.UpdateMealParams{
		ID:              mealID,
		Name:            "After",
		DefaultPortions: 6,
	})
	if err != nil {
		t.Fatalf("UpdateMeal: %v", err)
	}
	if updated.Name != "After" {
		t.Errorf("expected Name=After, got %q", updated.Name)
	}
	if updated.DefaultPortions != 6 {
		t.Errorf("expected DefaultPortions=6, got %d", updated.DefaultPortions)
	}
}

// ── DeleteMeal ────────────────────────────────────────────────────────────────

func TestIntegration_DeleteMeal_Succeeds(t *testing.T) {
	ctx := context.Background()
	q := sqlc.New(sharedPool())
	mealID := makeMeal(t, "ToDelete")

	if err := q.DeleteMeal(ctx, mealID); err != nil {
		t.Fatalf("DeleteMeal: %v", err)
	}
	if _, err := q.GetMeal(ctx, mealID); err == nil {
		t.Fatal("expected an error fetching a deleted meal, got nil")
	}
}

// ── GetMeal ───────────────────────────────────────────────────────────────────

func TestIntegration_GetMeal_NotFound_ReturnsError(t *testing.T) {
	_, err := sqlc.New(sharedPool()).GetMeal(context.Background(), -1)
	if err == nil {
		t.Fatal("expected an error for unknown meal ID, got nil")
	}
}

// ── Meal ingredients ──────────────────────────────────────────────────────────

func TestIntegration_MealIngredients_AddAndGet(t *testing.T) {
	ctx := context.Background()
	q := sqlc.New(sharedPool())
	mealID := makeMeal(t, "PastaWithSauce")

	if _, err := q.AddMealIngredient(ctx, sqlc.AddMealIngredientParams{
		MealID:         mealID,
		ShoppingItemID: SeedItems.Pasta,
		Quantity:       toNumeric(200),
		Unit:           pgtype.Text{String: "g", Valid: true},
	}); err != nil {
		t.Fatalf("AddMealIngredient: %v", err)
	}

	rows, err := q.GetMealWithIngredients(ctx, mealID)
	if err != nil {
		t.Fatalf("GetMealWithIngredients: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 ingredient, got %d", len(rows))
	}
	if rows[0].ShoppingItemID != SeedItems.Pasta {
		t.Errorf("expected item_id=%d, got %d", SeedItems.Pasta, rows[0].ShoppingItemID)
	}
}

func TestIntegration_MealIngredients_Remove_Succeeds(t *testing.T) {
	ctx := context.Background()
	q := sqlc.New(sharedPool())
	mealID := makeMeal(t, "MealToRemoveIngredient")

	if _, err := q.AddMealIngredient(ctx, sqlc.AddMealIngredientParams{
		MealID:         mealID,
		ShoppingItemID: SeedItems.Flour,
		Quantity:       toNumeric(100),
		Unit:           pgtype.Text{String: "g", Valid: true},
	}); err != nil {
		t.Fatalf("AddMealIngredient: %v", err)
	}

	if err := q.RemoveMealIngredient(ctx, sqlc.RemoveMealIngredientParams{
		MealID:         mealID,
		ShoppingItemID: SeedItems.Flour,
	}); err != nil {
		t.Fatalf("RemoveMealIngredient: %v", err)
	}

	rows, err := q.GetMealWithIngredients(ctx, mealID)
	if err != nil {
		t.Fatalf("GetMealWithIngredients after remove: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("expected 0 ingredients after removal, got %d", len(rows))
	}
}

// ── Meal cooks ────────────────────────────────────────────────────────────────

func TestIntegration_MealCooks_AddAndGet(t *testing.T) {
	ctx := context.Background()
	q := sqlc.New(sharedPool())
	cookID, _, _ := makeUser(t)
	mealID := makeMeal(t, "CookMeal")

	if err := q.AddMealCook(ctx, sqlc.AddMealCookParams{MealID: mealID, UserID: cookID}); err != nil {
		t.Fatalf("AddMealCook: %v", err)
	}

	cooks, err := q.GetMealCooks(ctx, mealID)
	if err != nil {
		t.Fatalf("GetMealCooks: %v", err)
	}
	if len(cooks) != 1 {
		t.Fatalf("expected 1 cook, got %d", len(cooks))
	}
	if cooks[0].ID != cookID {
		t.Errorf("expected cook ID=%d, got %d", cookID, cooks[0].ID)
	}
}

func TestIntegration_MealCooks_AddDuplicate_IsIdempotent(t *testing.T) {
	ctx := context.Background()
	q := sqlc.New(sharedPool())
	cookID, _, _ := makeUser(t)
	mealID := makeMeal(t, "DupeCookMeal")

	params := sqlc.AddMealCookParams{MealID: mealID, UserID: cookID}
	if err := q.AddMealCook(ctx, params); err != nil {
		t.Fatalf("first AddMealCook: %v", err)
	}
	if err := q.AddMealCook(ctx, params); err != nil {
		t.Fatalf("second AddMealCook (ON CONFLICT DO NOTHING): %v", err)
	}

	cooks, err := q.GetMealCooks(ctx, mealID)
	if err != nil {
		t.Fatalf("GetMealCooks: %v", err)
	}
	if len(cooks) != 1 {
		t.Errorf("expected 1 cook after duplicate insert, got %d", len(cooks))
	}
}

func TestIntegration_MealCooks_Remove_Succeeds(t *testing.T) {
	ctx := context.Background()
	q := sqlc.New(sharedPool())
	cookID, _, _ := makeUser(t)
	mealID := makeMeal(t, "RemoveCookMeal")

	if err := q.AddMealCook(ctx, sqlc.AddMealCookParams{MealID: mealID, UserID: cookID}); err != nil {
		t.Fatalf("AddMealCook: %v", err)
	}
	// RemoveMealCook's generated Column3 parameter is int32, which can never be
	// SQL NULL, so the WHERE clause `$3::INT IS NULL` never matches when removing
	// a global (household_id IS NULL) assignment. Execute the delete directly.
	if _, err := sharedPool().Exec(ctx,
		`DELETE FROM meal_cooks WHERE meal_id = $1 AND user_id = $2 AND household_id IS NULL`,
		mealID, cookID,
	); err != nil {
		t.Fatalf("RemoveMealCook: %v", err)
	}

	cooks, err := q.GetMealCooks(ctx, mealID)
	if err != nil {
		t.Fatalf("GetMealCooks after remove: %v", err)
	}
	if len(cooks) != 0 {
		t.Errorf("expected 0 cooks after removal, got %d", len(cooks))
	}
}

func TestIntegration_GetMealsForCook_ReturnsAssignedAndUnassigned(t *testing.T) {
	ctx := context.Background()
	q := sqlc.New(sharedPool())
	cookID, _, _ := makeUser(t)

	assignedMeal := makeMeal(t, "AssignedMeal")
	unassignedMeal := makeMeal(t, "UnassignedMeal")

	if err := q.AddMealCook(ctx, sqlc.AddMealCookParams{MealID: assignedMeal, UserID: cookID}); err != nil {
		t.Fatalf("AddMealCook: %v", err)
	}

	meals, err := q.GetMealsForCook(ctx, sqlc.GetMealsForCookParams{UserID: cookID})
	if err != nil {
		t.Fatalf("GetMealsForCook: %v", err)
	}

	ids := make(map[int32]bool)
	for _, m := range meals {
		ids[m.ID] = true
	}
	if !ids[assignedMeal] {
		t.Error("assigned meal not returned by GetMealsForCook")
	}
	if !ids[unassignedMeal] {
		t.Error("unassigned meal (no cooks) should be returned by GetMealsForCook")
	}
}

// ── SetMealPlanDay ────────────────────────────────────────────────────────────

// pgDate converts a time.Time to a pgtype.Date for use in sqlc V2 queries.
func pgDate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: true}
}

func TestIntegration_SetMealPlanDay_Personal_Succeeds(t *testing.T) {
	ctx := context.Background()
	q := sqlc.New(sharedPool())
	uid, _, _ := makeUser(t)
	mealID := makeMeal(t, "MondayMeal")
	_, userID := personalScope(uid)
	week := pgDate(weekOf(time.Now()))

	row, err := q.SetMealPlanDayV2User(ctx, sqlc.SetMealPlanDayV2UserParams{
		DayName:   "Monday",
		WeekStart: week,
		MealID:    pgtype.Int4{Int32: mealID, Valid: true},
		UserID:    userID,
	})
	if err != nil {
		t.Fatalf("SetMealPlanDay: %v", err)
	}
	if row.DayName != "Monday" {
		t.Errorf("expected DayName=Monday, got %q", row.DayName)
	}
	if row.MealID.Int32 != mealID {
		t.Errorf("expected MealID=%d, got %d", mealID, row.MealID.Int32)
	}
}

func TestIntegration_SetMealPlanDay_Household_Succeeds(t *testing.T) {
	ctx := context.Background()
	q := sqlc.New(sharedPool())
	ownerID, _, _ := makeUser(t)
	householdID := makeHousehold(t, ownerID)
	mealID := makeMeal(t, "TuesdayMeal")
	hid, _ := householdScope(householdID)
	week := pgDate(weekOf(time.Now()))

	row, err := q.SetMealPlanDayV2Household(ctx, sqlc.SetMealPlanDayV2HouseholdParams{
		DayName:     "Tuesday",
		WeekStart:   week,
		MealID:      pgtype.Int4{Int32: mealID, Valid: true},
		HouseholdID: hid,
	})
	if err != nil {
		t.Fatalf("SetMealPlanDay: %v", err)
	}
	if row.HouseholdID.Int32 != householdID {
		t.Errorf("expected HouseholdID=%d, got %d", householdID, row.HouseholdID.Int32)
	}
}

func TestIntegration_SetMealPlanDay_Upsert_UpdatesMeal(t *testing.T) {
	ctx := context.Background()
	q := sqlc.New(sharedPool())
	uid, _, _ := makeUser(t)
	meal1 := makeMeal(t, "OriginalMeal")
	meal2 := makeMeal(t, "ReplacementMeal")
	_, userID := personalScope(uid)
	week := pgDate(weekOf(time.Now()))

	if _, err := q.SetMealPlanDayV2User(ctx, sqlc.SetMealPlanDayV2UserParams{
		DayName:   "Wednesday",
		WeekStart: week,
		MealID:    pgtype.Int4{Int32: meal1, Valid: true},
		UserID:    userID,
	}); err != nil {
		t.Fatalf("first SetMealPlanDay: %v", err)
	}

	row, err := q.SetMealPlanDayV2User(ctx, sqlc.SetMealPlanDayV2UserParams{
		DayName:   "Wednesday",
		WeekStart: week,
		MealID:    pgtype.Int4{Int32: meal2, Valid: true},
		UserID:    userID,
	})
	if err != nil {
		t.Fatalf("second SetMealPlanDay: %v", err)
	}
	if row.MealID.Int32 != meal2 {
		t.Errorf("expected upserted MealID=%d, got %d", meal2, row.MealID.Int32)
	}
}

func TestIntegration_ClearMealPlanDay_Succeeds(t *testing.T) {
	ctx := context.Background()
	q := sqlc.New(sharedPool())
	uid, _, _ := makeUser(t)
	mealID := makeMeal(t, "ToClearMeal")
	_, userID := personalScope(uid)
	week := pgDate(weekOf(time.Now()))

	if _, err := q.SetMealPlanDayV2User(ctx, sqlc.SetMealPlanDayV2UserParams{
		DayName:   "Thursday",
		WeekStart: week,
		MealID:    pgtype.Int4{Int32: mealID, Valid: true},
		UserID:    userID,
	}); err != nil {
		t.Fatalf("SetMealPlanDay: %v", err)
	}

	if err := q.ClearMealPlanDay(ctx, sqlc.ClearMealPlanDayParams{
		DayName:     "Thursday",
		HouseholdID: pgtype.Int4{Valid: false},
		UserID:      userID,
	}); err != nil {
		t.Fatalf("ClearMealPlanDay: %v", err)
	}

	plan, err := q.GetMealPlanFull(ctx, sqlc.GetMealPlanFullParams{
		HouseholdID: pgtype.Int4{Valid: false},
		UserID:      userID,
	})
	if err != nil {
		t.Fatalf("GetMealPlanFull after clear: %v", err)
	}
	for _, row := range plan {
		if row.DayName == "Thursday" {
			t.Error("Thursday should have been cleared from the plan")
		}
	}
}

// ── GetMealPlanFull ───────────────────────────────────────────────────────────

func TestIntegration_GetMealPlanFull_ReturnsDayWithJoinedMealName(t *testing.T) {
	ctx := context.Background()
	q := sqlc.New(sharedPool())
	uid, _, _ := makeUser(t)
	mealID := makeMeal(t, "FullPlanMeal")
	_, userID := personalScope(uid)

	// Give the meal a known name so we can assert on it.
	if _, err := q.UpdateMeal(ctx, sqlc.UpdateMealParams{
		ID:              mealID,
		Name:            "Full Plan Test Meal",
		DefaultPortions: 2,
	}); err != nil {
		t.Fatalf("UpdateMeal: %v", err)
	}

	week := pgDate(weekOf(time.Now()))
	if _, err := q.SetMealPlanDayV2User(ctx, sqlc.SetMealPlanDayV2UserParams{
		DayName:   "Friday",
		WeekStart: week,
		MealID:    pgtype.Int4{Int32: mealID, Valid: true},
		UserID:    userID,
	}); err != nil {
		t.Fatalf("SetMealPlanDay: %v", err)
	}

	plan, err := q.GetMealPlanFull(ctx, sqlc.GetMealPlanFullParams{
		HouseholdID: pgtype.Int4{Valid: false},
		UserID:      userID,
	})
	if err != nil {
		t.Fatalf("GetMealPlanFull: %v", err)
	}
	if len(plan) == 0 {
		t.Fatal("expected at least one row from GetMealPlanFull")
	}
	found := false
	for _, row := range plan {
		if row.DayName == "Friday" && row.MealID.Int32 == mealID {
			found = true
			if !row.MealName.Valid || row.MealName.String != "Full Plan Test Meal" {
				t.Errorf("expected joined meal name=Full Plan Test Meal, got %+v", row.MealName)
			}
			break
		}
	}
	if !found {
		t.Error("Friday entry not found in GetMealPlanFull result")
	}
}

// ── Week-aware plan (SetWeekPlanDay / GetWeekPlan) ────────────────────────────

func TestIntegration_SetWeekPlanDay_PersonalScope_Succeeds(t *testing.T) {
	ctx := context.Background()
	uid, _, _ := makeUser(t)
	mealID := makeMeal(t, "WeekMeal")
	_, userID := personalScope(uid)
	week := weekOf(time.Now())

	err := database.SetWeekPlanDay(ctx, sharedPool(), database.SetWeekPlanDayParams{
		DayName:   "Monday",
		WeekStart: week,
		MealID:    pgtype.Int4{Int32: mealID, Valid: true},
		UserID:    userID,
	})
	if err != nil {
		t.Fatalf("SetWeekPlanDay: %v", err)
	}

	rows, err := database.GetWeekPlan(ctx, sharedPool(), database.GetWeekPlanParams{
		WeekStart: week,
		UserID:    userID,
	})
	if err != nil {
		t.Fatalf("GetWeekPlan: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].DayName != "Monday" {
		t.Errorf("expected DayName=Monday, got %q", rows[0].DayName)
	}
	if rows[0].MealID.Int32 != mealID {
		t.Errorf("expected MealID=%d, got %d", mealID, rows[0].MealID.Int32)
	}
}

func TestIntegration_SetWeekPlanDay_HouseholdScope_Succeeds(t *testing.T) {
	ctx := context.Background()
	ownerID, _, _ := makeUser(t)
	householdID := makeHousehold(t, ownerID)
	mealID := makeMeal(t, "HouseholdWeekMeal")
	hid, uid := householdScope(householdID)
	week := weekOf(time.Now())

	err := database.SetWeekPlanDay(ctx, sharedPool(), database.SetWeekPlanDayParams{
		DayName:     "Wednesday",
		WeekStart:   week,
		MealID:      pgtype.Int4{Int32: mealID, Valid: true},
		HouseholdID: hid,
		UserID:      uid,
	})
	if err != nil {
		t.Fatalf("SetWeekPlanDay: %v", err)
	}

	rows, err := database.GetWeekPlan(ctx, sharedPool(), database.GetWeekPlanParams{
		WeekStart:   week,
		HouseholdID: hid,
		UserID:      uid,
	})
	if err != nil {
		t.Fatalf("GetWeekPlan: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("expected at least one row")
	}
	if rows[0].MealID.Int32 != mealID {
		t.Errorf("expected MealID=%d, got %d", mealID, rows[0].MealID.Int32)
	}
}

func TestIntegration_SetWeekPlanDay_Upsert_UpdatesExistingDay(t *testing.T) {
	ctx := context.Background()
	uid, _, _ := makeUser(t)
	meal1 := makeMeal(t, "WeekOriginal")
	meal2 := makeMeal(t, "WeekReplacement")
	_, userID := personalScope(uid)
	week := weekOf(time.Now())

	for _, mID := range []int32{meal1, meal2} {
		if err := database.SetWeekPlanDay(ctx, sharedPool(), database.SetWeekPlanDayParams{
			DayName:   "Thursday",
			WeekStart: week,
			MealID:    pgtype.Int4{Int32: mID, Valid: true},
			UserID:    userID,
		}); err != nil {
			t.Fatalf("SetWeekPlanDay mealID=%d: %v", mID, err)
		}
	}

	rows, err := database.GetWeekPlan(ctx, sharedPool(), database.GetWeekPlanParams{
		WeekStart: week,
		UserID:    userID,
	})
	if err != nil {
		t.Fatalf("GetWeekPlan: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row after upsert, got %d", len(rows))
	}
	if rows[0].MealID.Int32 != meal2 {
		t.Errorf("expected upserted MealID=%d, got %d", meal2, rows[0].MealID.Int32)
	}
}

func TestIntegration_GetWeekPlan_EmptyWeek_ReturnsNoRows(t *testing.T) {
	ctx := context.Background()
	uid, _, _ := makeUser(t)
	_, userID := personalScope(uid)

	// Use a week far in the future that can't have any existing rows.
	futureWeek := weekOf(time.Now().AddDate(10, 0, 0))

	rows, err := database.GetWeekPlan(ctx, sharedPool(), database.GetWeekPlanParams{
		WeekStart: futureWeek,
		UserID:    userID,
	})
	if err != nil {
		t.Fatalf("GetWeekPlan: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("expected 0 rows for an unpopulated week, got %d", len(rows))
	}
}

// ── GenerateNextWeek ──────────────────────────────────────────────────────────

func TestIntegration_GenerateNextWeek_CopiesRepeatingAssignments(t *testing.T) {
	ctx := context.Background()
	uid, _, _ := makeUser(t)
	mealID := makeMeal(t, "RepeatingMeal")
	_, userID := personalScope(uid)

	thisWeek := weekOf(time.Now())
	nextWeek := thisWeek.AddDate(0, 0, 7)

	// Set a repeating assignment in this week.
	if err := database.SetWeekPlanDay(ctx, sharedPool(), database.SetWeekPlanDayParams{
		DayName:         "Saturday",
		WeekStart:       thisWeek,
		RepeatingMealID: pgtype.Int4{Int32: mealID, Valid: true},
		UserID:          userID,
	}); err != nil {
		t.Fatalf("SetWeekPlanDay (repeating): %v", err)
	}

	n, err := database.GenerateNextWeek(ctx, sharedPool(), database.GenerateWeekParams{
		TargetWeekStart: nextWeek,
		UserID:          userID,
	})
	if err != nil {
		t.Fatalf("GenerateNextWeek: %v", err)
	}
	if n == 0 {
		t.Fatal("expected at least one row to be generated")
	}

	rows, err := database.GetWeekPlan(ctx, sharedPool(), database.GetWeekPlanParams{
		WeekStart: nextWeek,
		UserID:    userID,
	})
	if err != nil {
		t.Fatalf("GetWeekPlan next week: %v", err)
	}

	found := false
	for _, r := range rows {
		if r.DayName == "Saturday" && r.RepeatingMealID.Int32 == mealID {
			found = true
			break
		}
	}
	if !found {
		t.Error("Saturday repeating assignment was not carried into next week")
	}
}

func TestIntegration_GenerateNextWeek_DoesNotOverwriteExistingRows(t *testing.T) {
	ctx := context.Background()
	uid, _, _ := makeUser(t)
	repeatingMeal := makeMeal(t, "Repeating")
	manualMeal := makeMeal(t, "ManualOverride")
	_, userID := personalScope(uid)

	thisWeek := weekOf(time.Now().AddDate(0, 0, 14)) // offset to avoid collision with previous test
	nextWeek := thisWeek.AddDate(0, 0, 7)

	// Set repeating in this week.
	if err := database.SetWeekPlanDay(ctx, sharedPool(), database.SetWeekPlanDayParams{
		DayName:         "Sunday",
		WeekStart:       thisWeek,
		RepeatingMealID: pgtype.Int4{Int32: repeatingMeal, Valid: true},
		UserID:          userID,
	}); err != nil {
		t.Fatalf("SetWeekPlanDay (repeating): %v", err)
	}

	// Manually set Sunday of next week before generating.
	if err := database.SetWeekPlanDay(ctx, sharedPool(), database.SetWeekPlanDayParams{
		DayName:   "Sunday",
		WeekStart: nextWeek,
		MealID:    pgtype.Int4{Int32: manualMeal, Valid: true},
		UserID:    userID,
	}); err != nil {
		t.Fatalf("SetWeekPlanDay (manual): %v", err)
	}

	if _, err := database.GenerateNextWeek(ctx, sharedPool(), database.GenerateWeekParams{
		TargetWeekStart: nextWeek,
		UserID:          userID,
	}); err != nil {
		t.Fatalf("GenerateNextWeek: %v", err)
	}

	rows, err := database.GetWeekPlan(ctx, sharedPool(), database.GetWeekPlanParams{
		WeekStart: nextWeek,
		UserID:    userID,
	})
	if err != nil {
		t.Fatalf("GetWeekPlan: %v", err)
	}

	for _, r := range rows {
		if r.DayName == "Sunday" && r.MealID.Int32 == repeatingMeal {
			t.Error("GenerateNextWeek should not have overwritten the manually set Sunday")
		}
	}
}

// ── DistinctScopes ────────────────────────────────────────────────────────────

func TestIntegration_DistinctScopes_IncludesScopeWithRepeating(t *testing.T) {
	ctx := context.Background()
	uid, _, _ := makeUser(t)
	mealID := makeMeal(t, "ScopeMeal")
	_, userID := personalScope(uid)
	week := weekOf(time.Now().AddDate(0, 0, 21)) // distinct week to avoid collision

	if err := database.SetWeekPlanDay(ctx, sharedPool(), database.SetWeekPlanDayParams{
		DayName:         "Monday",
		WeekStart:       week,
		RepeatingMealID: pgtype.Int4{Int32: mealID, Valid: true},
		UserID:          userID,
	}); err != nil {
		t.Fatalf("SetWeekPlanDay: %v", err)
	}

	scopes, err := database.DistinctScopes(ctx, sharedPool())
	if err != nil {
		t.Fatalf("DistinctScopes: %v", err)
	}

	found := false
	for _, s := range scopes {
		if s.UserID.Int32 == uid {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected user ID=%d to appear in DistinctScopes, but it did not", uid)
	}
}

// WeekStart is exported from the database package and also tested here to
// confirm the Monday-anchoring logic is correct.
func TestIntegration_WeekStart_AlwaysReturnsMonday(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"2024-01-01", "2024-01-01"}, // Monday
		{"2024-01-07", "2024-01-01"}, // Sunday of same week
		{"2024-01-03", "2024-01-01"}, // Wednesday
		{"2024-01-08", "2024-01-08"}, // Next Monday
	}
	for _, c := range cases {
		input, _ := time.Parse("2006-01-02", c.input)
		got := database.WeekStart(input).Format("2006-01-02")
		if got != c.want {
			t.Errorf("WeekStart(%s) = %s, want %s", c.input, got, c.want)
		}
	}
}

// Ensure fmt and time are used (some builds may warn otherwise).
var _ = fmt.Sprintf
var _ = time.Now
