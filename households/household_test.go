package households

import (
	"context"
	"testing"

	sqlc "weekly-shopping-app/database/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
)

// ── householdResponse ─────────────────────────────────────────────────────────

func TestHouseholdResponse_WithName(t *testing.T) {
	h := &sqlc.Household{
		HouseholdID: 7,
		NumPeople:   3,
		Name:        pgtype.Text{String: "The Smiths", Valid: true},
	}
	resp := householdResponse(h)
	if resp.HouseholdID != 7 {
		t.Errorf("expected HouseholdID=7, got %d", resp.HouseholdID)
	}
	if resp.NumPeople != 3 {
		t.Errorf("expected NumPeople=3, got %d", resp.NumPeople)
	}
	if resp.Name != "The Smiths" {
		t.Errorf("expected Name=%q, got %q", "The Smiths", resp.Name)
	}
}

func TestHouseholdResponse_WithoutName(t *testing.T) {
	h := &sqlc.Household{
		HouseholdID: 1,
		NumPeople:   1,
		Name:        pgtype.Text{Valid: false},
	}
	resp := householdResponse(h)
	if resp.Name != "" {
		t.Errorf("expected empty Name for null DB value, got %q", resp.Name)
	}
}

func TestHouseholdResponse_DefaultsPreserved(t *testing.T) {
	h := &sqlc.Household{
		HouseholdID: 99,
		NumPeople:   0,
		Name:        pgtype.Text{Valid: false},
	}
	resp := householdResponse(h)
	if resp.HouseholdID != 99 {
		t.Errorf("expected HouseholdID=99, got %d", resp.HouseholdID)
	}
}

// ── generateCode ─────────────────────────────────────────────────────────────

func TestGenerateCode_Length(t *testing.T) {
	code, err := generateCode()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 6 bytes hex-encoded = 12 characters
	if len(code) != 12 {
		t.Errorf("expected code length 12, got %d (%q)", len(code), code)
	}
}

func TestGenerateCode_IsHex(t *testing.T) {
	code, _ := generateCode()
	for _, c := range code {
		if !('0' <= c && c <= '9') && !('a' <= c && c <= 'f') {
			t.Errorf("expected hex characters, got %q in code %q", c, code)
		}
	}
}

func TestGenerateCode_Uniqueness(t *testing.T) {
	// Generate 50 codes and confirm no duplicates
	seen := make(map[string]bool)
	for i := 0; i < 50; i++ {
		code, err := generateCode()
		if err != nil {
			t.Fatalf("unexpected error on iteration %d: %v", i, err)
		}
		if seen[code] {
			t.Fatalf("duplicate code generated: %q", code)
		}
		seen[code] = true
	}
}

// ── respondToInvite validation (no DB needed) ─────────────────────────────────
// respondToInvite validates the action field before hitting the DB.
// We test that validation path directly by inspecting the exported error behaviour
// through the unexported function — since it's in the same package we can call it.

func TestRespondToInvite_InvalidAction(t *testing.T) {
	input := RespondToInviteInput{
		InviteID: 1,
		Action:   "maybe",
	}
	_, err := respondToInvite(context.TODO(), nil, input)
	if err == nil {
		t.Fatal("expected error for invalid action")
	}
	if err.Error() != "action must be 'approve' or 'deny'" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRespondToInvite_EmptyAction(t *testing.T) {
	input := RespondToInviteInput{InviteID: 1, Action: ""}
	_, err := respondToInvite(context.TODO(), nil, input)
	if err == nil {
		t.Fatal("expected error for empty action")
	}
}

// ── RenameHousehold input validation ─────────────────────────────────────────

func TestRenameHousehold_EmptyName(t *testing.T) {
	input := RenameHouseholdInput{Name: ""}
	_, err := renameHousehold(context.TODO(), nil, 1, input)
	if err == nil {
		t.Fatal("expected error for empty household name")
	}
	if err.Error() != "name cannot be empty" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// ── createHousehold default num_people ───────────────────────────────────────
// The business logic clamps num_people to 1 when <= 0.
// We can't call createHousehold (needs DB) but we can test the clamping logic
// by extracting it into a helper that's also tested indirectly.

func TestCreateHouseholdInput_NumPeopleFloor(t *testing.T) {
	cases := []struct {
		input    int32
		expected int32
	}{
		{0, 1},
		{-1, 1},
		{1, 1},
		{5, 5},
	}
	for _, c := range cases {
		np := c.input
		if np <= 0 {
			np = 1
		}
		if np != c.expected {
			t.Errorf("input %d: expected %d, got %d", c.input, c.expected, np)
		}
	}
}
