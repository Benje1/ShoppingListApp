package meals

import (
	"context"
	"fmt"

	sqlc "weekly-shopping-app/database/sqlc"

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

type MealResponse struct {
	ID              int32                `json:"id"`
	Name            string               `json:"name"`
	Description     string               `json:"description"`
	DefaultPortions int32                `json:"default_portions"`
	Ingredients     []IngredientResponse `json:"ingredients"`
	Cooks           []CookResponse       `json:"cooks"`
}

type MealSummary struct {
	ID              int32  `json:"id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	DefaultPortions int32  `json:"default_portions"`
	IngredientCount int64  `json:"ingredient_count"`
}

// ── Input types ───────────────────────────────────────────────────────────────

type CreateMealInput struct {
	Name            string             `json:"name"`
	Description     string             `json:"description"`
	DefaultPortions int32              `json:"default_portions"`
	Ingredients     []IngredientInput  `json:"ingredients"`
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
}

type AddIngredientInput struct {
	ItemID   int32   `json:"item_id"`
	Quantity float64 `json:"quantity"`
	Unit     string  `json:"unit"`
}

type RemoveIngredientInput struct {
	ItemID int32 `json:"item_id"`
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
	return MealResponse{
		ID:              meal.ID,
		Name:            meal.Name,
		Description:     desc,
		DefaultPortions: meal.DefaultPortions,
		Ingredients:     ingredients,
		Cooks:           []CookResponse{},
	}
}

// ── Business logic ────────────────────────────────────────────────────────────

func listMeals(ctx context.Context, db *pgxpool.Pool) ([]MealSummary, error) {
	q := sqlc.New(db)
	rows, err := q.ListMealsWithIngredientCount(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]MealSummary, len(rows))
	for i, r := range rows {
		desc := ""
		if r.Description.Valid {
			desc = r.Description.String
		}
		out[i] = MealSummary{
			ID:              r.ID,
			Name:            r.Name,
			Description:     desc,
			DefaultPortions: r.DefaultPortions,
			IngredientCount: r.IngredientCount,
		}
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
	r := buildMealResponse(meal, rows)
	r.Cooks = make([]CookResponse, len(cookRows))
	for i, c := range cookRows {
		r.Cooks[i] = CookResponse{ID: c.ID, Name: c.Name, Username: c.Username}
	}
	return &r, nil
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
	return getMeal(ctx, db, meal.ID)
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
	}); err != nil {
		return nil, err
	}
	return getMeal(ctx, db, id)
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
}

func getMealCooks(ctx context.Context, db *pgxpool.Pool, mealID int32) ([]CookResponse, error) {
	q := sqlc.New(db)
	rows, err := q.GetMealCooks(ctx, mealID)
	if err != nil {
		return nil, err
	}
	out := make([]CookResponse, len(rows))
	for i, r := range rows {
		out[i] = CookResponse{ID: r.ID, Name: r.Name, Username: r.Username}
	}
	return out, nil
}

func addMealCook(ctx context.Context, db *pgxpool.Pool, mealID, userID int32) ([]CookResponse, error) {
	q := sqlc.New(db)
	if err := q.AddMealCook(ctx, mealID, userID); err != nil {
		return nil, err
	}
	return getMealCooks(ctx, db, mealID)
}

func removeMealCook(ctx context.Context, db *pgxpool.Pool, mealID, userID int32) ([]CookResponse, error) {
	q := sqlc.New(db)
	if err := q.RemoveMealCook(ctx, mealID, userID); err != nil {
		return nil, err
	}
	return getMealCooks(ctx, db, mealID)
}

func getMealsForCook(ctx context.Context, db *pgxpool.Pool, userID int32) ([]MealSummary, error) {
	q := sqlc.New(db)
	rows, err := q.GetMealsForCook(ctx, userID)
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
	DayName     string `json:"day_name"`
	MealID      int32  `json:"meal_id"`
	CookUserID  int32  `json:"cook_user_id"`  // 0 = no cook assigned
	Scope       string `json:"scope"`
	HouseholdID int32  `json:"household_id"`
}

type ClearMealPlanInput struct {
	DayName     string `json:"day_name"`
	Scope       string `json:"scope"`
	HouseholdID int32  `json:"household_id"`
}

type AddCookInput struct {
	UserID int32 `json:"user_id"`
}

func nullableInt4(n int32) pgtype.Int4 {
	if n == 0 {
		return pgtype.Int4{Valid: false}
	}
	return pgtype.Int4{Int32: n, Valid: true}
}

func getMealPlanFull(ctx context.Context, db *pgxpool.Pool, userID, householdID int32) ([]MealPlanDayResponse, error) {
	q := sqlc.New(db)
	rows, err := q.GetMealPlanFull(ctx,
		pgtype.Int4{Int32: householdID, Valid: householdID != 0},
		pgtype.Int4{Int32: userID, Valid: true},
	)
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
	}
	return day, nil
}

func clearMealPlanDay(ctx context.Context, db *pgxpool.Pool, userID int32, input ClearMealPlanInput) error {
	q := sqlc.New(db)
	hid, uid := planScope(userID, input.HouseholdID, input.Scope)
	return q.ClearMealPlanDay(ctx, input.DayName, hid, uid)
}

func planScope(userID, householdID int32, scope string) (pgtype.Int4, pgtype.Int4) {
	if scope == "household" && householdID != 0 {
		return pgtype.Int4{Int32: householdID, Valid: true}, pgtype.Int4{Valid: false}
	}
	return pgtype.Int4{Valid: false}, pgtype.Int4{Int32: userID, Valid: true}
}
