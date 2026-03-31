package authntication_test

// pantry_service_test.go
// Tests for pantry business logic using FakePantryRepo.
// No real database or HTTP server is required.

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// ── Add to pantry ─────────────────────────────────────────────────────────────

func TestPantry_AddNewItem(t *testing.T) {
	repo := NewFakePantryRepo()
	uid := pgtype.Int4{Int32: 1, Valid: true}

	item := repo.Upsert(context.Background(), 10, 3.0, pgtype.Int4{Valid: false}, uid, pgtype.Date{Valid: false})

	if item.ShoppingItemID != 10 {
		t.Errorf("expected shopping_item_id=10, got %d", item.ShoppingItemID)
	}
	if item.PortionsRemaining != 3.0 {
		t.Errorf("expected portions_remaining=3.0, got %v", item.PortionsRemaining)
	}
	if item.Status != "fresh" {
		t.Errorf("expected status=fresh, got %q", item.Status)
	}
}

func TestPantry_AddAccumulatesPortions(t *testing.T) {
	repo := NewFakePantryRepo()
	uid := pgtype.Int4{Int32: 1, Valid: true}
	hid := pgtype.Int4{Valid: false}

	repo.Upsert(context.Background(), 5, 2.0, hid, uid, pgtype.Date{Valid: false})
	updated := repo.Upsert(context.Background(), 5, 1.5, hid, uid, pgtype.Date{Valid: false})

	if updated.PortionsRemaining != 3.5 {
		t.Errorf("expected accumulated portions=3.5, got %v", updated.PortionsRemaining)
	}
}

func TestPantry_HouseholdAndPersonalScopesAreIndependent(t *testing.T) {
	repo := NewFakePantryRepo()
	uid := pgtype.Int4{Int32: 1, Valid: true}
	hid := pgtype.Int4{Int32: 99, Valid: true}

	repo.Upsert(context.Background(), 7, 2.0, pgtype.Int4{Valid: false}, uid, pgtype.Date{Valid: false})
	repo.Upsert(context.Background(), 7, 5.0, hid, pgtype.Int4{Valid: false}, pgtype.Date{Valid: false})

	personal, ok := repo.Get(context.Background(), 7, pgtype.Int4{Valid: false}, uid)
	if !ok {
		t.Fatal("expected personal pantry entry")
	}
	if personal.PortionsRemaining != 2.0 {
		t.Errorf("personal portions should be 2.0, got %v", personal.PortionsRemaining)
	}

	household, ok := repo.Get(context.Background(), 7, hid, pgtype.Int4{Valid: false})
	if !ok {
		t.Fatal("expected household pantry entry")
	}
	if household.PortionsRemaining != 5.0 {
		t.Errorf("household portions should be 5.0, got %v", household.PortionsRemaining)
	}
}

// ── Remove from pantry ────────────────────────────────────────────────────────

func TestPantry_RemoveDeletesEntry(t *testing.T) {
	repo := NewFakePantryRepo()
	uid := pgtype.Int4{Int32: 1, Valid: true}

	item := repo.Upsert(context.Background(), 3, 1.0, pgtype.Int4{Valid: false}, uid, pgtype.Date{Valid: false})
	if err := repo.Remove(context.Background(), item.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.Count() != 0 {
		t.Errorf("expected empty pantry, got %d entries", repo.Count())
	}
}

func TestPantry_RemoveNonExistentEntryErrors(t *testing.T) {
	repo := NewFakePantryRepo()
	err := repo.Remove(context.Background(), 9999)
	if err == nil {
		t.Fatal("expected error removing non-existent pantry entry")
	}
}

// ── Decrement portions ────────────────────────────────────────────────────────

func TestPantry_DecrementReducesPortions(t *testing.T) {
	repo := NewFakePantryRepo()
	uid := pgtype.Int4{Int32: 1, Valid: true}
	hid := pgtype.Int4{Valid: false}

	repo.Upsert(context.Background(), 8, 6.0, hid, uid, pgtype.Date{Valid: false})
	repo.Decrement(context.Background(), 8, 2.0, hid, uid)

	item, ok := repo.Get(context.Background(), 8, hid, uid)
	if !ok {
		t.Fatal("expected pantry entry after decrement")
	}
	if item.PortionsRemaining != 4.0 {
		t.Errorf("expected 4.0 portions after decrement, got %v", item.PortionsRemaining)
	}
}

func TestPantry_DecrementClampsToZero(t *testing.T) {
	repo := NewFakePantryRepo()
	uid := pgtype.Int4{Int32: 1, Valid: true}
	hid := pgtype.Int4{Valid: false}

	repo.Upsert(context.Background(), 9, 1.0, hid, uid, pgtype.Date{Valid: false})
	repo.Decrement(context.Background(), 9, 100.0, hid, uid)

	item, ok := repo.Get(context.Background(), 9, hid, uid)
	if !ok {
		t.Fatal("expected pantry entry")
	}
	if item.PortionsRemaining < 0 {
		t.Errorf("portions_remaining should not be negative, got %v", item.PortionsRemaining)
	}
}

func TestPantry_DecrementMissingItemIsNoOp(t *testing.T) {
	repo := NewFakePantryRepo()
	uid := pgtype.Int4{Int32: 1, Valid: true}
	// Should not panic or error when the item doesn't exist
	repo.Decrement(context.Background(), 999, 1.0, pgtype.Int4{Valid: false}, uid)
}

// ── Expiry date handling ──────────────────────────────────────────────────────

func TestPantry_ExpiryDateStoredCorrectly(t *testing.T) {
	repo := NewFakePantryRepo()
	uid := pgtype.Int4{Int32: 1, Valid: true}

	expiresOn := pgtype.Date{
		Time:             time.Now().AddDate(0, 0, 7),
		Valid:            true,
		InfinityModifier: pgtype.Finite,
	}

	item := repo.Upsert(context.Background(), 11, 2.0, pgtype.Int4{Valid: false}, uid, expiresOn)
	if !item.ExpiresOn.Valid {
		t.Error("expected ExpiresOn to be valid")
	}
}

func TestPantry_NoExpiryDateIsNullable(t *testing.T) {
	repo := NewFakePantryRepo()
	uid := pgtype.Int4{Int32: 1, Valid: true}

	item := repo.Upsert(context.Background(), 12, 1.0, pgtype.Int4{Valid: false}, uid, pgtype.Date{Valid: false})
	if item.ExpiresOn.Valid {
		t.Error("expected ExpiresOn to be null when not provided")
	}
}

// ── Get ───────────────────────────────────────────────────────────────────────

func TestPantry_GetReturnsFalseForMissingItem(t *testing.T) {
	repo := NewFakePantryRepo()
	uid := pgtype.Int4{Int32: 1, Valid: true}
	_, ok := repo.Get(context.Background(), 999, pgtype.Int4{Valid: false}, uid)
	if ok {
		t.Fatal("expected ok=false for missing pantry item")
	}
}

func TestPantry_CountTracksEntries(t *testing.T) {
	repo := NewFakePantryRepo()
	uid := pgtype.Int4{Int32: 1, Valid: true}

	if repo.Count() != 0 {
		t.Fatalf("expected 0 initial entries, got %d", repo.Count())
	}
	repo.Upsert(context.Background(), 1, 1.0, pgtype.Int4{Valid: false}, uid, pgtype.Date{Valid: false})
	repo.Upsert(context.Background(), 2, 1.0, pgtype.Int4{Valid: false}, uid, pgtype.Date{Valid: false})
	if repo.Count() != 2 {
		t.Errorf("expected 2 entries, got %d", repo.Count())
	}
}
