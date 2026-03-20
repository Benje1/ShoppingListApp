package pantry

import (
	"context"
	"fmt"
	"time"

	sqlc "weekly-shopping-app/database/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ── Response types ────────────────────────────────────────────────────────────

type PantryItemResponse struct {
	ID                int32   `json:"id"`
	ShoppingItemID    int32   `json:"shopping_item_id"`
	ItemName          string  `json:"item_name"`
	ItemType          string  `json:"item_type"`
	PortionsPerUnit   int32   `json:"portions_per_unit"`
	ShelfLifeDays     *int32  `json:"shelf_life_days"`
	PortionsRemaining float64 `json:"portions_remaining"`
	ExpiresOn         *string `json:"expires_on"` // "2024-03-15" or null
	Status            string  `json:"status"`      // fresh | expiring_soon | expired
	BoughtAt          string  `json:"bought_at"`
}

// ── Input types ───────────────────────────────────────────────────────────────

type AddToPantryInput struct {
	ItemID      int32   `json:"item_id"`
	Portions    float64 `json:"portions"`    // how many portions being added
	Scope       string  `json:"scope"`       // "personal" or "household"
	HouseholdID int32   `json:"household_id"`
}

type CookMealInput struct {
	MealID      int32   `json:"meal_id"`
	Portions    float64 `json:"portions"`    // how many portions being cooked
	Scope       string  `json:"scope"`
	HouseholdID int32   `json:"household_id"`
}

type SetShelfLifeInput struct {
	Days *int32 `json:"days"` // null = no expiry
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func scopeParams(userID, householdID int32, scope string) (pgtype.Int4, pgtype.Int4) {
	if scope == "household" && householdID != 0 {
		return pgtype.Int4{Int32: householdID, Valid: true}, pgtype.Int4{}
	}
	return pgtype.Int4{}, pgtype.Int4{Int32: userID, Valid: true}
}

func toNumeric(f float64) pgtype.Numeric {
	n := pgtype.Numeric{}
	_ = n.Scan(fmt.Sprintf("%.4f", f))
	return n
}

func numericToFloat(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, _ := n.Float64Value()
	return f.Float64
}

func formatDate(d pgtype.Date) *string {
	if !d.Valid {
		return nil
	}
	s := d.Time.Format("2006-01-02")
	return &s
}

func expiresOn(shelfLifeDays pgtype.Int4) pgtype.Date {
	if !shelfLifeDays.Valid {
		return pgtype.Date{Valid: false}
	}
	t := time.Now().AddDate(0, 0, int(shelfLifeDays.Int32))
	return pgtype.Date{Time: t, Valid: true, InfinityModifier: pgtype.Finite}
}

func buildResponse(r sqlc.PantryRow) PantryItemResponse {
	resp := PantryItemResponse{
		ID:                r.ID,
		ShoppingItemID:    r.ShoppingItemID,
		ItemName:          r.ItemName,
		ItemType:          string(r.ItemType),
		PortionsPerUnit:   r.PortionsPerUnit,
		PortionsRemaining: numericToFloat(r.PortionsRemaining),
		Status:            r.Status,
		ExpiresOn:         formatDate(r.ExpiresOn),
	}
	if r.ShelfLifeDays.Valid {
		v := r.ShelfLifeDays.Int32
		resp.ShelfLifeDays = &v
	}
	if r.BoughtAt.Valid {
		resp.BoughtAt = r.BoughtAt.Time.UTC().Format(time.RFC3339)
	}
	return resp
}

func listParams(userID, householdID int32) sqlc.GetShoppingListParams {
	p := sqlc.GetShoppingListParams{UserID: pgtype.Int4{Int32: userID, Valid: true}}
	if householdID != 0 {
		p.HouseholdID = pgtype.Int4{Int32: householdID, Valid: true}
	}
	return p
}

// ── Business logic ────────────────────────────────────────────────────────────

func getPantry(ctx context.Context, db *pgxpool.Pool, userID, householdID int32) ([]PantryItemResponse, error) {
	q := sqlc.New(db)
	rows, err := q.GetPantry(ctx, listParams(userID, householdID))
	if err != nil {
		return nil, err
	}
	out := make([]PantryItemResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, buildResponse(r))
	}
	return out, nil
}

func addToPantry(ctx context.Context, db *pgxpool.Pool, userID int32, input AddToPantryInput) (PantryItemResponse, error) {
	if input.Portions <= 0 {
		input.Portions = 1
	}
	hid, uid := scopeParams(userID, input.HouseholdID, input.Scope)
	q := sqlc.New(db)

	// Look up the item's shelf life to compute expires_on
	item, err := q.GetShoppingItemByID(ctx, input.ItemID)
	if err != nil {
		return PantryItemResponse{}, fmt.Errorf("item not found: %w", err)
	}

	entry, err := q.UpsertPantryItem(ctx, sqlc.UpsertPantryParams{
		ShoppingItemID:    input.ItemID,
		HouseholdID:       hid,
		UserID:            uid,
		PortionsRemaining: toNumeric(input.Portions),
		ExpiresOn:         expiresOn(item.ShelfLifeDays),
	})
	if err != nil {
		return PantryItemResponse{}, err
	}

	// Re-fetch full row for the response (UpsertPantryItem returns PantryEntry without item details)
	rows, err := q.GetPantry(ctx, listParams(userID, input.HouseholdID))
	if err != nil {
		return PantryItemResponse{}, err
	}
	for _, r := range rows {
		if r.ID == entry.ID {
			return buildResponse(r), nil
		}
	}
	_ = entry
	return PantryItemResponse{}, fmt.Errorf("pantry entry not found after upsert")
}

func removePantryItem(ctx context.Context, db *pgxpool.Pool, id int32) error {
	return sqlc.New(db).RemovePantryItem(ctx, id)
}

// cookMeal decrements pantry portions for every ingredient in the meal
// (and recursively for sub-meals in a composite meal).
// scaleFactor = portions_cooked / default_portions of the meal.
func cookMeal(ctx context.Context, db *pgxpool.Pool, userID int32, input CookMealInput) error {
	q := sqlc.New(db)
	hid, uid := scopeParams(userID, input.HouseholdID, input.Scope)

	// Collect all ingredients across the meal and any sub-meals
	type deduct struct {
		itemID   int32
		quantity float64 // raw quantity from recipe, pre-scaling
	}
	var deductions []deduct

	var collectIngredients func(mealID int32, scale float64) error
	collectIngredients = func(mealID int32, scale float64) error {
		meal, err := q.GetMeal(ctx, mealID)
		if err != nil {
			return err
		}
		ings, err := q.GetMealWithIngredients(ctx, mealID)
		if err != nil {
			return err
		}
		mealScale := scale
		if meal.DefaultPortions > 0 {
			mealScale *= input.Portions / float64(meal.DefaultPortions)
		}
		for _, ing := range ings {
			deductions = append(deductions, deduct{
				itemID:   ing.ShoppingItemID,
				quantity: numericToFloat2(ing.Quantity) * mealScale,
			})
		}
		// Recurse into sub-meals
		components, err := q.GetMealComponents(ctx, mealID)
		if err != nil {
			return err
		}
		for _, c := range components {
			if err := collectIngredients(c.ID, scale); err != nil {
				return err
			}
		}
		return nil
	}

	if err := collectIngredients(input.MealID, 1.0); err != nil {
		return err
	}

	// Aggregate duplicates (same item used in multiple sub-meals)
	totals := make(map[int32]float64)
	for _, d := range deductions {
		totals[d.itemID] += d.quantity
	}

	for itemID, qty := range totals {
		_, err := q.DecrementPantryPortions(ctx, sqlc.DecrementPantryParams{
			ShoppingItemID: itemID,
			HouseholdID:    hid,
			Decrement:      toNumeric(qty),
			UserID:         uid,
		})
		if err != nil {
			// Item not in pantry is fine — just skip
			continue
		}
	}
	return nil
}

// numericToFloat2 is a local alias to avoid the package-level name clash
func numericToFloat2(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 1
	}
	f, _ := n.Float64Value()
	return f.Float64
}

func setShelfLife(ctx context.Context, db *pgxpool.Pool, itemID int32, input SetShelfLifeInput) (sqlc.ShoppingItem, error) {
	var days pgtype.Int4
	if input.Days != nil {
		days = pgtype.Int4{Int32: *input.Days, Valid: true}
	}
	return sqlc.New(db).UpdateShoppingItemShelfLife(ctx, itemID, days)
}

// RunExpiryJob is called by the scheduler — marks rows as expiring_soon / expired.
func RunExpiryJob(ctx context.Context, db *pgxpool.Pool) (int, error) {
	rows, err := sqlc.New(db).ExpirePantryItems(ctx)
	return len(rows), err
}
