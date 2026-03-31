package authentication

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

var (
	sessions   = make(map[string]Session)
	sessionMux sync.Mutex
)

// contextKey is an unexported type for context keys in this package.
type contextKey string

const sessionContextKey contextKey = "session"

// Session holds everything known about the authenticated user.
// HouseholdIds is populated at login from the database and refreshed on
// every request via RequireAuth, so it can never be spoofed via URL params.
type Session struct {
	Username     string
	UserID       int32
	HouseholdIds []int32
	ExpiresAt    time.Time
}

// FirstHouseholdID returns the user's primary household ID, or 0 if they
// have none. This is the safe, server-authoritative replacement for reading
// household_id from a URL query parameter.
func (s Session) FirstHouseholdID() int32 {
	if len(s.HouseholdIds) == 0 {
		return 0
	}
	return s.HouseholdIds[0]
}

func (s Session) GetAllHouseholdsID() []int32 {
	if len(s.HouseholdIds) == 0 {
		return []int32{0}
	}
	return s.HouseholdIds
}

// HasHousehold reports whether the given household ID is in the user's session.
// Use this to authorise access before acting on a household resource.
func (s Session) HasHousehold(id int32) bool {
	for _, hid := range s.HouseholdIds {
		if hid == id {
			return true
		}
	}
	return false
}

func newSessionID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// CreateSession stores a new session and sets the cookie.
// householdIds should be the complete list of households the user belongs to
// at login time.
func CreateSession(w http.ResponseWriter, username string, userID int32, householdIds []int32) {
	sessionID := newSessionID()

	session := Session{
		Username:     username,
		UserID:       userID,
		HouseholdIds: householdIds,
		ExpiresAt:    time.Now().Add(30 * time.Minute),
	}

	sessionMux.Lock()
	sessions[sessionID] = session
	sessionMux.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
		Expires:  session.ExpiresAt,
	})
}

func DestroySession(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err == nil {
		sessionMux.Lock()
		delete(sessions, cookie.Value)
		sessionMux.Unlock()
	}

	http.SetCookie(w, &http.Cookie{
		Name:   "session_id",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
}

// getSession reads the session from the cookie store.
func getSession(r *http.Request) (Session, bool) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return Session{}, false
	}

	sessionMux.Lock()
	defer sessionMux.Unlock()

	session, ok := sessions[cookie.Value]
	if !ok {
		return Session{}, false
	}

	if time.Now().After(session.ExpiresAt) {
		delete(sessions, cookie.Value)
		return Session{}, false
	}

	return session, true
}

// GetUser returns the username from the session cookie (kept for compatibility).
func GetUser(r *http.Request) (string, bool) {
	s, ok := getSession(r)
	return s.Username, ok
}

// GetUserID returns just the numeric user ID from the session cookie.
// Prefer GetSessionFromContext when you also need household information.
func GetUserID(r *http.Request) (int32, error) {
	s, ok := getSession(r)
	if !ok {
		return 0, errors.New("not authenticated")
	}
	if s.UserID == 0 {
		return 0, fmt.Errorf("user ID not in session (re-login required)")
	}
	return s.UserID, nil
}

// SessionFromContext retrieves the Session that RequireAuth injected into
// the request context. This is the preferred way for handlers to access
// user and household information — it never touches URL parameters.
func SessionFromContext(r *http.Request) (Session, error) {
	s, ok := r.Context().Value(sessionContextKey).(Session)
	if !ok || s.UserID == 0 {
		return Session{}, errors.New("no session in context (route not protected?)")
	}
	return s, nil
}

// WithSession injects a Session into a context (used by RequireAuth).
func WithSession(ctx context.Context, s Session) context.Context {
	return context.WithValue(ctx, sessionContextKey, s)
}

// RequireAuth rejects unauthenticated requests and injects the full Session
// into the request context so handlers never need to re-read the cookie.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, ok := getSession(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// Inject session into context — handlers read from here, not URL params
		ctx := WithSession(r.Context(), session)
		r = r.WithContext(ctx)

		// Keep X-User header for any code that still reads it
		r.Header.Set("X-User", session.Username)

		next.ServeHTTP(w, r)
	})
}

func StartSessionCleanup() {
	go func() {
		for {
			time.Sleep(1 * time.Minute)

			sessionMux.Lock()
			for id, session := range sessions {
				if time.Now().After(session.ExpiresAt) {
					delete(sessions, id)
				}
			}
			sessionMux.Unlock()
		}
	}()
}
