package authentication

import (
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

type Session struct {
	Username     string
	UserID       int32
	HouseholdIds []int32
	ExpiresAt    time.Time
}

func newSessionID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func CreateSession(w http.ResponseWriter, username string, userID int32, households []int32) {
	sessionID := newSessionID()

	session := Session{
		Username:     username,
		UserID:       userID,
		HouseholdIds: households,
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

func GetUser(r *http.Request) (string, bool) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return "", false
	}

	sessionMux.Lock()
	defer sessionMux.Unlock()

	session, ok := sessions[cookie.Value]
	if !ok {
		return "", false
	}

	if time.Now().After(session.ExpiresAt) {
		delete(sessions, cookie.Value)
		return "", false
	}

	return session.Username, true
}

// GetUserID returns the numeric ID of the currently authenticated user.
// The ID is stored in the session at login time.
func GetUserID(r *http.Request) (int32, error) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return 0, errors.New("not authenticated")
	}

	sessionMux.Lock()
	defer sessionMux.Unlock()

	session, ok := sessions[cookie.Value]
	if !ok {
		return 0, errors.New("session not found")
	}
	if time.Now().After(session.ExpiresAt) {
		delete(sessions, cookie.Value)
		return 0, errors.New("session expired")
	}
	if session.UserID == 0 {
		return 0, fmt.Errorf("user ID not in session (re-login required)")
	}
	return session.UserID, nil
}

// RequireAuth is standard http.Handler middleware that rejects unauthenticated
// requests. RegisterEndpoint applies this automatically when Public is false.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := GetUser(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		r.Header.Set("X-User", user)
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
