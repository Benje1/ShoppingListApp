package shoppinglist

import (
	"testing"

	sqlc "weekly-shopping-app/database/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
)

// ── scopeParams ───────────────────────────────────────────────────────────────

func TestScopeParams_HouseholdScope(t *testing.T) {
	hid, uid := scopeParams(10, 5, "household")
	if !hid.Valid || hid.Int32 != 5 {
		t.Fatalf("expected household_id=5, got %+v", hid)
	}
	if uid.Valid {
		t.Fatal("expected user_id to be NULL for household scope")
	}
}

func TestScopeParams_PersonalScope(t *testing.T) {
	hid, uid := scopeParams(10, 5, "personal")
	if hid.Valid {
		t.Fatal("expected household_id to be NULL for personal scope")
	}
	if !uid.Valid || uid.Int32 != 10 {
		t.Fatalf("expected user_id=10, got %+v", uid)
	}
}

func TestScopeParams_UnknownScopeTreatedAsPersonal(t *testing.T) {
	hid, uid := scopeParams(10, 5, "unknown")
	if hid.Valid {
		t.Fatal("unknown scope should not set household_id")
	}
	if !uid.Valid || uid.Int32 != 10 {
		t.Fatalf("unknown scope should fall back to user_id=10, got %+v", uid)
	}
}

func TestScopeParams_HouseholdScope_ZeroHouseholdID(t *testing.T) {
	// household_id=0 with "household" scope → no real household, should use user_id
	hid, uid := scopeParams(10, 0, "household")
	if hid.Valid {
		t.Fatal("expected household_id to be NULL when householdID=0")
	}
	if !uid.Valid || uid.Int32 != 10 {
		t.Fatalf("expected fallback to user_id=10, got %+v", uid)
	}
}

// ── listParams ────────────────────────────────────────────────────────────────

func TestListParams_WithHousehold(t *testing.T) {
	params := listParams(10, 5)
	if !params.UserID.Valid || params.UserID.Int32 != 10 {
		t.Errorf("expected UserID=10, got %+v", params.UserID)
	}
	if !params.HouseholdID.Valid || params.HouseholdID.Int32 != 5 {
		t.Errorf("expected HouseholdID=5, got %+v", params.HouseholdID)
	}
}

func TestListParams_WithoutHousehold(t *testing.T) {
	params := listParams(10, 0)
	if !params.UserID.Valid || params.UserID.Int32 != 10 {
		t.Errorf("expected UserID=10, got %+v", params.UserID)
	}
	if params.HouseholdID.Valid {
		t.Error("expected HouseholdID to be NULL when householdID=0")
	}
}

func TestListParams_IsCorrectType(t *testing.T) {
	params := listParams(1, 2)
	// Ensure returned type is correct for the sqlc query
	var _ sqlc.GetShoppingListParams = params
}

// ── getShoppingListUpdatedAt response formatting ──────────────────────────────

func TestTimestampFormatting_ValidTimestamp(t *testing.T) {
	// Test the timestamp formatting logic used in getShoppingListUpdatedAt
	// and getMealPlanUpdatedAt — isolated here as a pure function test.
	ts := pgtype.Timestamp{}
	_ = ts.Scan("2024-01-15 10:30:00")

	if !ts.Valid {
		t.Fatal("expected valid timestamp after scan")
	}
	formatted := ts.Time.UTC().Format("2006-01-02T15:04:05Z")
	if formatted == "" {
		t.Fatal("expected non-empty formatted timestamp")
	}
	// Should be ISO 8601
	if len(formatted) != 20 {
		t.Errorf("expected 20-char ISO timestamp, got %q (len=%d)", formatted, len(formatted))
	}
}

// ── HaveItInput scope handling ────────────────────────────────────────────────

func TestHaveItInput_ScopeParamsConsistency(t *testing.T) {
	// Ensure HaveItInput scope flows through scopeParams correctly
	cases := []struct {
		input   HaveItInput
		wantHID bool
		wantUID bool
	}{
		{HaveItInput{ItemID: 1, Scope: "household", HouseholdID: 5}, true, false},
		{HaveItInput{ItemID: 1, Scope: "personal", HouseholdID: 5}, false, true},
		{HaveItInput{ItemID: 1, Scope: "household", HouseholdID: 0}, false, true},
	}

	for _, c := range cases {
		hid, uid := scopeParams(99, c.input.HouseholdID, c.input.Scope)
		if hid.Valid != c.wantHID {
			t.Errorf("scope=%q hid=%d: expected hid.Valid=%v, got %v",
				c.input.Scope, c.input.HouseholdID, c.wantHID, hid.Valid)
		}
		if uid.Valid != c.wantUID {
			t.Errorf("scope=%q hid=%d: expected uid.Valid=%v, got %v",
				c.input.Scope, c.input.HouseholdID, c.wantUID, uid.Valid)
		}
	}
}
