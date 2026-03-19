package authentication_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"weekly-shopping-app/authentication"
)

// ── HashPassword / CheckPassword ─────────────────────────────────────────────

func TestHashPassword_ProducesNonEmptyHash(t *testing.T) {
	hash, err := authentication.HashPassword("mysecret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
}

func TestHashPassword_DifferentPasswordsProduceDifferentHashes(t *testing.T) {
	h1, _ := authentication.HashPassword("password1")
	h2, _ := authentication.HashPassword("password2")
	if h1 == h2 {
		t.Fatal("different passwords should not produce the same hash")
	}
}

func TestHashPassword_SamePasswordProducesDifferentHashes(t *testing.T) {
	// bcrypt uses a random salt — two calls on the same input must differ
	h1, _ := authentication.HashPassword("same")
	h2, _ := authentication.HashPassword("same")
	if h1 == h2 {
		t.Fatal("bcrypt should produce different hashes for the same password due to salting")
	}
}

func TestCheckPassword_CorrectPassword(t *testing.T) {
	hash, _ := authentication.HashPassword("correct")
	if !authentication.CheckPassword(hash, "correct") {
		t.Fatal("CheckPassword should return true for the correct password")
	}
}

func TestCheckPassword_WrongPassword(t *testing.T) {
	hash, _ := authentication.HashPassword("correct")
	if authentication.CheckPassword(hash, "wrong") {
		t.Fatal("CheckPassword should return false for a wrong password")
	}
}

func TestCheckPassword_EmptyPassword(t *testing.T) {
	hash, _ := authentication.HashPassword("notempty")
	if authentication.CheckPassword(hash, "") {
		t.Fatal("CheckPassword should return false for an empty password")
	}
}

// ── Session lifecycle ─────────────────────────────────────────────────────────

func TestCreateSession_SetsCookie(t *testing.T) {
	w := httptest.NewRecorder()
	authentication.CreateSession(w, "alice", 42)
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected a session cookie to be set")
	}
	if cookies[0].Name != "session_id" {
		t.Fatalf("expected cookie named session_id, got %q", cookies[0].Name)
	}
}

func TestGetUser_ValidSession(t *testing.T) {
	w := httptest.NewRecorder()
	authentication.CreateSession(w, "alice", 42)
	cookie := w.Result().Cookies()[0]

	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(cookie)

	username, ok := authentication.GetUser(r)
	if !ok {
		t.Fatal("expected GetUser to return true for a valid session")
	}
	if username != "alice" {
		t.Fatalf("expected username %q, got %q", "alice", username)
	}
}

func TestGetUser_NoCookie(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	_, ok := authentication.GetUser(r)
	if ok {
		t.Fatal("expected GetUser to return false when no cookie is present")
	}
}

func TestGetUser_InvalidCookieValue(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{Name: "session_id", Value: "not-a-real-session"})
	_, ok := authentication.GetUser(r)
	if ok {
		t.Fatal("expected GetUser to return false for an unknown session ID")
	}
}

func TestGetUserID_ValidSession(t *testing.T) {
	w := httptest.NewRecorder()
	authentication.CreateSession(w, "bob", 99)
	cookie := w.Result().Cookies()[0]

	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(cookie)

	id, err := authentication.GetUserID(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 99 {
		t.Fatalf("expected user ID 99, got %d", id)
	}
}

func TestGetUserID_NoSession(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	_, err := authentication.GetUserID(r)
	if err == nil {
		t.Fatal("expected error when no session cookie is present")
	}
}

func TestDestroySession_RemovesSession(t *testing.T) {
	w := httptest.NewRecorder()
	authentication.CreateSession(w, "carol", 7)
	cookie := w.Result().Cookies()[0]

	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(cookie)

	if _, ok := authentication.GetUser(r); !ok {
		t.Fatal("session should be valid before destroy")
	}

	w2 := httptest.NewRecorder()
	authentication.DestroySession(w2, r)

	if _, ok := authentication.GetUser(r); ok {
		t.Fatal("session should be invalid after destroy")
	}
}

func TestCreateSession_MultipleUsers_Independent(t *testing.T) {
	w1, w2 := httptest.NewRecorder(), httptest.NewRecorder()
	authentication.CreateSession(w1, "user1", 1)
	authentication.CreateSession(w2, "user2", 2)

	r1 := httptest.NewRequest("GET", "/", nil)
	r1.AddCookie(w1.Result().Cookies()[0])
	r2 := httptest.NewRequest("GET", "/", nil)
	r2.AddCookie(w2.Result().Cookies()[0])

	u1, ok1 := authentication.GetUser(r1)
	u2, ok2 := authentication.GetUser(r2)

	if !ok1 || !ok2 {
		t.Fatal("both sessions should be valid")
	}
	if u1 != "user1" || u2 != "user2" {
		t.Fatalf("expected user1/user2, got %q/%q", u1, u2)
	}
}
