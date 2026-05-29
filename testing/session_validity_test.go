package authntication_test

// session_validity_test.go
// Tests for expired and invalid session handling:
//   - Expired sessions are rejected
//   - Missing/empty tokens are rejected
//   - A valid session is accepted and returns the correct user
//   - Expiry helper works correctly in test context

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"weekly-shopping-app/authentication"
)

// createTestSession creates a session for the given user and returns both the
// session cookie and the raw session ID (used by expiry helpers).
func createTestSession(t *testing.T, username string, userID int32, householdIDs []int32) (*http.Cookie, string) {
	t.Helper()
	w := httptest.NewRecorder()
	sessionID := authentication.CreateSession(w, username, userID, householdIDs)
	if sessionID == "" {
		t.Fatal("CreateSession returned empty session ID")
	}
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("no cookie set by CreateSession")
	}
	return cookies[0], sessionID
}

// ── Valid session ─────────────────────────────────────────────────────────────

func TestSession_ValidSession_ReturnsUser(t *testing.T) {
	cookie, _ := createTestSession(t, "alice", 1, nil)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(cookie)

	user, ok := authentication.GetUser(r)
	if !ok {
		t.Fatal("expected valid session to return ok=true")
	}
	if user != "alice" {
		t.Errorf("expected username=alice, got %q", user)
	}
}

func TestSession_ValidSession_BearerToken(t *testing.T) {
	_, sessionID := createTestSession(t, "bob", 2, nil)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer "+sessionID)

	user, ok := authentication.GetUser(r)
	if !ok {
		t.Fatal("expected Bearer token session to return ok=true")
	}
	if user != "bob" {
		t.Errorf("expected username=bob, got %q", user)
	}
}

// ── Expired session ───────────────────────────────────────────────────────────

func TestSession_ExpiredSession_IsRejected(t *testing.T) {
	cookie, sessionID := createTestSession(t, "carol", 3, nil)

	authentication.ExpireSessionForTesting(sessionID)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(cookie)

	_, ok := authentication.GetUser(r)
	if ok {
		t.Fatal("expected expired session to be rejected")
	}
}

func TestSession_ExpiredSession_ReturnsEmptyUsername(t *testing.T) {
	cookie, sessionID := createTestSession(t, "dave", 4, nil)
	authentication.ExpireSessionForTesting(sessionID)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(cookie)

	user, _ := authentication.GetUser(r)
	if user != "" {
		t.Errorf("expected empty username after expiry, got %q", user)
	}
}

func TestSession_ExpiredSession_CannotBeReused(t *testing.T) {
	cookie, sessionID := createTestSession(t, "eve", 5, nil)
	authentication.ExpireSessionForTesting(sessionID)

	// Multiple requests all fail — the session is not "un-expired" on retry.
	for i := range 3 {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.AddCookie(cookie)
		if _, ok := authentication.GetUser(r); ok {
			t.Errorf("request %d: expected expired session to be rejected on every attempt", i+1)
		}
	}
}

// ── Invalid / missing token ───────────────────────────────────────────────────

func TestSession_NoToken_IsRejected(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	// No cookie, no Authorization header.
	_, ok := authentication.GetUser(r)
	if ok {
		t.Fatal("expected request with no token to be rejected")
	}
}

func TestSession_EmptyCookieValue_IsRejected(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: "session_id", Value: ""})

	_, ok := authentication.GetUser(r)
	if ok {
		t.Fatal("expected empty session_id cookie to be rejected")
	}
}

func TestSession_RandomToken_IsRejected(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer totallyfaketoken")

	_, ok := authentication.GetUser(r)
	if ok {
		t.Fatal("expected unknown Bearer token to be rejected")
	}
}

func TestSession_MalformedBearerHeader_IsRejected(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "NotBearer sometoken")

	_, ok := authentication.GetUser(r)
	if ok {
		t.Fatal("expected malformed Authorization header to be rejected")
	}
}

// ── Session isolation ─────────────────────────────────────────────────────────

func TestSession_TwoUsers_SessionsAreIsolated(t *testing.T) {
	cookieA, _ := createTestSession(t, "frank", 6, nil)
	cookieB, _ := createTestSession(t, "grace", 7, nil)

	rA := httptest.NewRequest(http.MethodGet, "/", nil)
	rA.AddCookie(cookieA)
	userA, okA := authentication.GetUser(rA)

	rB := httptest.NewRequest(http.MethodGet, "/", nil)
	rB.AddCookie(cookieB)
	userB, okB := authentication.GetUser(rB)

	if !okA || userA != "frank" {
		t.Errorf("expected frank's session to return frank, got ok=%v user=%q", okA, userA)
	}
	if !okB || userB != "grace" {
		t.Errorf("expected grace's session to return grace, got ok=%v user=%q", okB, userB)
	}
}

func TestSession_ExpiringOneSession_DoesNotAffectAnother(t *testing.T) {
	cookieA, sessionIDA := createTestSession(t, "harry", 8, nil)
	cookieB, _ := createTestSession(t, "iris", 9, nil)

	authentication.ExpireSessionForTesting(sessionIDA)

	// Harry's session is expired.
	rA := httptest.NewRequest(http.MethodGet, "/", nil)
	rA.AddCookie(cookieA)
	if _, ok := authentication.GetUser(rA); ok {
		t.Error("expected harry's session to be expired")
	}

	// Iris's session is unaffected.
	rB := httptest.NewRequest(http.MethodGet, "/", nil)
	rB.AddCookie(cookieB)
	userB, ok := authentication.GetUser(rB)
	if !ok || userB != "iris" {
		t.Errorf("expected iris's session to still be valid, got ok=%v user=%q", ok, userB)
	}
}
