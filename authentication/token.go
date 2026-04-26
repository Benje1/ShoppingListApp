package authentication

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// contextKey is an unexported type for context keys in this package.
type contextKey string

const sessionContextKey contextKey = "session"

// db is the pool used for session persistence.
// Initialised by InitSessionStore before any HTTP traffic is served.
var db *pgxpool.Pool

// InitSessionStore wires up the database pool used by all session functions.
// Call this once from main, before starting the HTTP server.
func InitSessionStore(pool *pgxpool.Pool) {
	db = pool
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

// CreateSession persists a new session to Postgres and sets the cookie.
func CreateSession(w http.ResponseWriter, username string, userID int32, householdIds []int32) {
	sessionID := newSessionID()
	expiresAt := time.Now().Add(7 * 24 * time.Hour) // 7 days

	// Convert []int32 to []int64 for pgx
	hids := make([]int64, len(householdIds))
	for i, h := range householdIds {
		hids[i] = int64(h)
	}

	_, err := db.Exec(context.Background(),
		`INSERT INTO sessions (id, user_id, username, household_ids, expires_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		sessionID, userID, username, hids, expiresAt,
	)
	if err != nil {
		fmt.Printf("session insert error: %v\n", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

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
}

// DestroySession deletes the session from Postgres and clears the cookie.
func DestroySession(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err == nil {
		db.Exec(context.Background(),
			`DELETE FROM sessions WHERE id = $1`, cookie.Value,
		)
	}

	http.SetCookie(w, &http.Cookie{
		Name:   "session_id",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
}

// getSession loads and validates a session from Postgres.
func getSession(r *http.Request) (Session, bool) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return Session{}, false
	}

	var (
		username  string
		userID    int32
		hids      []int64
		expiresAt time.Time
	)

	err = db.QueryRow(context.Background(),
		`SELECT username, user_id, household_ids, expires_at
		 FROM sessions WHERE id = $1`,
		cookie.Value,
	).Scan(&username, &userID, &hids, &expiresAt)

	if err != nil {
		return Session{}, false
	}

	if time.Now().After(expiresAt) {
		db.Exec(context.Background(), `DELETE FROM sessions WHERE id = $1`, cookie.Value)
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

// GetUser returns the username from the session cookie (kept for compatibility).
func GetUser(r *http.Request) (string, bool) {
	s, ok := getSession(r)
	return s.Username, ok
}

// GetUserID returns just the numeric user ID from the session cookie.
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
			db.Exec(context.Background(),
				`DELETE FROM sessions WHERE expires_at < now()`,
			)
		}
	}()
}
