package authentication

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// contextKey is an unexported type for context keys in this package.
type contextKey string

const sessionContextKey contextKey = "session"

// pool is the DB connection used for session persistence.
// Initialised by InitSessionStore before any HTTP traffic is served.
var pool *pgxpool.Pool

// inMemorySessions is used as a fallback when pool is nil (e.g. in unit tests).
var inMemorySessions = struct {
	sync.RWMutex
	data map[string]inMemorySession
}{data: make(map[string]inMemorySession)}

type inMemorySession struct {
	session   Session
	expiresAt time.Time
}

// InitSessionStore wires up the database pool used by all session functions.
// Call this once from main, before starting the HTTP server.
func InitSessionStore(p *pgxpool.Pool) {
	pool = p
}

// Session holds everything known about the authenticated user.
type Session struct {
	Username     string
	UserID       int32
	HouseholdIds []int32
	ExpiresAt    time.Time
}

// FirstHouseholdID returns the user's primary household ID, or 0 if they have none.
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

// CreateSession persists a new session (to Postgres when configured, or to an
// in-memory store when pool is nil — used by unit tests), sets the cookie, and
// returns the raw session ID so the caller can include it in the response body.
func CreateSession(w http.ResponseWriter, username string, userID int32, householdIds []int32) string {
	sessionID := newSessionID()
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	if pool == nil {
		// In-memory fallback (used by unit tests that don't have a DB).
		inMemorySessions.Lock()
		inMemorySessions.data[sessionID] = inMemorySession{
			session: Session{
				Username:     username,
				UserID:       userID,
				HouseholdIds: householdIds,
				ExpiresAt:    expiresAt,
			},
			expiresAt: expiresAt,
		}
		inMemorySessions.Unlock()
	} else {
		hids := make([]int64, len(householdIds))
		for i, h := range householdIds {
			hids[i] = int64(h)
		}

		_, err := pool.Exec(context.Background(),
			`INSERT INTO sessions (id, user_id, username, household_ids, expires_at)
		 VALUES ($1, $2, $3, $4, $5)`,
			sessionID, userID, username, hids, expiresAt,
		)
		if err != nil {
			fmt.Printf("session insert error: %v\n", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return ""
		}
	}

	// Also set the cookie — still works for same-origin / curl / Postman usage.
	secure := os.Getenv("ENVIRONMENT") == "production"
	sameSite := http.SameSiteLaxMode
	if secure {
		sameSite = http.SameSiteNoneMode
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		Expires:  expiresAt,
	})

	return sessionID
}

// DestroySession deletes the session from the store and clears the cookie.
func DestroySession(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token != "" {
		if pool == nil {
			inMemorySessions.Lock()
			delete(inMemorySessions.data, token)
			inMemorySessions.Unlock()
		} else {
			pool.Exec(context.Background(), `DELETE FROM sessions WHERE id = $1`, token)
		}
	}
	http.SetCookie(w, &http.Cookie{
		Name:   "session_id",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
}

// extractToken retrieves the session token from either:
//  1. Authorization: Bearer <token> header  (used by cross-origin frontends)
//  2. session_id cookie                     (used by same-origin / server-rendered)
func extractToken(r *http.Request) string {
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	if cookie, err := r.Cookie("session_id"); err == nil {
		return cookie.Value
	}
	return ""
}

// getSession loads and validates a session from the configured store.
func getSession(r *http.Request) (Session, bool) {
	token := extractToken(r)
	if token == "" {
		return Session{}, false
	}

	if pool == nil {
		// In-memory fallback for unit tests.
		inMemorySessions.RLock()
		entry, ok := inMemorySessions.data[token]
		inMemorySessions.RUnlock()
		if !ok {
			return Session{}, false
		}
		if time.Now().After(entry.expiresAt) {
			inMemorySessions.Lock()
			delete(inMemorySessions.data, token)
			inMemorySessions.Unlock()
			return Session{}, false
		}
		return entry.session, true
	}

	var (
		username  string
		userID    int32
		hids      []int64
		expiresAt time.Time
	)

	err := pool.QueryRow(context.Background(),
		`SELECT username, user_id, household_ids, expires_at
		 FROM sessions WHERE id = $1`,
		token,
	).Scan(&username, &userID, &hids, &expiresAt)
	if err != nil {
		return Session{}, false
	}

	if time.Now().After(expiresAt) {
		pool.Exec(context.Background(), `DELETE FROM sessions WHERE id = $1`, token)
		return Session{}, false
	}

	householdIds := make([]int32, len(hids))
	for i, h := range hids {
		householdIds[i] = int32(h)
	}

	return Session{
		Username:     username,
		UserID:       userID,
		HouseholdIds: householdIds,
		ExpiresAt:    expiresAt,
	}, true
}

// GetUser returns the username from the session (kept for compatibility).
func GetUser(r *http.Request) (string, bool) {
	s, ok := getSession(r)
	return s.Username, ok
}

// GetUserID returns just the numeric user ID from the session.
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

// SessionFromContext retrieves the Session that RequireAuth injected into the request context.
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

// RequireAuth rejects unauthenticated requests and injects the full Session into the request context.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, ok := getSession(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := WithSession(r.Context(), session)
		r = r.WithContext(ctx)
		r.Header.Set("X-User", session.Username)
		next.ServeHTTP(w, r)
	})
}

// StartSessionCleanup periodically deletes expired sessions from Postgres.
func StartSessionCleanup() {
	go func() {
		for {
			time.Sleep(10 * time.Minute)
			pool.Exec(context.Background(), `DELETE FROM sessions WHERE expires_at < now()`)
		}
	}()
}

// ExpireSessionForTesting forcibly expires a session identified by its cookie value.
// This is intended only for use in unit tests — it has no effect when pool != nil.
func ExpireSessionForTesting(sessionID string) {
	if pool != nil {
		return
	}
	inMemorySessions.Lock()
	if entry, ok := inMemorySessions.data[sessionID]; ok {
		entry.expiresAt = time.Now().Add(-time.Second)
		entry.session.ExpiresAt = entry.expiresAt
		inMemorySessions.data[sessionID] = entry
	}
	inMemorySessions.Unlock()
}
