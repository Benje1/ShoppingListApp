package integration_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	sqlc "weekly-shopping-app/database/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
)

// toNumeric converts a float64 to the pgtype.Numeric the pantry queries expect.
func toNumeric(f float64) pgtype.Numeric {
	n := pgtype.Numeric{}
	_ = n.Scan(fmt.Sprintf("%.4f", f))
	return n
}

// numericToFloat converts a pgtype.Numeric back to float64 for assertions.
func numericToFloat(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, _ := n.Float64Value()
	return f.Float64
}

// ── UpsertPantryItem — personal scope ─────────────────────────────────────────

func TestIntegration_Pantry_Upsert_PersonalScope_Succeeds(t *testing.T) {
	ctx := context.Background()
	q := sqlc.New(sharedPool())
	uid, _, _ := makeUser(t)
	_, userID := personalScope(uid)

	entry, err := q.UpsertPantryItem(ctx, sqlc.UpsertPantryItemParams{
		ShoppingItemID:    SeedItems.Pasta,
		UserID:            userID,
		PortionsRemaining: toNumeric(4.0),
		ExpiresOn:         pgtype.Date{Valid: false},
	})
	if err != nil {
		t.Fatalf("UpsertPantryItem: %v", err)
	}
	if entry.ShoppingItemID != SeedItems.Pasta {
		t.Errorf("expected shopping_item_id=%d, got %d", SeedItems.Pasta, entry.ShoppingItemID)
	}
	if entry.Status != "fresh" {
		t.Errorf("expected status=fresh, got %q", entry.Status)
	}
	if got := numericToFloat(entry.PortionsRemaining); got != 4.0 {
		t.Errorf("expected portions_remaining=4.0, got %v", got)
	}
}

// ── UpsertPantryItem — household scope ────────────────────────────────────────

func TestIntegration_Pantry_Upsert_HouseholdScope_Succeeds(t *testing.T) {
	ctx := context.Background()
	q := sqlc.New(sharedPool())
	ownerID, _, _ := makeUser(t)
	householdID := makeHousehold(t, ownerID)
	hid, uid := householdScope(householdID)

	entry, err := q.UpsertPantryItem(ctx, sqlc.UpsertPantryItemParams{
		ShoppingItemID:    SeedItems.Rice,
		HouseholdID:       hid,
		UserID:            uid,
		PortionsRemaining: toNumeric(3.0),
		ExpiresOn:         pgtype.Date{Valid: false},
	})
	if err != nil {
		t.Fatalf("UpsertPantryItem: %v", err)
	}
	if entry.HouseholdID.Int32 != householdID {
		t.Errorf("expected household_id=%d, got %d", householdID, entry.HouseholdID.Int32)
	}
}

// ── Accumulation on conflict ──────────────────────────────────────────────────

func TestIntegration_Pantry_Upsert_SameItemTwice_AccumulatesPortions(t *testing.T) {
	ctx := context.Background()
	q := sqlc.New(sharedPool())
	ownerID, _, _ := makeUser(t)
	householdID := makeHousehold(t, ownerID)
	hid, uid := householdScope(householdID)

	params := sqlc.UpsertPantryItemParams{
		ShoppingItemID:    SeedItems.Flour,
		HouseholdID:       hid,
		UserID:            uid,
		PortionsRemaining: toNumeric(2.0),
		ExpiresOn:         pgtype.Date{Valid: false},
	}
	if _, err := q.UpsertPantryItem(ctx, params); err != nil {
		t.Fatalf("first UpsertPantryItem: %v", err)
	}
	second, err := q.UpsertPantryItem(ctx, params)
	if err != nil {
		t.Fatalf("second UpsertPantryItem: %v", err)
	}
	if got := numericToFloat(second.PortionsRemaining); got != 4.0 {
		t.Errorf("expected accumulated portions=4.0, got %v", got)
	}
}

// ── GetPantry ─────────────────────────────────────────────────────────────────

func TestIntegration_Pantry_GetPantry_ReturnsEntry(t *testing.T) {
	ctx := context.Background()
	q := sqlc.New(sharedPool())
	uid, _, _ := makeUser(t)
	_, userID := personalScope(uid)

	if _, err := q.UpsertPantryItem(ctx, sqlc.UpsertPantryItemParams{
		ShoppingItemID:    SeedItems.Oats,
		UserID:            userID,
		PortionsRemaining: toNumeric(2.0),
		ExpiresOn:         pgtype.Date{Valid: false},
	}); err != nil {
		t.Fatalf("UpsertPantryItem: %v", err)
	}

	rows, err := q.GetPantry(ctx, sqlc.GetPantryParams{
		HouseholdID: pgtype.Int4{Valid: false},
		UserID:      userID,
	})
	if err != nil {
		t.Fatalf("GetPantry: %v", err)
	}
	found := false
	for _, r := range rows {
		if r.ShoppingItemID == SeedItems.Oats {
			found = true
			break
		}
	}
	if !found {
		t.Error("upserted item not found in GetPantry results")
	}
}

// ── RemovePantryItem ──────────────────────────────────────────────────────────

func TestIntegration_Pantry_Remove_Succeeds(t *testing.T) {
	ctx := context.Background()
	q := sqlc.New(sharedPool())
	uid, _, _ := makeUser(t)
	_, userID := personalScope(uid)

	entry, err := q.UpsertPantryItem(ctx, sqlc.UpsertPantryItemParams{
		ShoppingItemID:    SeedItems.Chickpeas,
		UserID:            userID,
		PortionsRemaining: toNumeric(1.0),
		ExpiresOn:         pgtype.Date{Valid: false},
	})
	if err != nil {
		t.Fatalf("UpsertPantryItem: %v", err)
	}

	if err := q.RemovePantryItem(ctx, entry.ID); err != nil {
		t.Fatalf("RemovePantryItem: %v", err)
	}

	rows, err := q.GetPantry(ctx, sqlc.GetPantryParams{
		HouseholdID: pgtype.Int4{Valid: false},
		UserID:      userID,
	})
	if err != nil {
		t.Fatalf("GetPantry after remove: %v", err)
	}
	for _, r := range rows {
		if r.ID == entry.ID {
			t.Error("pantry entry still present after removal")
		}
	}
}

// ── DecrementPantryPortions ───────────────────────────────────────────────────

func TestIntegration_Pantry_Decrement_ReducesPortions(t *testing.T) {
	ctx := context.Background()
	q := sqlc.New(sharedPool())
	uid, _, _ := makeUser(t)
	_, userID := personalScope(uid)

	if _, err := q.UpsertPantryItem(ctx, sqlc.UpsertPantryItemParams{
		ShoppingItemID:    SeedItems.Lentils,
		UserID:            userID,
		PortionsRemaining: toNumeric(6.0),
		ExpiresOn:         pgtype.Date{Valid: false},
	}); err != nil {
		t.Fatalf("UpsertPantryItem: %v", err)
	}

	updated, err := q.DecrementPantryPortions(ctx, sqlc.DecrementPantryPortionsParams{
		ShoppingItemID:    SeedItems.Lentils,
		HouseholdID:       pgtype.Int4{Valid: false},
		UserID:            userID,
		PortionsRemaining: toNumeric(2.0),
	})
	if err != nil {
		t.Fatalf("DecrementPantryPortions: %v", err)
	}
	if got := numericToFloat(updated.PortionsRemaining); got != 4.0 {
		t.Errorf("expected 4.0 portions after decrement, got %v", got)
	}
}

func TestIntegration_Pantry_Decrement_ClampsToZero(t *testing.T) {
	ctx := context.Background()
	q := sqlc.New(sharedPool())
	ownerID, _, _ := makeUser(t)
	householdID := makeHousehold(t, ownerID)
	hid, uid := householdScope(householdID)

	// Fresh catalogue item so starting portions are unambiguous.
	freshItem, err := q.CreateShoppingItem(ctx, sqlc.CreateShoppingItemParams{
		Name:            fmt.Sprintf("ClampItem_%d", time.Now().UnixNano()),
		ItemType:        sqlc.ShoppingItemTypePantry,
		PortionsPerUnit: 1,
	})
	if err != nil {
		t.Fatalf("CreateShoppingItem: %v", err)
	}

	if _, err := q.UpsertPantryItem(ctx, sqlc.UpsertPantryItemParams{
		ShoppingItemID:    freshItem.ID,
		HouseholdID:       hid,
		UserID:            uid,
		PortionsRemaining: toNumeric(1.0),
		ExpiresOn:         pgtype.Date{Valid: false},
	}); err != nil {
		t.Fatalf("UpsertPantryItem: %v", err)
	}

	updated, err := q.DecrementPantryPortions(ctx, sqlc.DecrementPantryPortionsParams{
		ShoppingItemID:    freshItem.ID,
		HouseholdID:       hid,
		UserID:            uid,
		PortionsRemaining: toNumeric(999.0),
	})
	if err != nil {
		t.Fatalf("DecrementPantryPortions: %v", err)
	}
	if got := numericToFloat(updated.PortionsRemaining); got != 0.0 {
		t.Errorf("expected portions to clamp to 0, got %v", got)
	}
}

// ── ExpirePantryItems ─────────────────────────────────────────────────────────

func TestIntegration_Pantry_ExpiryJob_MarksExpiredAndExpiringSoon(t *testing.T) {
	ctx := context.Background()
	q := sqlc.New(sharedPool())
	uid, _, _ := makeUser(t)
	_, userID := personalScope(uid)

	yesterday := pgtype.Date{Time: time.Now().AddDate(0, 0, -1), Valid: true, InfinityModifier: pgtype.Finite}
	tomorrow := pgtype.Date{Time: time.Now().AddDate(0, 0, 1), Valid: true, InfinityModifier: pgtype.Finite}

	nano := time.Now().UnixNano()
	expiredItem, err := q.CreateShoppingItem(ctx, sqlc.CreateShoppingItemParams{
		Name: fmt.Sprintf("ExpiredItem_%d", nano), ItemType: sqlc.ShoppingItemTypeDairy, PortionsPerUnit: 1,
	})
	if err != nil {
		t.Fatalf("CreateShoppingItem (expired): %v", err)
	}
	soonItem, err := q.CreateShoppingItem(ctx, sqlc.CreateShoppingItemParams{
		Name: fmt.Sprintf("SoonItem_%d", nano), ItemType: sqlc.ShoppingItemTypeDairy, PortionsPerUnit: 1,
	})
	if err != nil {
		t.Fatalf("CreateShoppingItem (soon): %v", err)
	}

	for _, p := range []struct {
		id      int32
		expires pgtype.Date
	}{
		{expiredItem.ID, yesterday},
		{soonItem.ID, tomorrow},
	} {
		if _, err := q.UpsertPantryItem(ctx, sqlc.UpsertPantryItemParams{
			ShoppingItemID:    p.id,
			UserID:            userID,
			PortionsRemaining: toNumeric(1.0),
			ExpiresOn:         p.expires,
		}); err != nil {
			t.Fatalf("UpsertPantryItem id=%d: %v", p.id, err)
		}
	}

	if _, err := q.ExpirePantryItems(ctx); err != nil {
		t.Fatalf("ExpirePantryItems: %v", err)
	}

	rows, err := q.GetPantry(ctx, sqlc.GetPantryParams{
		HouseholdID: pgtype.Int4{Valid: false},
		UserID:      userID,
	})
	if err != nil {
		t.Fatalf("GetPantry after expiry: %v", err)
	}

	statuses := make(map[int32]string)
	for _, r := range rows {
		statuses[r.ShoppingItemID] = r.Status
	}
	if got := statuses[expiredItem.ID]; got != "expired" {
		t.Errorf("expected status=expired, got %q", got)
	}
	if got := statuses[soonItem.ID]; got != "expiring_soon" {
		t.Errorf("expected status=expiring_soon, got %q", got)
	}
}
