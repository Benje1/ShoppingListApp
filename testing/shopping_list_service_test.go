package authntication_test

// shopping_list_service_test.go
// Tests for shopping list business logic that can run without a real database.
// These exercise the fake repo directly rather than the HTTP layer.

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func personalUID(id int32) pgtype.Int4 {
	return pgtype.Int4{Int32: id, Valid: true}
}

func householdHID(id int32) pgtype.Int4 {
	return pgtype.Int4{Int32: id, Valid: true}
}

var noHousehold = pgtype.Int4{Valid: false}

// ── Add to list ───────────────────────────────────────────────────────────────

func TestShoppingList_AddNewItem(t *testing.T) {
	repo := NewFakeShoppingListRepo()
	uid := personalUID(10)

	entry, err := repo.Add(context.Background(), 42, 2, noHousehold, uid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.ShoppingItemID != 42 {
		t.Errorf("expected item_id=42, got %d", entry.ShoppingItemID)
	}
	if entry.Quantity != 2 {
		t.Errorf("expected quantity=2, got %d", entry.Quantity)
	}
}

func TestShoppingList_AddAccumulatesQuantity(t *testing.T) {
	repo := NewFakeShoppingListRepo()
	uid := personalUID(10)

	_, _ = repo.Add(context.Background(), 5, 1, noHousehold, uid)
	updated, err := repo.Add(context.Background(), 5, 3, noHousehold, uid)
	if err != nil {
		t.Fatalf("unexpected error on second add: %v", err)
	}
	if updated.Quantity != 4 {
		t.Errorf("expected accumulated quantity=4, got %d", updated.Quantity)
	}
}

func TestShoppingList_AddDifferentScopesAreIndependent(t *testing.T) {
	repo := NewFakeShoppingListRepo()
	uid := personalUID(10)
	hid := householdHID(99)

	_, _ = repo.Add(context.Background(), 7, 1, noHousehold, uid)
	_, _ = repo.Add(context.Background(), 7, 1, hid, pgtype.Int4{Valid: false})

	personalEntries := repo.List(context.Background(), noHousehold, uid)
	if len(personalEntries) != 1 {
		t.Errorf("expected 1 personal entry, got %d", len(personalEntries))
	}
	householdEntries := repo.List(context.Background(), hid, pgtype.Int4{Valid: false})
	if len(householdEntries) != 1 {
		t.Errorf("expected 1 household entry, got %d", len(householdEntries))
	}
}

// ── Remove from list ──────────────────────────────────────────────────────────

func TestShoppingList_RemoveDeletesEntry(t *testing.T) {
	repo := NewFakeShoppingListRepo()
	uid := personalUID(10)

	entry, _ := repo.Add(context.Background(), 3, 1, noHousehold, uid)
	if err := repo.Remove(context.Background(), entry.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	remaining := repo.List(context.Background(), noHousehold, uid)
	if len(remaining) != 0 {
		t.Errorf("expected list to be empty after remove, got %d entries", len(remaining))
	}
}

func TestShoppingList_RemoveNonExistentEntryReturnsError(t *testing.T) {
	repo := NewFakeShoppingListRepo()
	err := repo.Remove(context.Background(), 9999)
	if err == nil {
		t.Fatal("expected error when removing non-existent entry")
	}
}

// ── Have-it ───────────────────────────────────────────────────────────────────

func TestShoppingList_MarkAndUnmarkHaveIt(t *testing.T) {
	repo := NewFakeShoppingListRepo()

	if err := repo.MarkHaveIt(context.Background(), 10); err != nil {
		t.Fatalf("MarkHaveIt returned unexpected error: %v", err)
	}
	if !repo.HasItem(10) {
		t.Fatal("expected item 10 to be in have-it set after mark")
	}

	if err := repo.UnmarkHaveIt(context.Background(), 10); err != nil {
		t.Fatalf("UnmarkHaveIt returned unexpected error: %v", err)
	}
	if repo.HasItem(10) {
		t.Fatal("expected item 10 to be absent from have-it set after unmark")
	}
}

func TestShoppingList_UnmarkNotPresentIsIdempotent(t *testing.T) {
	repo := NewFakeShoppingListRepo()
	// Unmarking something that was never marked should not error
	if err := repo.UnmarkHaveIt(context.Background(), 999); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestShoppingList_HaveItIsolatedPerItem(t *testing.T) {
	repo := NewFakeShoppingListRepo()

	_ = repo.MarkHaveIt(context.Background(), 1)
	_ = repo.MarkHaveIt(context.Background(), 2)

	if !repo.HasItem(1) {
		t.Error("expected item 1 to be in have-it set")
	}
	if !repo.HasItem(2) {
		t.Error("expected item 2 to be in have-it set")
	}

	_ = repo.UnmarkHaveIt(context.Background(), 1)
	if repo.HasItem(1) {
		t.Error("item 1 should have been removed")
	}
	if !repo.HasItem(2) {
		t.Error("item 2 should still be in have-it set")
	}
}

// ── List filtering ────────────────────────────────────────────────────────────

func TestShoppingList_ListReturnsOnlyMatchingScope(t *testing.T) {
	repo := NewFakeShoppingListRepo()
	uid1 := personalUID(1)
	uid2 := personalUID(2)

	_, _ = repo.Add(context.Background(), 10, 1, noHousehold, uid1)
	_, _ = repo.Add(context.Background(), 20, 1, noHousehold, uid2)

	user1List := repo.List(context.Background(), noHousehold, uid1)
	if len(user1List) != 1 {
		t.Errorf("expected 1 entry for user 1, got %d", len(user1List))
	}
	if user1List[0].ShoppingItemID != 10 {
		t.Errorf("expected item_id=10, got %d", user1List[0].ShoppingItemID)
	}
}

func TestShoppingList_EmptyListForUnknownUser(t *testing.T) {
	repo := NewFakeShoppingListRepo()
	entries := repo.List(context.Background(), noHousehold, personalUID(999))
	if len(entries) != 0 {
		t.Errorf("expected empty list for unknown user, got %d entries", len(entries))
	}
}
