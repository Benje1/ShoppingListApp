package authntication_test

// household_service_test.go
// Tests for household and invite logic using FakeHouseholdRepo.
// No real database required.

import (
	"context"
	"testing"
)

// ── Insert household ──────────────────────────────────────────────────────────

func TestHousehold_InsertCreatesEntry(t *testing.T) {
	repo := NewFakeHouseholdRepo()

	h, err := repo.InsertHousehold(context.Background(), 3, "The Smiths")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.NumPeople != 3 {
		t.Errorf("expected NumPeople=3, got %d", h.NumPeople)
	}
	if !h.Name.Valid || h.Name.String != "The Smiths" {
		t.Errorf("expected Name=The Smiths, got %+v", h.Name)
	}
}

func TestHousehold_InsertAutoIncrementsIDs(t *testing.T) {
	repo := NewFakeHouseholdRepo()

	h1, _ := repo.InsertHousehold(context.Background(), 1, "A")
	h2, _ := repo.InsertHousehold(context.Background(), 2, "B")

	if h1.HouseholdID == h2.HouseholdID {
		t.Errorf("expected distinct IDs, both are %d", h1.HouseholdID)
	}
}

// ── Get household ─────────────────────────────────────────────────────────────

func TestHousehold_GetReturnsInserted(t *testing.T) {
	repo := NewFakeHouseholdRepo()

	inserted, _ := repo.InsertHousehold(context.Background(), 2, "Casa Nova")
	got, err := repo.GetHousehold(context.Background(), inserted.HouseholdID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.HouseholdID != inserted.HouseholdID {
		t.Errorf("expected ID %d, got %d", inserted.HouseholdID, got.HouseholdID)
	}
}

func TestHousehold_GetMissingReturnsError(t *testing.T) {
	repo := NewFakeHouseholdRepo()
	_, err := repo.GetHousehold(context.Background(), 9999)
	if err == nil {
		t.Fatal("expected error for unknown household ID")
	}
}

// ── Rename household ──────────────────────────────────────────────────────────

func TestHousehold_RenameUpdatesName(t *testing.T) {
	repo := NewFakeHouseholdRepo()

	h, _ := repo.InsertHousehold(context.Background(), 1, "Old Name")
	renamed, err := repo.RenameHousehold(context.Background(), h.HouseholdID, "New Name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if renamed.Name.String != "New Name" {
		t.Errorf("expected Name=New Name, got %q", renamed.Name.String)
	}
}

func TestHousehold_RenameNonExistentReturnsError(t *testing.T) {
	repo := NewFakeHouseholdRepo()
	_, err := repo.RenameHousehold(context.Background(), 9999, "Anything")
	if err == nil {
		t.Fatal("expected error renaming non-existent household")
	}
}

// ── Delete household ──────────────────────────────────────────────────────────

func TestHousehold_DeleteRemovesEntry(t *testing.T) {
	repo := NewFakeHouseholdRepo()

	h, _ := repo.InsertHousehold(context.Background(), 1, "Gone")
	if err := repo.DeleteHousehold(context.Background(), h.HouseholdID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err := repo.GetHousehold(context.Background(), h.HouseholdID)
	if err == nil {
		t.Fatal("expected error fetching deleted household")
	}
}

func TestHousehold_DeleteNonExistentReturnsError(t *testing.T) {
	repo := NewFakeHouseholdRepo()
	err := repo.DeleteHousehold(context.Background(), 9999)
	if err == nil {
		t.Fatal("expected error deleting non-existent household")
	}
}

// ── Members ───────────────────────────────────────────────────────────────────

func TestHousehold_AddMemberIsRetrievable(t *testing.T) {
	repo := NewFakeHouseholdRepo()
	h, _ := repo.InsertHousehold(context.Background(), 2, "Family")

	repo.AddMember(context.Background(), h.HouseholdID, 42)
	repo.AddMember(context.Background(), h.HouseholdID, 43)

	members := repo.GetMembers(context.Background(), h.HouseholdID)
	if len(members) != 2 {
		t.Errorf("expected 2 members, got %d", len(members))
	}
}

func TestHousehold_MembersOfDifferentHouseholdsAreIsolated(t *testing.T) {
	repo := NewFakeHouseholdRepo()

	h1, _ := repo.InsertHousehold(context.Background(), 1, "H1")
	h2, _ := repo.InsertHousehold(context.Background(), 1, "H2")

	repo.AddMember(context.Background(), h1.HouseholdID, 1)
	repo.AddMember(context.Background(), h2.HouseholdID, 2)

	m1 := repo.GetMembers(context.Background(), h1.HouseholdID)
	m2 := repo.GetMembers(context.Background(), h2.HouseholdID)
	if len(m1) != 1 || m1[0] != 1 {
		t.Errorf("H1 members unexpected: %v", m1)
	}
	if len(m2) != 1 || m2[0] != 2 {
		t.Errorf("H2 members unexpected: %v", m2)
	}
}

// ── Invites ───────────────────────────────────────────────────────────────────

func TestHousehold_CreateInviteStoresEntry(t *testing.T) {
	repo := NewFakeHouseholdRepo()
	h, _ := repo.InsertHousehold(context.Background(), 1, "Test")

	inv, err := repo.CreateInvite(context.Background(), h.HouseholdID, "abc123", 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv.InviteCode != "abc123" {
		t.Errorf("expected InviteCode=abc123, got %q", inv.InviteCode)
	}
	if inv.Status != "pending" {
		t.Errorf("expected Status=pending, got %q", inv.Status)
	}
	if inv.RequestedByUserID != 7 {
		t.Errorf("expected RequestedByUserID=7, got %d", inv.RequestedByUserID)
	}
}

func TestHousehold_GetInviteByCode(t *testing.T) {
	repo := NewFakeHouseholdRepo()
	h, _ := repo.InsertHousehold(context.Background(), 1, "Test")

	_, _ = repo.CreateInvite(context.Background(), h.HouseholdID, "secretcode", 3)
	found, err := repo.GetInviteByCode(context.Background(), "secretcode")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found.InviteCode != "secretcode" {
		t.Errorf("expected code=secretcode, got %q", found.InviteCode)
	}
}

func TestHousehold_GetInviteByCode_NotFound(t *testing.T) {
	repo := NewFakeHouseholdRepo()
	_, err := repo.GetInviteByCode(context.Background(), "doesnotexist")
	if err == nil {
		t.Fatal("expected error for unknown invite code")
	}
}

func TestHousehold_GetInviteByID(t *testing.T) {
	repo := NewFakeHouseholdRepo()
	h, _ := repo.InsertHousehold(context.Background(), 1, "Test")

	inv, _ := repo.CreateInvite(context.Background(), h.HouseholdID, "code1", 5)
	fetched, err := repo.GetInviteByID(context.Background(), inv.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fetched.ID != inv.ID {
		t.Errorf("expected ID=%d, got %d", inv.ID, fetched.ID)
	}
}

func TestHousehold_RespondToInvite_Approve(t *testing.T) {
	repo := NewFakeHouseholdRepo()
	h, _ := repo.InsertHousehold(context.Background(), 1, "Test")

	inv, _ := repo.CreateInvite(context.Background(), h.HouseholdID, "approve_me", 10)
	updated, err := repo.RespondToInvite(context.Background(), inv.ID, "approved")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Status != "approved" {
		t.Errorf("expected Status=approved, got %q", updated.Status)
	}
}

func TestHousehold_RespondToInvite_Deny(t *testing.T) {
	repo := NewFakeHouseholdRepo()
	h, _ := repo.InsertHousehold(context.Background(), 1, "Test")

	inv, _ := repo.CreateInvite(context.Background(), h.HouseholdID, "deny_me", 11)
	updated, err := repo.RespondToInvite(context.Background(), inv.ID, "denied")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Status != "denied" {
		t.Errorf("expected Status=denied, got %q", updated.Status)
	}
}

func TestHousehold_RespondToInvite_NotFound(t *testing.T) {
	repo := NewFakeHouseholdRepo()
	_, err := repo.RespondToInvite(context.Background(), 9999, "approved")
	if err == nil {
		t.Fatal("expected error for unknown invite ID")
	}
}

func TestHousehold_InviteCount(t *testing.T) {
	repo := NewFakeHouseholdRepo()
	h, _ := repo.InsertHousehold(context.Background(), 1, "Test")

	if repo.InviteCount() != 0 {
		t.Fatalf("expected 0 initial invites, got %d", repo.InviteCount())
	}
	_, _ = repo.CreateInvite(context.Background(), h.HouseholdID, "c1", 1)
	_, _ = repo.CreateInvite(context.Background(), h.HouseholdID, "c2", 2)
	if repo.InviteCount() != 2 {
		t.Errorf("expected 2 invites, got %d", repo.InviteCount())
	}
}
