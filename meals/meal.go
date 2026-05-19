package meals

import (
	"context"
	"fmt"
	"math"

	sqlc "weekly-shopping-app/database/sqlc"
	"weekly-shopping-app/internal/logger"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ── Response types ────────────────────────────────────────────────────────────

type IngredientResponse struct {
	ItemID          int32   `json:"item_id"`
	ItemName        string  `json:"item_name"`
	ItemType        string  `json:"item_type"`
	Quantity        float64 `json:"quantity"`
	Unit            string  `json:"unit"`
	PortionsPerUnit int32   `json:"portions_per_unit"`
}

type ParentRef struct {
	MealID int32  `json:"meal_id"`
	Name   string `json:"name"`
}

type MealResponse struct {
	ID              int32  `json:"id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	DefaultPortions int32  `json:"default_portions"`
	Season          string `json:"season"` // empty string means no season set
	// HouseholdID is nil for global/shared meals.
	HouseholdID  *int32                     `json:"household_id"`
	Ingredients  []IngredientResponse       `json:"ingredients"`
	Cooks        []CookResponse             `json:"cooks"`
	Components   []ComponentResponse        `json:"components"`  // sub-meals
	PartOf       []ParentRef                `json:"part_of"`     // composite meals that include this
	OptionGroups []OptionGroupEntryResponse `json:"option_groups"` // optional choice groups
}

type MealSummary struct {
	ID              int32  `json:"id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	DefaultPortions int32  `json:"default_portions"`
	Season          string `json:"season"` // empty string means no season set
	IngredientCount int64  `json:"ingredient_count"`
	// HouseholdID is nil for global/shared meals.
	HouseholdID *int32 `json:"household_id"`
}

// ── Input types ───────────────────────────────────────────────────────────────

type CreateMealInput struct {
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	DefaultPortions int32             `json:"default_portions"`
	Season          string            `json:"season"` // "spring"|"summer"|"autumn"|"winter"|"" (nullable)
	Ingredients     []IngredientInput `json:"ingredients"`
	// HouseholdID makes this meal household-specific. Omit (or set 0) for a global meal.
	HouseholdID int32 `json:"household_id"`
}

type IngredientInput struct {
	ItemID   int32   `json:"item_id"`
	Quantity float64 `json:"quantity"`
	Unit     string  `json:"unit"`
}

type UpdateMealInput struct {
	Name            string `json:"name"`
	Description     string `json:"description"`
	DefaultPortions int32  `json:"default_portions"`
	Season          string `json:"season"` // "spring"|"summer"|"autumn"|"winter"|"" (nullable)
	// HouseholdID makes this meal household-specific. Set 0 to make it global again.
	HouseholdID int32 `json:"household_id"`
}

type AddIngredientInput struct {
	ItemID   int32   `json:"item_id"`
	Quantity float64 `json:"quantity"`
	Unit     string  `json:"unit"`
}

type UpdateIngredientInput struct {
	ItemID   int32   `json:"item_id"`
	Quantity float64 `json:"quantity"`
	Unit     string  `json:"unit"`
}

type RemoveIngredientInput struct {
	ItemID int32 `json:"item_id"`
}

// ── Option groups ──────────────────────────────────────────────────────────────
// An option group is a named set of choices on a meal. Each entry is either a
// shopping item (e.g. "chicken breast") or a whole sub-meal (e.g. "mashed
// potatoes"). The user picks from the group when adding the meal to a plan;
// their selections are passed as IncludedOptionEntries on SetMealPlanInput.
//
// option_type "one_of"  — user picks exactly one entry from the group
// option_type "many_of" — user picks any subset of entries from the group

// OptionGroupEntryResponse is returned when reading a meal's option groups.
type OptionGroupEntryResponse struct {
	ID          int32  `json:"id"`
	OptionGroup string `json:"option_group"` // user-defined label, e.g. "potatoes", "meat"
	OptionType  string `json:"option_type"`  // "one_of" | "many_of"
	SortOrder   int32  `json:"sort_order"`
	// Set when the entry is a shopping item
	ItemID   *int32 `json:"item_id,omitempty"`
	ItemName string `json:"item_name,omitempty"`
	// Set when the entry is a sub-meal
	SubMealID   *int32 `json:"sub_meal_id,omitempty"`
	SubMealName string `json:"sub_meal_name,omitempty"`
}

// AddOptionGroupEntryInput adds one entry to an option group.
// Exactly one of ItemID or SubMealID must be non-zero.
type AddOptionGroupEntryInput struct {
	OptionGroup string `json:"option_group"`
	OptionType  string `json:"option_type"`  // "one_of" | "many_of"
	SortOrder   int32  `json:"sort_order"`
	ItemID      int32  `json:"item_id"`     // 0 = not set
	SubMealID   int32  `json:"sub_meal_id"` // 0 = not set
}

// UpdateOptionGroupEntryInput updates an existing entry by its ID.
// Exactly one of ItemID or SubMealID must be non-zero.
type UpdateOptionGroupEntryInput struct {
	EntryID     int32  `json:"entry_id"`
	OptionGroup string `json:"option_group"`
	OptionType  string `json:"option_type"`
	SortOrder   int32  `json:"sort_order"`
	ItemID      int32  `json:"item_id"`
	SubMealID   int32  `json:"sub_meal_id"`
}

// RemoveOptionGroupEntryInput removes one entry by its ID.
type RemoveOptionGroupEntryInput struct {
	EntryID int32 `json:"entry_id"`
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func toText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}

func toNumeric(f float64) pgtype.Numeric {
	n := pgtype.Numeric{}
	_ = n.Scan(fmt.Sprintf("%.2f", f))
	return n
}

func toNullSeason(s string) sqlc.NullSeason {
	switch sqlc.Season(s) {
	case sqlc.SeasonSpring, sqlc.SeasonSummer, sqlc.SeasonAutumn, sqlc.SeasonWinter:
		return sqlc.NullSeason{Season: sqlc.Season(s), Valid: true}
	}
	return sqlc.NullSeason{}
}

func numericToFloat(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 1
	}
	f, _ := n.Float64Value()
	return f.Float64
}

func buildMealResponse(meal sqlc.Meal, rows []sqlc.GetMealWithIngredientsRow) MealResponse {
	desc := ""
	if meal.Description.Valid {
		desc = meal.Description.String
	}
	ingredients := make([]IngredientResponse, 0, len(rows))
	for _, r := range rows {
		unit := ""
		if r.Unit.Valid {
			unit = r.Unit.String
		}
		ingredients = append(ingredients, IngredientResponse{
			ItemID:          r.ShoppingItemID,
			ItemName:        r.IngredientName,
			ItemType:        string(r.IngredientType),
			Quantity:        numericToFloat(r.Quantity),
			Unit:            unit,
			PortionsPerUnit: r.PortionsPerUnit,
		})
	}
	resp := MealResponse{
		ID:              meal.ID,
		Name:            meal.Name,
		Description:     desc,
		DefaultPortions: meal.DefaultPortions,
		Ingredients:     ingredients,
		Cooks:           []CookResponse{},
		Components:      []ComponentResponse{},
		PartOf:          []ParentRef{},
		OptionGroups:    []OptionGroupEntryResponse{},
	}
	if meal.Season.Valid {
		resp.Season = string(meal.Season.Season)
	}
	if meal.HouseholdID.Valid {
		hid := meal.HouseholdID.Int32
		resp.HouseholdID = &hid
	}
	return resp
}

// ── Business logic ────────────────────────────────────────────────────────────

func listMeals(ctx context.Context, db *pgxpool.Pool, householdID pgtype.Int4) ([]MealSummary, error) {
	q := sqlc.New(db)
	rows, err := q.ListMealsWithIngredientCount(ctx, householdID)
	if err != nil {
		return nil, err
	}
	out := make([]MealSummary, len(rows))
	for i, r := range rows {
		desc := ""
		if r.Description.Valid {
			desc = r.Description.String
		}
		s := MealSummary{
			ID:              r.ID,
			Name:            r.Name,
			Description:     desc,
			DefaultPortions: r.DefaultPortions,
			IngredientCount: r.IngredientCount,
		}
		if r.Season.Valid {
			s.Season = string(r.Season.Season)
		}
		if r.HouseholdID.Valid {
			hid := r.HouseholdID.Int32
			s.HouseholdID = &hid
		}
		out[i] = s
	}
	return out, nil
}

func getMeal(ctx context.Context, db *pgxpool.Pool, id int32) (*MealResponse, error) {
	q := sqlc.New(db)
	meal, err := q.GetMeal(ctx, id)
	if err != nil {
		return nil, err
	}
	rows, err := q.GetMealWithIngredients(ctx, id)
	if err != nil {
		return nil, err
	}
	cookRows, err := q.GetMealCooks(ctx, id)
	if err != nil {
		return nil, err
	}
	ogRows, err := q.GetMealOptionGroups(ctx, id)
	if err != nil {
		return nil, err
	}
	r := buildMealResponse(meal, rows)
	r.Cooks = make([]CookResponse, len(cookRows))
	for i, c := range cookRows {
		cr := CookResponse{ID: c.ID, Name: c.Name, Username: c.Username}
		if c.HouseholdID.Valid {
			hid := c.HouseholdID.Int32
			cr.HouseholdID = &hid
		}
		if c.HouseholdName.Valid {
			cr.HouseholdName = c.HouseholdName.String
		}
		r.Cooks[i] = cr
	}
	r.OptionGroups = buildOptionGroupResponse(ogRows)
	return &r, nil
}

func buildOptionGroupResponse(rows []sqlc.GetMealOptionGroupsRow) []OptionGroupEntryResponse {
	out := make([]OptionGroupEntryResponse, 0, len(rows))
	for _, r := range rows {
		entry := OptionGroupEntryResponse{
			ID:          r.ID,
			OptionGroup: r.OptionGroup,
			OptionType:  r.OptionType,
			SortOrder:   r.SortOrder,
		}
		if r.ShoppingItemID.Valid {
			id := r.ShoppingItemID.Int32
			entry.ItemID = &id
			if r.ItemName.Valid {
				entry.ItemName = r.ItemName.String
			}
		}
		if r.SubMealID.Valid {
			id := r.SubMealID.Int32
			entry.SubMealID = &id
			if r.SubMealName.Valid {
				entry.SubMealName = r.SubMealName.String
			}
		}
		out = append(out, entry)
	}
	return out
}

func createMeal(ctx context.Context, db *pgxpool.Pool, input CreateMealInput) (*MealResponse, error) {
	if input.DefaultPortions <= 0 {
		input.DefaultPortions = 2
	}
	q := sqlc.New(db)
	meal, err := q.CreateMeal(ctx, sqlc.CreateMealParams{
		Name:            input.Name,
		Description:     toText(input.Description),
		DefaultPortions: input.DefaultPortions,
		Season:          toNullSeason(input.Season),
		HouseholdID:     nullableInt4(input.HouseholdID),
	})
	if err != nil {
		return nil, err
	}
	// Add any ingredients provided at creation time
	for _, ing := range input.Ingredients {
		if _, err := q.AddMealIngredient(ctx, sqlc.AddMealIngredientParams{
			MealID:         meal.ID,
			ShoppingItemID: ing.ItemID,
			Quantity:       toNumeric(ing.Quantity),
			Unit:           toText(ing.Unit),
		}); err != nil {
			return nil, fmt.Errorf("adding ingredient %d: %w", ing.ItemID, err)
		}
	}
	return getMealFull(ctx, db, meal.ID)
}

func updateMeal(ctx context.Context, db *pgxpool.Pool, id int32, input UpdateMealInput) (*MealResponse, error) {
	if input.DefaultPortions <= 0 {
		input.DefaultPortions = 2
	}
	q := sqlc.New(db)
	if _, err := q.UpdateMeal(ctx, sqlc.UpdateMealParams{
		ID:              id,
		Name:            input.Name,
		Description:     toText(input.Description),
		DefaultPortions: input.DefaultPortions,
		Season:          toNullSeason(input.Season),
		HouseholdID:     nullableInt4(input.HouseholdID),
	}); err != nil {
		return nil, err
	}
	return getMealFull(ctx, db, id)
}

func deleteMeal(ctx context.Context, db *pgxpool.Pool, id int32) error {
	q := sqlc.New(db)
	return q.DeleteMeal(ctx, id)
}

func addIngredient(ctx context.Context, db *pgxpool.Pool, mealID int32, input AddIngredientInput) (*MealResponse, error) {
	q := sqlc.New(db)
	if _, err := q.AddMealIngredient(ctx, sqlc.AddMealIngredientParams{
		MealID:         mealID,
		ShoppingItemID: input.ItemID,
		Quantity:       toNumeric(input.Quantity),
		Unit:           toText(input.Unit),
	}); err != nil {
		return nil, err
	}
	return getMeal(ctx, db, mealID)
}

func updateIngredient(ctx context.Context, db *pgxpool.Pool, mealID int32, input UpdateIngredientInput) (*MealResponse, error) {
	q := sqlc.New(db)
	if _, err := q.UpdateMealIngredient(ctx, sqlc.UpdateMealIngredientParams{
		MealID:         mealID,
		ShoppingItemID: input.ItemID,
		Quantity:       toNumeric(input.Quantity),
		Unit:           toText(input.Unit),
	}); err != nil {
		return nil, err
	}
	return getMeal(ctx, db, mealID)
}

// ── Option group CRUD ─────────────────────────────────────────────────────────

func getOptionGroups(ctx context.Context, db *pgxpool.Pool, mealID int32) ([]OptionGroupEntryResponse, error) {
	q := sqlc.New(db)
	rows, err := q.GetMealOptionGroups(ctx, mealID)
	if err != nil {
		return nil, err
	}
	return buildOptionGroupResponse(rows), nil
}

func addOptionGroupEntry(ctx context.Context, db *pgxpool.Pool, mealID int32, input AddOptionGroupEntryInput) ([]OptionGroupEntryResponse, error) {
	if (input.ItemID == 0) == (input.SubMealID == 0) {
		return nil, fmt.Errorf("exactly one of item_id or sub_meal_id must be set")
	}
	q := sqlc.New(db)
	if _, err := q.AddMealOptionGroupEntry(ctx, sqlc.AddMealOptionGroupEntryParams{
		MealID:         mealID,
		OptionGroup:    input.OptionGroup,
		OptionType:     input.OptionType,
		SortOrder:      input.SortOrder,
		ShoppingItemID: nullableInt4(input.ItemID),
		SubMealID:      nullableInt4(input.SubMealID),
	}); err != nil {
		return nil, err
	}
	return getOptionGroups(ctx, db, mealID)
}

func updateOptionGroupEntry(ctx context.Context, db *pgxpool.Pool, mealID int32, input UpdateOptionGroupEntryInput) ([]OptionGroupEntryResponse, error) {
	if (input.ItemID == 0) == (input.SubMealID == 0) {
		return nil, fmt.Errorf("exactly one of item_id or sub_meal_id must be set")
	}
	q := sqlc.New(db)
	if _, err := q.UpdateMealOptionGroupEntry(ctx, sqlc.UpdateMealOptionGroupEntryParams{
		ID:             input.EntryID,
		OptionGroup:    input.OptionGroup,
		OptionType:     input.OptionType,
		SortOrder:      input.SortOrder,
		ShoppingItemID: nullableInt4(input.ItemID),
		SubMealID:      nullableInt4(input.SubMealID),
	}); err != nil {
		return nil, err
	}
	return getOptionGroups(ctx, db, mealID)
}

func removeOptionGroupEntry(ctx context.Context, db *pgxpool.Pool, mealID int32, input RemoveOptionGroupEntryInput) ([]OptionGroupEntryResponse, error) {
	q := sqlc.New(db)
	if err := q.RemoveMealOptionGroupEntry(ctx, sqlc.RemoveMealOptionGroupEntryParams{
		ID:     input.EntryID,
		MealID: mealID,
	}); err != nil {
		return nil, err
	}
	return getOptionGroups(ctx, db, mealID)
}

func removeIngredient(ctx context.Context, db *pgxpool.Pool, mealID int32, input RemoveIngredientInput) (*MealResponse, error) {
	q := sqlc.New(db)
	if err := q.RemoveMealIngredient(ctx, sqlc.RemoveMealIngredientParams{
		MealID:         mealID,
		ShoppingItemID: input.ItemID,
	}); err != nil {
		return nil, err
	}
	return getMeal(ctx, db, mealID)
}

// ── Cook management ───────────────────────────────────────────────────────────

type CookResponse struct {
	ID       int32  `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	// HouseholdID is nil when the assignment applies across all households.
	HouseholdID   *int32 `json:"household_id"`
	HouseholdName string `json:"household_name"`
}

func getMealCooks(ctx context.Context, db *pgxpool.Pool, mealID int32) ([]CookResponse, error) {
	q := sqlc.New(db)
	rows, err := q.GetMealCooks(ctx, mealID)
	if err != nil {
		return nil, err
	}
	out := make([]CookResponse, len(rows))
	for i, r := range rows {
		c := CookResponse{ID: r.ID, Name: r.Name, Username: r.Username}
		if r.HouseholdID.Valid {
			hid := r.HouseholdID.Int32
			c.HouseholdID = &hid
		}
		if r.HouseholdName.Valid {
			c.HouseholdName = r.HouseholdName.String
		}
		out[i] = c
	}
	return out, nil
}

func addMealCook(ctx context.Context, db *pgxpool.Pool, mealID, userID int32, householdID pgtype.Int4) ([]CookResponse, error) {
	q := sqlc.New(db)
	if err := q.AddMealCook(ctx, sqlc.AddMealCookParams{
		MealID:      mealID,
		UserID:      userID,
		HouseholdID: householdID,
	}); err != nil {
		return nil, err
	}
	return getMealCooks(ctx, db, mealID)
}

func removeMealCook(ctx context.Context, db *pgxpool.Pool, mealID, userID int32, householdID pgtype.Int4) ([]CookResponse, error) {
	q := sqlc.New(db)
	if err := q.RemoveMealCook(ctx, sqlc.RemoveMealCookParams{
		MealID:  mealID,
		UserID:  userID,
		Column3: householdID.Int32,
	}); err != nil {
		return nil, err
	}
	return getMealCooks(ctx, db, mealID)
}

func getMealsForCook(ctx context.Context, db *pgxpool.Pool, userID int32, householdID pgtype.Int4) ([]MealSummary, error) {
	q := sqlc.New(db)
	rows, err := q.GetMealsForCook(ctx, sqlc.GetMealsForCookParams{
		UserID:      userID,
		HouseholdID: householdID,
	})
	if err != nil {
		return nil, err
	}
	out := make([]MealSummary, len(rows))
	for i, r := range rows {
		desc := ""
		if r.Description.Valid {
			desc = r.Description.String
		}
		out[i] = MealSummary{ID: r.ID, Name: r.Name, Description: desc, DefaultPortions: r.DefaultPortions}
	}
	return out, nil
}

// ── Meal plan ─────────────────────────────────────────────────────────────────

type MealPlanDayResponse struct {
	DayName         string        `json:"day_name"`
	MealID          *int32        `json:"meal_id"`
	MealName        string        `json:"meal_name"`
	MealDescription string        `json:"meal_description"`
	DefaultPortions int32         `json:"default_portions"`
	Cook            *CookResponse `json:"cook"`
}

type SetMealPlanInput struct {
	DayName     string  `json:"day_name"`
	MealID      int32   `json:"meal_id"`
	CookUserID  int32   `json:"cook_user_id"` // 0 = no cook assigned
	Scope       string  `json:"scope"`
	HouseholdID int32   `json:"household_id"`
	// IncludedOptionEntries lists the IDs of option group entries the user has
	// chosen to include when adding this meal to the plan. Each ID corresponds
	// to a row in meal_option_group_entries. Required entries (no option group)
	// are always added regardless.
	IncludedOptionEntries []int32 `json:"included_option_entries"`
}

type ClearMealPlanInput struct {
	DayName     string `json:"day_name"`
	Scope       string `json:"scope"`
	HouseholdID int32  `json:"household_id"`
}

type AddCookInput struct {
	UserID int32 `json:"user_id"`
	// HouseholdID scopes this cook assignment to a specific household.
	// Omit (or set 0) to create a cross-household assignment.
	HouseholdID int32 `json:"household_id"`
}

func nullableInt4(n int32) pgtype.Int4 {
	if n == 0 {
		return pgtype.Int4{Valid: false}
	}
	return pgtype.Int4{Int32: n, Valid: true}
}

func getMealPlanFull(ctx context.Context, db *pgxpool.Pool, userID, householdID int32) ([]MealPlanDayResponse, error) {
	q := sqlc.New(db)
	rows, err := q.GetMealPlanFull(ctx, sqlc.GetMealPlanFullParams{
		HouseholdID: pgtype.Int4{Int32: householdID, Valid: householdID != 0},
		UserID:      pgtype.Int4{Int32: userID, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	out := make([]MealPlanDayResponse, 0, len(rows))
	for _, r := range rows {
		day := MealPlanDayResponse{DayName: r.DayName}
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
		if r.CookName.Valid {
			day.Cook = &CookResponse{
				ID:       r.CookUserID.Int32,
				Name:     r.CookName.String,
				Username: r.CookUsername.String,
			}
		}
		out = append(out, day)
	}
	return out, nil
}

func setMealPlanDay(ctx context.Context, db *pgxpool.Pool, userID int32, input SetMealPlanInput) (*MealPlanDayResponse, error) {
	q := sqlc.New(db)
	hid, uid := planScope(userID, input.HouseholdID, input.Scope)
	result, err := q.SetMealPlanDay(ctx, sqlc.SetMealPlanDayParams{
		DayName:     input.DayName,
		MealID:      nullableInt4(input.MealID),
		CookUserID:  nullableInt4(input.CookUserID),
		HouseholdID: hid,
		UserID:      uid,
	})
	if err != nil {
		return nil, err
	}
	day := &MealPlanDayResponse{DayName: result.DayName}
	if result.MealID.Valid {
		id := result.MealID.Int32
		day.MealID = &id
		// Fetch meal name for response
		meal, err := q.GetMeal(ctx, id)
		if err == nil {
			day.MealName = meal.Name
			if meal.Description.Valid {
				day.MealDescription = meal.Description.String
			}
			day.DefaultPortions = meal.DefaultPortions
		}
		// Add all meal ingredients (including sub-meals) to the shopping list
		if addErr := addMealIngredientsToShoppingList(ctx, db, id, hid, uid, input.IncludedOptionEntries); addErr != nil {
			// Non-fatal — plan was saved, log and continue
			logger.Warn("meal plan: could not add ingredients to shopping list", "err", addErr)
		}
	}
	return day, nil
}

// addMealIngredientsToShoppingList collects all ingredients from a meal and its
// sub-meal components (recursively) and upserts them onto the shopping list.
// Quantities are rounded up to the nearest integer (ceiling) since the shopping
// list stores int quantities.
//
// includedOptionEntries is the set of meal_option_group_entries IDs the user
// selected. For each selected entry: if it references a shopping item that item
// is added; if it references a sub-meal that meal is recursed into. Entries not
// in this set are skipped entirely.
func addMealIngredientsToShoppingList(ctx context.Context, db *pgxpool.Pool, mealID int32, hid, uid pgtype.Int4, includedOptionEntries []int32) error {
	q := sqlc.New(db)

	// Build a fast lookup set of chosen option entry IDs
	selectedEntries := make(map[int32]bool, len(includedOptionEntries))
	for _, id := range includedOptionEntries {
		selectedEntries[id] = true
	}

	// Accumulate total quantity per item across this meal and all sub-meals
	totals := make(map[int32]float64)

	var collect func(id int32) error
	collect = func(id int32) error {
		// Required ingredients — always included
		ings, err := q.GetMealWithIngredients(ctx, id)
		if err != nil {
			return err
		}
		for _, ing := range ings {
			qty := numericToFloat(ing.Quantity)
			if qty <= 0 {
				qty = 1
			}
			totals[ing.ShoppingItemID] += qty
		}

		// Option group entries — only included when the user selected them
		ogRows, err := q.GetMealOptionGroups(ctx, id)
		if err != nil {
			return err
		}
		for _, og := range ogRows {
			if !selectedEntries[og.ID] {
				continue
			}
			if og.ShoppingItemID.Valid {
				// Entry is a shopping item — add it directly
				totals[og.ShoppingItemID.Int32] += 1
			} else if og.SubMealID.Valid {
				// Entry is a sub-meal — recurse into it to collect its ingredients
				if err := collect(og.SubMealID.Int32); err != nil {
					return err
				}
			}
		}

		// Recurse into fixed sub-meal components
		components, err := q.GetMealComponents(ctx, id)
		if err != nil {
			return err
		}
		for _, c := range components {
			if err := collect(c.ID); err != nil {
				return err
			}
		}
		return nil
	}

	if err := collect(mealID); err != nil {
		return err
	}

	// Upsert each ingredient onto the shopping list
	for itemID, qty := range totals {
		// Round up — we never want to add 0 of something
		rounded := int32(math.Ceil(qty))
		if rounded < 1 {
			rounded = 1
		}
		_, err := q.AddToShoppingList(ctx, sqlc.AddToShoppingListParams{
			ShoppingItemID: itemID,
			Quantity:       rounded,
			HouseholdID:    hid,
			UserID:         uid,
		})
		if err != nil {
			return fmt.Errorf("adding item %d to shopping list: %w", itemID, err)
		}
	}
	return nil
}

func clearMealPlanDay(ctx context.Context, db *pgxpool.Pool, userID int32, input ClearMealPlanInput) error {
	q := sqlc.New(db)
	hid, uid := planScope(userID, input.HouseholdID, input.Scope)
	return q.ClearMealPlanDay(ctx, sqlc.ClearMealPlanDayParams{
		DayName:     input.DayName,
		HouseholdID: hid,
		UserID:      uid,
	})
}

func planScope(userID, householdID int32, scope string) (pgtype.Int4, pgtype.Int4) {
	if scope == "household" && householdID != 0 {
		return pgtype.Int4{Int32: householdID, Valid: true}, pgtype.Int4{Valid: false}
	}
	return pgtype.Int4{Valid: false}, pgtype.Int4{Int32: userID, Valid: true}
}

// ── Meal components ───────────────────────────────────────────────────────────

type ComponentResponse struct {
	SortOrder       int32  `json:"sort_order"`
	MealID          int32  `json:"meal_id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	DefaultPortions int32  `json:"default_portions"`
}

type AddComponentInput struct {
	SubMealID int32 `json:"sub_meal_id"`
	SortOrder int32 `json:"sort_order"`
}

type RemoveComponentInput struct {
	SubMealID int32 `json:"sub_meal_id"`
}

func getComponents(ctx context.Context, db *pgxpool.Pool, parentID int32) ([]ComponentResponse, error) {
	q := sqlc.New(db)
	rows, err := q.GetMealComponents(ctx, parentID)
	if err != nil {
		return nil, err
	}
	out := make([]ComponentResponse, len(rows))
	for i, r := range rows {
		desc := ""
		if r.Description.Valid {
			desc = r.Description.String
		}
		out[i] = ComponentResponse{
			SortOrder:       r.SortOrder,
			MealID:          r.ID,
			Name:            r.Name,
			Description:     desc,
			DefaultPortions: r.DefaultPortions,
		}
	}
	return out, nil
}

func addComponent(ctx context.Context, db *pgxpool.Pool, parentID int32, input AddComponentInput) ([]ComponentResponse, error) {
	if parentID == input.SubMealID {
		return nil, fmt.Errorf("a meal cannot be a component of itself")
	}
	// Guard against cycles: check that the sub-meal doesn't already have parentID as a sub-meal
	q := sqlc.New(db)
	existing, err := q.GetMealComponents(ctx, input.SubMealID)
	if err != nil {
		return nil, err
	}
	for _, c := range existing {
		if c.ID == parentID {
			return nil, fmt.Errorf("adding this component would create a cycle")
		}
	}
	if err := q.AddMealComponent(ctx, sqlc.AddMealComponentParams{ParentMealID: parentID, SubMealID: input.SubMealID, SortOrder: input.SortOrder}); err != nil {
		return nil, err
	}
	return getComponents(ctx, db, parentID)
}

func removeComponent(ctx context.Context, db *pgxpool.Pool, parentID int32, input RemoveComponentInput) ([]ComponentResponse, error) {
	q := sqlc.New(db)
	if err := q.RemoveMealComponent(ctx, sqlc.RemoveMealComponentParams{ParentMealID: parentID, SubMealID: input.SubMealID}); err != nil {
		return nil, err
	}
	return getComponents(ctx, db, parentID)
}

// getMeal now includes components and which composite meals use this meal
func getMealFull(ctx context.Context, db *pgxpool.Pool, id int32) (*MealResponse, error) {
	meal, err := getMeal(ctx, db, id)
	if err != nil {
		return nil, err
	}
	// Components (sub-meals)
	comps, err := getComponents(ctx, db, id)
	if err != nil {
		return nil, err
	}
	meal.Components = comps
	// Parent meals that use this meal as a component
	parents, err := sqlc.New(db).GetParentMeals(ctx, id)
	if err != nil {
		return nil, err
	}
	meal.PartOf = make([]ParentRef, len(parents))
	for i, p := range parents {
		meal.PartOf[i] = ParentRef{MealID: p.ParentMealID, Name: p.ParentName}
	}
	return meal, nil
}
