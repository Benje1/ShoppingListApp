package meals

// ── Extended Meal Plan types (migration 009) ─────────────────────────────────
//
// Each weekday slot now carries four independent values that resolve in this
// priority order when deciding what to cook and who cooks it:
//
//   1. TempCook / TempMeal   – one-off override for the coming week
//   2. RepeatingCook / RepeatingMeal – standing weekly assignment
//
// The existing Cook / MealID fields on MealPlanDayResponse represent the
// *effective* (already-resolved) values the shopping-list logic already reads.

// CookSlot is a nullable reference to a user acting as cook.
type CookSlot struct {
	ID       int32  `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
}

// MealSlot is a nullable reference to a meal.
type MealSlot struct {
	ID   int32  `json:"id"`
	Name string `json:"name"`
}

// MealPlanDayResponseV2 extends the original MealPlanDayResponse with the
// four slots introduced in migration 009.
type MealPlanDayResponseV2 struct {
	DayName         string        `json:"day_name"`
	MealID          *int32        `json:"meal_id"`
	MealName        string        `json:"meal_name"`
	MealDescription string        `json:"meal_description"`
	DefaultPortions int32         `json:"default_portions"`
	Cook            *CookResponse `json:"cook"`

	// Standing weekly assignments
	RepeatingCook *CookSlot `json:"repeating_cook"`
	RepeatingMeal *MealSlot `json:"repeating_meal"`

	// One-off overrides for the coming week
	TempCook *CookSlot `json:"temp_cook"`
	TempMeal *MealSlot `json:"temp_meal"`
}

// EffectiveCook returns the cook that should be used this week:
// TempCook takes priority; falls back to RepeatingCook; nil if neither set.
func (d MealPlanDayResponseV2) EffectiveCook() *CookSlot {
	if d.TempCook != nil && d.TempCook.ID != 0 {
		return d.TempCook
	}
	return d.RepeatingCook
}

// EffectiveMeal returns the meal that should be served this week:
// TempMeal takes priority; falls back to RepeatingMeal; nil if neither set.
func (d MealPlanDayResponseV2) EffectiveMeal() *MealSlot {
	if d.TempMeal != nil && d.TempMeal.ID != 0 {
		return d.TempMeal
	}
	return d.RepeatingMeal
}

// SetMealPlanInputV2 is the request body for the extended upsert endpoint.
type SetMealPlanInputV2 struct {
	DayName     string `json:"day_name"`
	Scope       string `json:"scope"`
	HouseholdID int32  `json:"household_id"`

	// WeekOffset selects which week to write: 0 = this week (default), 1 = next week.
	WeekOffset int `json:"week_offset"`

	// Effective overrides (replaces both slots at once if provided)
	MealID     int32 `json:"meal_id"`
	CookUserID int32 `json:"cook_user_id"`

	// Granular slot control
	RepeatingMealID      int32 `json:"repeating_meal_id"`
	RepeatingCookUserID  int32 `json:"repeating_cook_user_id"`
	TempMealID           int32 `json:"temp_meal_id"`
	TempCookUserID       int32 `json:"temp_cook_user_id"`
}

// ClearTempInput is the request body for rolling over a week (clears temp
// fields and promotes repeating values into the effective slots).
type ClearTempInput struct {
	DayName     string `json:"day_name"`      // empty = clear all days
	Scope       string `json:"scope"`
	HouseholdID int32  `json:"household_id"`
}
