package integration_test

import (
	"context"
	"testing"

	sqlc "weekly-shopping-app/database/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
)

// personalScope returns pgtype values for a personal (user-scoped) list entry.
func personalScope(userID int32) (hid pgtype.Int4, uid pgtype.Int4) {
	return pgtype.Int4{Valid: false}, pgtype.Int4{Int32: userID, Valid: true}
}

// householdScope returns pgtype values for a household-scoped list entry.
func householdScope(householdID int32) (hid pgtype.Int4, uid pgtype.Int4) {
	return pgtype.Int4{Int32: householdID, Valid: true}, pgtype.Int4{Valid: false}
}

// ── CreateShoppingItem ────────────────────────────────────────────────────────

func TestIntegration_CreateShoppingItem_Succeeds(t *testing.T) {
	item, err := sqlc.New(TestPool()).CreateShoppingItem(context.Background(), sqlc.CreateShoppingItemParams{
		Name:            "New Catalogue Item",
		ItemType:        sqlc.ShoppingItemTypePantry,
		PortionsPerUnit: 4,
	})
	if err != nil {
		t.Fatalf("CreateShoppingItem: %v", err)
	}
	if item.ID == 0 {
		t.Error("expected a non-zero ID")
	}
	if item.Name != "New Catalogue Item" {
		t.Errorf("expected Name=New Catalogue Item, got %q", item.Name)
	}
}

// ── Add to list — personal scope ─────────────────────────────────────────────

func TestIntegration_ShoppingList_AddPersonal_Succeeds(t *testing.T) {
	ctx := context.Background()
	q := sqlc.New(TestPool())
	uid, _, _ := makeUser(t)
	hid, userID := personalScope(uid)

	entry, err := q.AddToShoppingList(ctx, sqlc.AddToShoppingListParams{
		ShoppingItemID: SeedItems.Milk,
		Quantity:       2,
		HouseholdID:    hid,
		UserID:         userID,
	})
	if err != nil {
		t.Fatalf("AddToShoppingList: %v", err)
	}
	if entry.ShoppingItemID != SeedItems.Milk {
		t.Errorf("expected item_id=%d, got %d", SeedItems.Milk, entry.ShoppingItemID)
	}
	if entry.Quantity != 2 {
		t.Errorf("expected quantity=2, got %d", entry.Quantity)
	}
	if entry.UserID.Int32 != uid {
		t.Errorf("expected user_id=%d, got %d", uid, entry.UserID.Int32)
	}
}

// ── Add to list — household scope ─────────────────────────────────────────────

func TestIntegration_ShoppingList_AddHousehold_Succeeds(t *testing.T) {
	ctx := context.Background()
	q := sqlc.New(TestPool())
	ownerID, _, _ := makeUser(t)
	householdID := makeHousehold(t, ownerID)
	hid, uid := householdScope(householdID)

	entry, err := q.AddToShoppingList(ctx, sqlc.AddToShoppingListParams{
		ShoppingItemID: SeedItems.Bread,
		Quantity:       1,
		HouseholdID:    hid,
		UserID:         uid,
	})
	if err != nil {
		t.Fatalf("AddToShoppingList: %v", err)
	}
	if entry.HouseholdID.Int32 != householdID {
		t.Errorf("expected household_id=%d, got %d", householdID, entry.HouseholdID.Int32)
	}
}

// ── Quantity accumulation on conflict ─────────────────────────────────────────

func TestIntegration_ShoppingList_AddSameItemTwice_AccumulatesQuantity(t *testing.T) {
	ctx := context.Background()
	q := sqlc.New(TestPool())
	ownerID, _, _ := makeUser(t)
	householdID := makeHousehold(t, ownerID)
	hid, uid := householdScope(householdID)

	params := sqlc.AddToShoppingListParams{
		ShoppingItemID: SeedItems.Eggs,
		Quantity:       3,
		HouseholdID:    hid,
		UserID:         uid,
	}
	if _, err := q.AddToShoppingList(ctx, params); err != nil {
		t.Fatalf("first AddToShoppingList: %v", err)
	}
	second, err := q.AddToShoppingList(ctx, params)
	if err != nil {
		t.Fatalf("second AddToShoppingList: %v", err)
	}
	if second.Quantity != 6 {
		t.Errorf("expected accumulated quantity=6, got %d", second.Quantity)
	}
}

// ── Remove from list ──────────────────────────────────────────────────────────

func TestIntegration_ShoppingList_Remove_Succeeds(t *testing.T) {
	ctx := context.Background()
	q := sqlc.New(TestPool())
	uid, _, _ := makeUser(t)
	hid, userID := personalScope(uid)

	entry, err := q.AddToShoppingList(ctx, sqlc.AddToShoppingListParams{
		ShoppingItemID: SeedItems.Butter,
		Quantity:       1,
		HouseholdID:    hid,
		UserID:         userID,
	})
	if err != nil {
		t.Fatalf("AddToShoppingList: %v", err)
	}

	if err := q.RemoveFromShoppingList(ctx, entry.ID); err != nil {
		t.Fatalf("RemoveFromShoppingList: %v", err)
	}

	rows, err := TestPool().Query(ctx, "SELECT id FROM shopping_list WHERE id = $1", entry.ID)
	if err != nil {
		t.Fatalf("query after remove: %v", err)
	}
	defer rows.Close()
	if rows.Next() {
		t.Error("expected row to be deleted, but it still exists")
	}
}

// ── Household scope is isolated from other households ─────────────────────────

func TestIntegration_ShoppingList_HouseholdScope_Isolated(t *testing.T) {
	ctx := context.Background()
	q := sqlc.New(TestPool())

	owner1, _, _ := makeUser(t)
	owner2, _, _ := makeUser(t)
	hid1 := makeHousehold(t, owner1)
	hid2 := makeHousehold(t, owner2)

	h1, _ := householdScope(hid1)
	h2, _ := householdScope(hid2)

	if _, err := q.AddToShoppingList(ctx, sqlc.AddToShoppingListParams{
		ShoppingItemID: SeedItems.Cheese,
		Quantity:       1,
		HouseholdID:    h1,
	}); err != nil {
		t.Fatalf("AddToShoppingList h1: %v", err)
	}

	// Household 2 should see no entries for that item.
	rows, err := TestPool().Query(ctx,
		"SELECT id FROM shopping_list WHERE household_id = $1 AND shopping_item_id = $2",
		hid2, SeedItems.Cheese,
	)
	if err != nil {
		t.Fatalf("query h2: %v", err)
	}
	defer rows.Close()
	if rows.Next() {
		t.Error("household 2 should not see household 1's shopping list entry")
	}
}

// ── Have-it ───────────────────────────────────────────────────────────────────

func TestIntegration_ShoppingList_MarkHaveIt_AppearsInGetHaveIt(t *testing.T) {
	ctx := context.Background()
	q := sqlc.New(TestPool())
	uid, _, _ := makeUser(t)
	_, userID := personalScope(uid)
	noHousehold := pgtype.Int4{Valid: false}

	if _, err := q.MarkHaveIt(ctx, sqlc.MarkHaveItParams{
		ShoppingItemID: SeedItems.Cheese,
		HouseholdID:    noHousehold,
		UserID:         userID,
	}); err != nil {
		t.Fatalf("MarkHaveIt: %v", err)
	}

	rows, err := q.GetHaveIt(ctx, sqlc.GetHaveItParams{
		HouseholdID: noHousehold,
		UserID:      userID,
	})
	if err != nil {
		t.Fatalf("GetHaveIt: %v", err)
	}
	found := false
	for _, r := range rows {
		if r.ShoppingItemID == SeedItems.Cheese {
			found = true
			break
		}
	}
	if !found {
		t.Error("marked item not found in GetHaveIt results")
	}
}

func TestIntegration_ShoppingList_UnmarkHaveIt_DisappearsFromGetHaveIt(t *testing.T) {
	ctx := context.Background()
	q := sqlc.New(TestPool())
	uid, _, _ := makeUser(t)
	_, userID := personalScope(uid)
	noHousehold := pgtype.Int4{Valid: false}

	if _, err := q.MarkHaveIt(ctx, sqlc.MarkHaveItParams{
		ShoppingItemID: SeedItems.Oats,
		HouseholdID:    noHousehold,
		UserID:         userID,
	}); err != nil {
		t.Fatalf("MarkHaveIt: %v", err)
	}

	if err := q.UnmarkHaveIt(ctx, sqlc.UnmarkHaveItParams{
		ShoppingItemID: SeedItems.Oats,
		HouseholdID:    noHousehold,
		UserID:         userID,
	}); err != nil {
		t.Fatalf("UnmarkHaveIt: %v", err)
	}

	rows, err := q.GetHaveIt(ctx, sqlc.GetHaveItParams{
		HouseholdID: noHousehold,
		UserID:      userID,
	})
	if err != nil {
		t.Fatalf("GetHaveIt: %v", err)
	}
	for _, r := range rows {
		if r.ShoppingItemID == SeedItems.Oats {
			t.Error("unmarked item still appears in GetHaveIt results")
		}
	}
}
