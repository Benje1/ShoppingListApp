package meals

import (
	"testing"

	sqlc "weekly-shopping-app/database/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
)

// ── packsToBuy ──────────────────────────────────────────────────────────────

func TestPacksToBuy(t *testing.T) {
	ppp := func(f float64) *float64 { return &f }
	cases := []struct {
		name       string
		perPortion *float64
		portions   int32
		packSize   float64
		want       int32
	}{
		{"legacy nil per-portion", nil, 4, 500, 0},
		{"pasta 100g x4 in 500g packs", ppp(100), 4, 500, 1},   // 400g -> 1 pack
		{"pasta 100g x6 in 500g packs", ppp(100), 6, 500, 2},   // 600g -> 2 packs
		{"one potato per person, 4-pack", ppp(1), 4, 4, 1},     // exactly one pack
		{"one potato x5, 4-pack", ppp(1), 5, 4, 2},             // 5 -> 2 packs
		{"half a tin per person, single tins", ppp(0.5), 3, 1, 2}, // 1.5 -> 2 tins
		{"zero portions treated as one", ppp(100), 0, 500, 1},
		{"zero pack size treated as one", ppp(2), 3, 0, 6},
	}
	for _, c := range cases {
		if got := packsToBuy(c.perPortion, c.portions, c.packSize); got != c.want {
			t.Errorf("%s: packsToBuy = %d, want %d", c.name, got, c.want)
		}
	}
}

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

// ── textOrEmpty ────────────────────────────────────────────────────────────────

func TestTextOrEmpty(t *testing.T) {
	if got := textOrEmpty(pgtype.Text{String: "x", Valid: true}); got != "x" {
		t.Errorf("expected %q, got %q", "x", got)
	}
	if got := textOrEmpty(pgtype.Text{Valid: false}); got != "" {
		t.Errorf("expected empty string for invalid Text, got %q", got)
	}
}

// ── allergensOrEmpty ───────────────────────────────────────────────────────────

func TestAllergensOrEmpty_Nil(t *testing.T) {
	got := allergensOrEmpty(nil)
	if got == nil {
		t.Fatal("expected non-nil slice so JSON encodes [] not null")
	}
	if len(got) != 0 {
		t.Fatalf("expected empty slice, got %v", got)
	}
}

func TestAllergensOrEmpty_PassesThrough(t *testing.T) {
	in := []string{"gluten", "dairy"}
	got := allergensOrEmpty(in)
	if len(got) != 2 || got[0] != "gluten" || got[1] != "dairy" {
		t.Fatalf("expected passthrough, got %v", got)
	}
}

// ── sanitizeAllergens ──────────────────────────────────────────────────────────

func TestSanitizeAllergens_LowercasesTrimsAndDedups(t *testing.T) {
	got, err := sanitizeAllergens([]string{"Gluten", " dairy ", "DAIRY", "gluten"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0] != "gluten" || got[1] != "dairy" {
		t.Fatalf("expected [gluten dairy], got %v", got)
	}
}

func TestSanitizeAllergens_SkipsEmptyEntries(t *testing.T) {
	got, err := sanitizeAllergens([]string{"", "   ", "eggs"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0] != "eggs" {
		t.Fatalf("expected [eggs], got %v", got)
	}
}

func TestSanitizeAllergens_NilReturnsNonNilEmpty(t *testing.T) {
	got, err := sanitizeAllergens(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil slice")
	}
	if len(got) != 0 {
		t.Fatalf("expected empty slice, got %v", got)
	}
}

func TestSanitizeAllergens_UnknownReturnsError(t *testing.T) {
	_, err := sanitizeAllergens([]string{"gluten", "kryptonite"})
	if err == nil {
		t.Fatal("expected error for unknown allergen")
	}
}

// ── buildMealResponse: meal-card fields ────────────────────────────────────────

func TestBuildMealResponse_CardFields(t *testing.T) {
	meal := sqlc.Meal{
		ID:              7,
		Name:            "Curry",
		DefaultPortions: 4,
		PhotoUrl:        pgtype.Text{String: "http://img/curry.jpg", Valid: true},
		Recipe:          pgtype.Text{String: "Simmer for 20 min", Valid: true},
		Allergens:       []string{"gluten", "dairy"},
	}
	resp := buildMealResponse(meal, nil)
	if resp.PhotoURL != "http://img/curry.jpg" {
		t.Errorf("expected photo url, got %q", resp.PhotoURL)
	}
	if resp.Recipe != "Simmer for 20 min" {
		t.Errorf("expected recipe, got %q", resp.Recipe)
	}
	if len(resp.Allergens) != 2 {
		t.Errorf("expected 2 allergens, got %v", resp.Allergens)
	}
}

func TestBuildMealResponse_NilAllergensEncodesEmpty(t *testing.T) {
	meal := sqlc.Meal{ID: 8, Name: "Plain", DefaultPortions: 1}
	resp := buildMealResponse(meal, nil)
	if resp.Allergens == nil {
		t.Fatal("expected non-nil allergens slice")
	}
	if resp.PhotoURL != "" || resp.Recipe != "" {
		t.Errorf("expected empty photo/recipe, got %q / %q", resp.PhotoURL, resp.Recipe)
	}
}

func TestBuildMealResponse_OptionalIngredient(t *testing.T) {
	meal := sqlc.Meal{ID: 9, Name: "Salad", DefaultPortions: 2}
	rows := []sqlc.GetMealWithIngredientsRow{
		{ShoppingItemID: 1, IngredientName: "Lettuce", Optional: false},
		{ShoppingItemID: 2, IngredientName: "Croutons", Optional: true},
	}
	resp := buildMealResponse(meal, rows)
	if resp.Ingredients[0].Optional {
		t.Error("expected Lettuce to be non-optional")
	}
	if !resp.Ingredients[1].Optional {
		t.Error("expected Croutons to be optional")
	}
}
