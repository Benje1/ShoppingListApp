package integration_test

import (
	"context"
	"testing"

	"weekly-shopping-app/authentication"
	"weekly-shopping-app/database"
)

func newHouseholdRepo() *database.PostgresHouseholdRepo {
	return &database.PostgresHouseholdRepo{DB: sharedPool()}
}

// ── InsertHousehold ───────────────────────────────────────────────────────────

func TestIntegration_InsertHousehold_Succeeds(t *testing.T) {
	ownerID, _, _ := makeUser(t)
	hid := makeHousehold(t, ownerID)

	h, err := newHouseholdRepo().GetHousehold(context.Background(), hid)
	if err != nil {
		t.Fatalf("GetHousehold: %v", err)
	}
	if h.HouseholdID != hid {
		t.Errorf("expected HouseholdID=%d, got %d", hid, h.HouseholdID)
	}
	if h.NumPeople != 2 {
		t.Errorf("expected NumPeople=2, got %d", h.NumPeople)
	}
}

// ── GetHousehold ──────────────────────────────────────────────────────────────

func TestIntegration_GetHousehold_NotFound_ReturnsError(t *testing.T) {
	_, err := newHouseholdRepo().GetHousehold(context.Background(), -1)
	if err == nil {
		t.Fatal("expected an error for unknown household ID, got nil")
	}
}

// ── RenameHousehold ───────────────────────────────────────────────────────────

func TestIntegration_RenameHousehold_Succeeds(t *testing.T) {
	repo := newHouseholdRepo()
	ctx := context.Background()
	ownerID, _, _ := makeUser(t)
	hid := makeHousehold(t, ownerID)

	renamed, err := repo.RenameHousehold(ctx, hid, "After Rename")
	if err != nil {
		t.Fatalf("RenameHousehold: %v", err)
	}
	if !renamed.Name.Valid || renamed.Name.String != "After Rename" {
		t.Errorf("expected Name=After Rename, got %+v", renamed.Name)
	}

	// Confirm it persisted.
	fetched, err := repo.GetHousehold(ctx, hid)
	if err != nil {
		t.Fatalf("GetHousehold after rename: %v", err)
	}
	if fetched.Name.String != "After Rename" {
		t.Errorf("rename did not persist: got %q", fetched.Name.String)
	}
}

// ── DeleteHousehold ───────────────────────────────────────────────────────────

func TestIntegration_DeleteHousehold_Succeeds(t *testing.T) {
	repo := newHouseholdRepo()
	ctx := context.Background()
	ownerID, _, _ := makeUser(t)
	hid := makeHousehold(t, ownerID)

	if err := repo.DeleteHousehold(ctx, hid); err != nil {
		t.Fatalf("DeleteHousehold: %v", err)
	}
	if _, err := repo.GetHousehold(ctx, hid); err == nil {
		t.Fatal("expected an error fetching a deleted household, got nil")
	}
}

// ── GetHouseholdMembers ───────────────────────────────────────────────────────

func TestIntegration_GetHouseholdMembers_ReturnsOwner(t *testing.T) {
	ownerID, _, _ := makeUser(t)
	hid := makeHousehold(t, ownerID)

	members, err := newHouseholdRepo().GetHouseholdMembers(context.Background(), hid)
	if err != nil {
		t.Fatalf("GetHouseholdMembers: %v", err)
	}
	if len(members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(members))
	}
	if members[0].ID != ownerID {
		t.Errorf("expected member ID=%d, got %d", ownerID, members[0].ID)
	}
}

// ── Invite flow ───────────────────────────────────────────────────────────────

func TestIntegration_InviteFlow_ApproveJoin(t *testing.T) {
	hrepo := newHouseholdRepo()
	urepo := newUserRepo()
	ctx := context.Background()

	ownerID, _, _ := makeUser(t)
	joinerID, _, _ := makeUser(t)
	hid := makeHousehold(t, ownerID)

	// Owner generates a shareable code.
	invite, err := hrepo.CreateInvite(ctx, hid, uniqueUsername("code"), ownerID)
	if err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}
	if invite.Status != "pending" {
		t.Errorf("expected status=pending, got %q", invite.Status)
	}

	// Joiner requests to join via the code.
	found, err := hrepo.GetInviteByCode(ctx, invite.InviteCode)
	if err != nil {
		t.Fatalf("GetInviteByCode: %v", err)
	}
	joinReq, err := hrepo.CreateInvite(ctx, found.HouseholdID, uniqueUsername("joincode"), joinerID)
	if err != nil {
		t.Fatalf("CreateInvite (join request): %v", err)
	}

	// Owner approves.
	if err := urepo.AddUserToHousehold(ctx, joinReq.RequestedByUserID, joinReq.HouseholdID); err != nil {
		t.Fatalf("AddUserToHousehold: %v", err)
	}
	approved, err := hrepo.RespondToInvite(ctx, joinReq.ID, "approved")
	if err != nil {
		t.Fatalf("RespondToInvite: %v", err)
	}
	if approved.Status != "approved" {
		t.Errorf("expected status=approved, got %q", approved.Status)
	}

	members, err := hrepo.GetHouseholdMembers(ctx, hid)
	if err != nil {
		t.Fatalf("GetHouseholdMembers: %v", err)
	}
	found_joiner := false
	for _, m := range members {
		if m.ID == joinerID {
			found_joiner = true
			break
		}
	}
	if !found_joiner {
		t.Error("joiner was not found in household members after approval")
	}
}

func TestIntegration_InviteFlow_DenyJoin(t *testing.T) {
	hrepo := newHouseholdRepo()
	ctx := context.Background()

	ownerID, _, _ := makeUser(t)
	joinerID, _, _ := makeUser(t)
	hid := makeHousehold(t, ownerID)

	joinReq, err := hrepo.CreateInvite(ctx, hid, uniqueUsername("denycode"), joinerID)
	if err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}

	denied, err := hrepo.RespondToInvite(ctx, joinReq.ID, "denied")
	if err != nil {
		t.Fatalf("RespondToInvite: %v", err)
	}
	if denied.Status != "denied" {
		t.Errorf("expected status=denied, got %q", denied.Status)
	}

	members, err := hrepo.GetHouseholdMembers(ctx, hid)
	if err != nil {
		t.Fatalf("GetHouseholdMembers: %v", err)
	}
	for _, m := range members {
		if m.ID == joinerID {
			t.Error("denied joiner should not appear in household members")
		}
	}
}

func TestIntegration_InviteFlow_NonExistentCode_Fails(t *testing.T) {
	_, err := newHouseholdRepo().GetInviteByCode(context.Background(), "no-such-code-xyz")
	if err == nil {
		t.Fatal("expected an error for unknown invite code, got nil")
	}
}

func TestIntegration_GetPendingInvites_ReturnsOnlyPending(t *testing.T) {
	hrepo := newHouseholdRepo()
	ctx := context.Background()

	ownerID, _, _ := makeUser(t)
	u1ID, _, _ := makeUser(t)
	u2ID, _, _ := makeUser(t)
	hid := makeHousehold(t, ownerID)

	inv1, err := hrepo.CreateInvite(ctx, hid, uniqueUsername("p1"), u1ID)
	if err != nil {
		t.Fatalf("CreateInvite u1: %v", err)
	}
	if _, err := hrepo.CreateInvite(ctx, hid, uniqueUsername("p2"), u2ID); err != nil {
		t.Fatalf("CreateInvite u2: %v", err)
	}

	// Approve the first — it must disappear from pending.
	if _, err := hrepo.RespondToInvite(ctx, inv1.ID, "approved"); err != nil {
		t.Fatalf("RespondToInvite: %v", err)
	}

	pending, err := hrepo.GetPendingInvites(ctx, hid)
	if err != nil {
		t.Fatalf("GetPendingInvites: %v", err)
	}
	if len(pending) != 1 {
		t.Errorf("expected 1 pending invite, got %d", len(pending))
	}
	if pending[0].RequestedByUserID != u2ID {
		t.Errorf("expected pending invite from u2 (%d), got from %d", u2ID, pending[0].RequestedByUserID)
	}
}

func TestIntegration_RespondToInvite_AlreadyActioned_Fails(t *testing.T) {
	hrepo := newHouseholdRepo()
	ctx := context.Background()

	ownerID, _, _ := makeUser(t)
	joinerID, _, _ := makeUser(t)
	hid := makeHousehold(t, ownerID)

	inv, err := hrepo.CreateInvite(ctx, hid, uniqueUsername("actionedcode"), joinerID)
	if err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}

	// Approve once.
	if _, err := hrepo.RespondToInvite(ctx, inv.ID, "approved"); err != nil {
		t.Fatalf("first RespondToInvite: %v", err)
	}

	// Trying to action an already-actioned invite should fail.
	if _, err := hrepo.RespondToInvite(ctx, inv.ID, "denied"); err == nil {
		t.Fatal("expected an error responding to an already-actioned invite, got nil")
	}
}

// ── AddUserToHousehold ────────────────────────────────────────────────────────

func TestIntegration_AddUserToHousehold_SecondMember_Succeeds(t *testing.T) {
	urepo := newUserRepo()
	hrepo := newHouseholdRepo()
	ctx := context.Background()

	ownerID, _, _ := makeUser(t)
	secondID, _, _ := makeUser(t)
	hid := makeHousehold(t, ownerID)

	if err := urepo.AddUserToHousehold(ctx, secondID, hid); err != nil {
		t.Fatalf("AddUserToHousehold: %v", err)
	}

	members, err := hrepo.GetHouseholdMembers(ctx, hid)
	if err != nil {
		t.Fatalf("GetHouseholdMembers: %v", err)
	}
	if len(members) != 2 {
		t.Errorf("expected 2 members, got %d", len(members))
	}
}

// This test uses a named variable for the authentication import used in
// the invite flow tests. Keep it to avoid "imported and not used" errors.
var _ = authentication.HashPassword
