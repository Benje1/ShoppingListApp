package integration_test

import (
	"context"
	"sort"
	"testing"

	sqlc "weekly-shopping-app/database/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
)

// makeItemWithAllergens creates an isolated catalogue item carrying the given
// allergen tags, for tests that exercise allergen derivation.
func makeItemWithAllergens(t *testing.T, name string, allergens ...string) int32 {
	t.Helper()
	item, err := sqlc.New(sharedPool()).CreateShoppingItem(context.Background(), sqlc.CreateShoppingItemParams{
		Name:            uniqueUsername(name),
		ItemType:        sqlc.ShoppingItemTypePantry,
		PortionsPerUnit: 1,
		Allergens:       allergens,
	})
	if err != nil {
		t.Fatalf("makeItemWithAllergens %q: %v", name, err)
	}
	return item.ID
}

// numeric builds a pgtype.Numeric from a decimal string for NUMERIC columns.
func numeric(t *testing.T, s string) pgtype.Numeric {
	t.Helper()
	var n pgtype.Numeric
	if err := n.Scan(s); err != nil {
		t.Fatalf("numeric(%q): %v", s, err)
	}
	return n
}

// containsAll reports whether got contains every tag in want.
func containsAll(got []string, want ...string) bool {
	set := make(map[string]bool, len(got))
	for _, g := range got {
		set[g] = true
	}
	for _, w := range want {
		if !set[w] {
			return false
		}
	}
	return true
}

// ── shopping_items.allergens round-trips ─────────────────────────────────────

func TestIntegration_CreateShoppingItem_StoresAllergens(t *testing.T) {
	item, err := sqlc.New(sharedPool()).CreateShoppingItem(context.Background(), sqlc.CreateShoppingItemParams{
		Name:            uniqueUsername("Allergen Item"),
		ItemType:        sqlc.ShoppingItemTypePantry,
		PortionsPerUnit: 1,
		Allergens:       []string{"gluten", "soy"},
	})
	if err != nil {
		t.Fatalf("CreateShoppingItem: %v", err)
	}
	if !containsAll(item.Allergens, "gluten", "soy") {
		t.Errorf("expected allergens to include gluten & soy, got %v", item.Allergens)
	}
}

func TestIntegration_CreateShoppingItem_NilAllergens_DefaultsEmpty(t *testing.T) {
	// A nil slice must not violate the NOT NULL column; COALESCE fills '{}'.
	item, err := sqlc.New(sharedPool()).CreateShoppingItem(context.Background(), sqlc.CreateShoppingItemParams{
		Name:            uniqueUsername("No Allergen Item"),
		ItemType:        sqlc.ShoppingItemTypePantry,
		PortionsPerUnit: 1,
		Allergens:       nil,
	})
	if err != nil {
		t.Fatalf("CreateShoppingItem with nil allergens: %v", err)
	}
	if item.Allergens == nil || len(item.Allergens) != 0 {
		t.Errorf("expected empty non-nil allergens, got %v", item.Allergens)
	}
}

// ── Sub-ingredient CRUD ──────────────────────────────────────────────────────

func TestIntegration_SubIngredient_AddAndList(t *testing.T) {
	ctx := context.Background()
	q := sqlc.New(sharedPool())
	itemID := makeItemWithAllergens(t, "Stock Cube")

	sub, err := q.AddSubIngredient(ctx, sqlc.AddSubIngredientParams{
		ShoppingItemID: itemID,
		Name:           "Yeast Extract",
		Allergens:      []string{"gluten"},
		SortOrder:      0,
	})
	if err != nil {
		t.Fatalf("AddSubIngredient: %v", err)
	}
	if sub.ID == 0 {
		t.Error("expected a non-zero sub-ingredient ID")
	}
	if !containsAll(sub.Allergens, "gluten") {
		t.Errorf("expected sub-ingredient allergen gluten, got %v", sub.Allergens)
	}

	list, err := q.ListSubIngredientsForItem(ctx, itemID)
	if err != nil {
		t.Fatalf("ListSubIngredientsForItem: %v", err)
	}
	if len(list) != 1 || list[0].Name != "Yeast Extract" {
		t.Errorf("expected one sub-ingredient named Yeast Extract, got %+v", list)
	}
}

func TestIntegration_SubIngredient_Update(t *testing.T) {
	ctx := context.Background()
	q := sqlc.New(sharedPool())
	itemID := makeItemWithAllergens(t, "Stock Cube")

	sub, err := q.AddSubIngredient(ctx, sqlc.AddSubIngredientParams{
		ShoppingItemID: itemID,
		Name:           "Salt",
		Allergens:      nil,
		SortOrder:      0,
	})
	if err != nil {
		t.Fatalf("AddSubIngredient: %v", err)
	}

	updated, err := q.UpdateSubIngredient(ctx, sqlc.UpdateSubIngredientParams{
		ID:        sub.ID,
		Name:      "Celery Salt",
		Allergens: []string{"celery"},
		SortOrder: 5,
	})
	if err != nil {
		t.Fatalf("UpdateSubIngredient: %v", err)
	}
	if updated.Name != "Celery Salt" {
		t.Errorf("expected name Celery Salt, got %q", updated.Name)
	}
	if !containsAll(updated.Allergens, "celery") {
		t.Errorf("expected allergen celery, got %v", updated.Allergens)
	}
	if updated.SortOrder != 5 {
		t.Errorf("expected sort_order 5, got %d", updated.SortOrder)
	}
}

func TestIntegration_SubIngredient_Remove(t *testing.T) {
	ctx := context.Background()
	q := sqlc.New(sharedPool())
	itemID := makeItemWithAllergens(t, "Stock Cube")

	sub, err := q.AddSubIngredient(ctx, sqlc.AddSubIngredientParams{
		ShoppingItemID: itemID,
		Name:           "MSG",
		SortOrder:      0,
	})
	if err != nil {
		t.Fatalf("AddSubIngredient: %v", err)
	}

	if err := q.RemoveSubIngredient(ctx, sub.ID); err != nil {
		t.Fatalf("RemoveSubIngredient: %v", err)
	}

	list, err := q.ListSubIngredientsForItem(ctx, itemID)
	if err != nil {
		t.Fatalf("ListSubIngredientsForItem: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected no sub-ingredients after removal, got %+v", list)
	}
}

func TestIntegration_SubIngredient_CascadesOnItemDelete(t *testing.T) {
	ctx := context.Background()
	q := sqlc.New(sharedPool())
	itemID := makeItemWithAllergens(t, "Disposable Item")

	if _, err := q.AddSubIngredient(ctx, sqlc.AddSubIngredientParams{
		ShoppingItemID: itemID,
		Name:           "Whey",
		Allergens:      []string{"dairy"},
	}); err != nil {
		t.Fatalf("AddSubIngredient: %v", err)
	}

	if _, err := sharedPool().Exec(ctx, "DELETE FROM shopping_items WHERE id = $1", itemID); err != nil {
		t.Fatalf("delete item: %v", err)
	}

	list, err := q.ListSubIngredientsForItem(ctx, itemID)
	if err != nil {
		t.Fatalf("ListSubIngredientsForItem: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected sub-ingredients to cascade-delete, got %+v", list)
	}
}

// ── Derived allergens across meal → ingredient → sub-ingredient ──────────────

func TestIntegration_MealDerivedAllergens_UnionOfAllLevels(t *testing.T) {
	ctx := context.Background()
	q := sqlc.New(sharedPool())

	// Meal carries a manual tag; ingredient carries its own; sub-ingredient a third.
	mealID := makeMeal(t, "Stock Soup")
	if _, err := sharedPool().Exec(ctx,
		"UPDATE meals SET allergens = $1 WHERE id = $2",
		[]string{"mustard"}, mealID,
	); err != nil {
		t.Fatalf("set meal allergens: %v", err)
	}

	itemID := makeItemWithAllergens(t, "Stock Cube", "soy")
	if _, err := q.AddMealIngredient(ctx, sqlc.AddMealIngredientParams{
		MealID:         mealID,
		ShoppingItemID: itemID,
		Quantity:       numeric(t, "1"),
		Unit:           pgtype.Text{String: "cube", Valid: true},
	}); err != nil {
		t.Fatalf("AddMealIngredient: %v", err)
	}
	if _, err := q.AddSubIngredient(ctx, sqlc.AddSubIngredientParams{
		ShoppingItemID: itemID,
		Name:           "Yeast Extract",
		Allergens:      []string{"gluten"},
	}); err != nil {
		t.Fatalf("AddSubIngredient: %v", err)
	}

	// ListSubIngredientsForMeal must reach the sub-ingredient via the ingredient.
	subs, err := q.ListSubIngredientsForMeal(ctx, mealID)
	if err != nil {
		t.Fatalf("ListSubIngredientsForMeal: %v", err)
	}
	if len(subs) != 1 || subs[0].Name != "Yeast Extract" {
		t.Errorf("expected one sub-ingredient for the meal, got %+v", subs)
	}

	// The derived allergen column must be the union of all three levels.
	rows, err := q.ListMealsWithIngredientCount(ctx, pgtype.Int4{Valid: false})
	if err != nil {
		t.Fatalf("ListMealsWithIngredientCount: %v", err)
	}
	var derived []string
	for _, r := range rows {
		if r.ID == mealID {
			derived = r.DerivedAllergens
			break
		}
	}
	if derived == nil {
		t.Fatal("meal not found in ListMealsWithIngredientCount")
	}
	sort.Strings(derived)
	if !containsAll(derived, "gluten", "mustard", "soy") {
		t.Errorf("expected derived allergens to union to [gluten mustard soy], got %v", derived)
	}
}

func TestIntegration_GetMealWithIngredients_IncludesIngredientAllergens(t *testing.T) {
	ctx := context.Background()
	q := sqlc.New(sharedPool())

	mealID := makeMeal(t, "Allergen Meal")
	itemID := makeItemWithAllergens(t, "Peanut Butter", "peanuts")
	if _, err := q.AddMealIngredient(ctx, sqlc.AddMealIngredientParams{
		MealID:         mealID,
		ShoppingItemID: itemID,
		Quantity:       numeric(t, "1"),
	}); err != nil {
		t.Fatalf("AddMealIngredient: %v", err)
	}

	rows, err := q.GetMealWithIngredients(ctx, mealID)
	if err != nil {
		t.Fatalf("GetMealWithIngredients: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one ingredient row, got %d", len(rows))
	}
	if !containsAll(rows[0].IngredientAllergens, "peanuts") {
		t.Errorf("expected ingredient allergen peanuts, got %v", rows[0].IngredientAllergens)
	}
}
