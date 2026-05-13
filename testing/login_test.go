package authntication_test

import (
	"net/http/httptest"
	"testing"

	"weekly-shopping-app/authentication"
)

func TestSessionTokenExpiration(t *testing.T) {
	w := httptest.NewRecorder()
	// CreateSession now requires a userID — pass 1 as a dummy value for tests
	sessionID := authentication.CreateSession(w, "test", 1, nil)
	res := w.Result()
	cookies := res.Cookies()

	if len(cookies) == 0 {
		t.Fatal("no session cookie was set")
	}

	sessionCookie := cookies[0]

	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(sessionCookie)

	user, ok := authentication.GetUser(r)
	if !ok {
		t.Fatal("error getting user by session")
	}

	if user != "test" {
		t.Fatal(user)
	}

	// Force-expire the session using the test helper (avoids real time.Sleep).
	authentication.ExpireSessionForTesting(sessionID)

	user, ok = authentication.GetUser(r)
	if ok {
		t.Fatal("session has not expired")
	}

	if user != "" {
		t.Fatalf("expected empty username after expiry, got %q", user)
	}
}
