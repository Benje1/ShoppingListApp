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
	r := buildMealResponse(meal, rows)
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
