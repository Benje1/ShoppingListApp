package authntication_test

import (
	"net/http/httptest"
	"testing"
	"testing/synctest"
	"time"

	"weekly-shopping-app/authentication"
)

func TestSessionTokenExpiration(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := httptest.NewRecorder()
		authentication.CreateSession(w, "test")
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

		time.Sleep(time.Minute * 31)
		synctest.Wait()

		user, ok = authentication.GetUser(r)
		if ok {
			t.Fatal("session has not expired")
		}

		if user != "" {
			t.Fatal("session expired")
		}
	})
}
