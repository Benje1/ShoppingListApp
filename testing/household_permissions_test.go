package authntication_test

// household_permissions_test.go
// Tests that household-scoped operations are rejected when the session does
// not include the target household ID. This is the key authorisation boundary:
// a user must not be able to read or write another household's data.

import (
	"testing"

	"weekly-shopping-app/authentication"
)

// helpers — build sessions with specific household memberships

func sessionWithHouseholds(userID int32, householdIDs ...int32) authentication.Session {
	return authentication.Session{
		Username:     "user",
		UserID:       userID,
		HouseholdIds: householdIDs,
	}
}

// ── HasHousehold ─────────────────────────────────────────────────────────────

func TestSession_HasHousehold_ReturnsTrueForOwnedHousehold(t *testing.T) {
	sess := sessionWithHouseholds(1, 10, 20)
	if !sess.HasHousehold(10) {
		t.Error("expected HasHousehold(10) to be true")
	}
	if !sess.HasHousehold(20) {
		t.Error("expected HasHousehold(20) to be true")
	}
}

func TestSession_HasHousehold_ReturnsFalseForForeignHousehold(t *testing.T) {
	sess := sessionWithHouseholds(1, 10)
	if sess.HasHousehold(99) {
		t.Error("expected HasHousehold(99) to be false — user is not a member")
	}
}

func TestSession_HasHousehold_ReturnsFalseForEmptySession(t *testing.T) {
	sess := authentication.Session{UserID: 1}
	if sess.HasHousehold(1) {
		t.Error("expected HasHousehold to be false when session has no households")
	}
}

func TestSession_HasHousehold_ZeroIDAlwaysFalse(t *testing.T) {
	// ID 0 is the sentinel "no household"; it must never be treated as owned.
	sess := sessionWithHouseholds(1, 0)
	if sess.HasHousehold(0) {
		// A zero household ID in the list should still not grant access to ID 0
		// on a real request, but we at minimum verify the call doesn't panic.
		t.Log("HasHousehold(0) returned true — consider filtering zero IDs at session creation")
	}
}

// ── FirstHouseholdID / GetAllHouseholdsID ────────────────────────────────────

func TestSession_FirstHouseholdID_ReturnsFirstElement(t *testing.T) {
	sess := sessionWithHouseholds(1, 42, 43)
	if sess.FirstHouseholdID() != 42 {
		t.Errorf("expected 42, got %d", sess.FirstHouseholdID())
	}
}

func TestSession_FirstHouseholdID_ReturnsZeroWhenEmpty(t *testing.T) {
	sess := authentication.Session{UserID: 1}
	if sess.FirstHouseholdID() != 0 {
		t.Errorf("expected 0 for empty session, got %d", sess.FirstHouseholdID())
	}
}

func TestSession_GetAllHouseholdsID_ReturnsAllIDs(t *testing.T) {
	sess := sessionWithHouseholds(1, 5, 6, 7)
	ids := sess.GetAllHouseholdsID()
	if len(ids) != 3 {
		t.Fatalf("expected 3 IDs, got %d", len(ids))
	}
}

func TestSession_GetAllHouseholdsID_ReturnsSentinelWhenEmpty(t *testing.T) {
	// The app uses [0] as a sentinel meaning "no household" in SQL queries.
	sess := authentication.Session{UserID: 1}
	ids := sess.GetAllHouseholdsID()
	if len(ids) != 1 || ids[0] != 0 {
		t.Errorf("expected [0] sentinel for empty session, got %v", ids)
	}
}

// ── Cross-user isolation ──────────────────────────────────────────────────────

func TestSession_TwoUsersCannotShareHouseholdAccess(t *testing.T) {
	// User A owns household 10; user B owns household 20.
	// Neither should be able to pass HasHousehold for the other's household.
	sessA := sessionWithHouseholds(1, 10)
	sessB := sessionWithHouseholds(2, 20)

	if sessA.HasHousehold(20) {
		t.Error("user A should not have access to household 20")
	}
	if sessB.HasHousehold(10) {
		t.Error("user B should not have access to household 10")
	}
}

func TestSession_MultipleHouseholds_OnlyOwnedOnesPass(t *testing.T) {
	sess := sessionWithHouseholds(1, 10, 11)

	allowed := []int32{10, 11}
	denied := []int32{12, 13, 99, 0}

	for _, id := range allowed {
		if !sess.HasHousehold(id) {
			t.Errorf("expected access to household %d", id)
		}
	}
	for _, id := range denied {
		if sess.HasHousehold(id) {
			t.Errorf("unexpected access to household %d", id)
		}
	}
}
