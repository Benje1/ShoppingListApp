package meals

import (
	"testing"

	sqlc "weekly-shopping-app/database/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
)

// ── toText ────────────────────────────────────────────────────────────────────

func TestToText_NonEmpty(t *testing.T) {
	result := toText("hello")
	if !result.Valid {
		t.Fatal("expected Valid=true for non-empty string")
	}
	if result.String != "hello" {
		t.Fatalf("expected %q, got %q", "hello", result.String)
	}
}

func TestToText_Empty(t *testing.T) {
	result := toText("")
	if result.Valid {
		t.Fatal("expected Valid=false for empty string")
	}
}

// ── toNumeric / numericToFloat roundtrip ──────────────────────────────────────

func TestNumericRoundtrip(t *testing.T) {
	cases := []float64{1, 0.5, 2.25, 100, 0.01}
	for _, c := range cases {
		n := toNumeric(c)
		got := numericToFloat(n)
		if got != c {
			t.Errorf("toNumeric(%.2f) -> numericToFloat = %.2f, want %.2f", c, got, c)
		}
	}
}

func TestNumericToFloat_InvalidReturnsOne(t *testing.T) {
	// An uninitialised Numeric (Valid=false) should return the safe default of 1
	got := numericToFloat(pgtype.Numeric{})
	if got != 1 {
		t.Fatalf("expected 1 for invalid Numeric, got %v", got)
	}
}

// ── nullableInt4 ──────────────────────────────────────────────────────────────

func TestNullableInt4_NonZero(t *testing.T) {
	result := nullableInt4(42)
	if !result.Valid {
		t.Fatal("expected Valid=true for non-zero int")
	}
	if result.Int32 != 42 {
		t.Fatalf("expected 42, got %d", result.Int32)
	}
}

func TestNullableInt4_Zero(t *testing.T) {
	result := nullableInt4(0)
	if result.Valid {
		t.Fatal("expected Valid=false for zero (represents no value)")
	}
}

// ── planScope ────────────────────────────────────────────────────────────────

func TestPlanScope_HouseholdScope(t *testing.T) {
	hid, uid := planScope(10, 5, "household")
	if !hid.Valid || hid.Int32 != 5 {
		t.Fatalf("expected household_id=5, got %+v", hid)
	}
	if uid.Valid {
		t.Fatal("expected user_id to be null for household scope")
	}
}

func TestPlanScope_PersonalScope(t *testing.T) {
	hid, uid := planScope(10, 5, "personal")
	if hid.Valid {
		t.Fatal("expected household_id to be null for personal scope")
	}
	if !uid.Valid || uid.Int32 != 10 {
		t.Fatalf("expected user_id=10, got %+v", uid)
	}
}

func TestPlanScope_HouseholdScopeButZeroID_FallsBackToPersonal(t *testing.T) {
	// household_id=0 means no household — should fall back to personal scope
	hid, uid := planScope(10, 0, "household")
	if hid.Valid {
		t.Fatal("expected household_id to be null when householdID=0")
	}
	if !uid.Valid || uid.Int32 != 10 {
		t.Fatalf("expected user_id=10, got %+v", uid)
	}
}

// ── buildMealResponse ─────────────────────────────────────────────────────────

func TestBuildMealResponse_NoIngredients(t *testing.T) {
	meal := sqlc.Meal{
		ID:              1,
		Name:            "Pasta",
		Description:     pgtype.Text{String: "A classic", Valid: true},
		DefaultPortions: 4,
	}
	resp := buildMealResponse(meal, nil)

	if resp.ID != 1 {
		t.Errorf("expected ID=1, got %d", resp.ID)
	}
	if resp.Name != "Pasta" {
		t.Errorf("expected Name=%q, got %q", "Pasta", resp.Name)
	}
	if resp.Description != "A classic" {
		t.Errorf("expected Description=%q, got %q", "A classic", resp.Description)
	}
	if resp.DefaultPortions != 4 {
		t.Errorf("expected DefaultPortions=4, got %d", resp.DefaultPortions)
	}
	if len(resp.Ingredients) != 0 {
		t.Errorf("expected 0 ingredients, got %d", len(resp.Ingredients))
	}
}

func TestBuildMealResponse_NoDescription(t *testing.T) {
	meal := sqlc.Meal{
		ID:              2,
		Name:            "Soup",
		Description:     pgtype.Text{Valid: false},
		DefaultPortions: 2,
	}
	resp := buildMealResponse(meal, nil)
	if resp.Description != "" {
		t.Errorf("expected empty description, got %q", resp.Description)
	}
}

func TestBuildMealResponse_WithIngredients(t *testing.T) {
	meal := sqlc.Meal{ID: 3, Name: "Stew", DefaultPortions: 6}

	n := pgtype.Numeric{}
	_ = n.Scan("2.50")

	rows := []sqlc.GetMealWithIngredientsRow{
		{
			ShoppingItemID:  10,
			IngredientName:  "Carrots",
			IngredientType:  sqlc.ShoppingItemTypeVegetable,
			Quantity:        n,
			Unit:            pgtype.Text{String: "g", Valid: true},
			PortionsPerUnit: 1,
		},
		{
			ShoppingItemID:  11,
			IngredientName:  "Onion",
			IngredientType:  sqlc.ShoppingItemTypeVegetable,
			Quantity:        n,
			Unit:            pgtype.Text{Valid: false}, // no unit
			PortionsPerUnit: 1,
		},
	}

	resp := buildMealResponse(meal, rows)
	if len(resp.Ingredients) != 2 {
		t.Fatalf("expected 2 ingredients, got %d", len(resp.Ingredients))
	}

	carrot := resp.Ingredients[0]
	if carrot.ItemName != "Carrots" {
		t.Errorf("expected ItemName=%q, got %q", "Carrots", carrot.ItemName)
	}
	if carrot.Unit != "g" {
		t.Errorf("expected Unit=%q, got %q", "g", carrot.Unit)
	}
	if carrot.Quantity != 2.5 {
		t.Errorf("expected Quantity=2.5, got %v", carrot.Quantity)
	}

	onion := resp.Ingredients[1]
	if onion.Unit != "" {
		t.Errorf("expected empty unit for onion, got %q", onion.Unit)
	}
}
