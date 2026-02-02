package authentication

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
)

var (
	sessions   = make(map[string]string)
	sessionMux sync.Mutex
)

func newSessionID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func CreateSession(w http.ResponseWriter, username string) {
	sessionID := newSessionID()

	sessionMux.Lock()
	sessions[sessionID] = username
	sessionMux.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // true in prod
		SameSite: http.SameSiteStrictMode,
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

	user, ok := sessions[cookie.Value]
	return user, ok
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