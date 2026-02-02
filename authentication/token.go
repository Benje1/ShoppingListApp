package authentication

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

var (
	sessions   = make(map[string]Session)
	sessionMux sync.Mutex
)

type Session struct {
	Username  string
	ExpiresAt time.Time
}

func newSessionID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func CreateSession(w http.ResponseWriter, username string) {
	sessionID := newSessionID()

	session := Session{
		Username:  username,
		ExpiresAt: time.Now().Add(30 * time.Minute),
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

	// Check expiration
	if time.Now().After(session.ExpiresAt) {
		delete(sessions, cookie.Value)
		return "", false
	}

	return session.Username, true
}

func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := GetUser(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		r.Header.Set("X-User", user)
		next(w, r)
	}
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
