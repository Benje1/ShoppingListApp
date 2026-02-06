package authntication_test

import (
	"context"
	"testing"

	"weekly-shopping-app/authentication"
	"weekly-shopping-app/database"
)

func TestLoginService(t *testing.T) {
	t.Run("test with right password", func(t *testing.T) {
		hash, _ := authentication.HashPassword("secret")

		repo := &FakeUserRepo{
			User: &database.User{
				Username:     "bob",
				PasswordHash: hash,
			},
		}

		err := authentication.LoginService(context.Background(), repo, "bob", "secret")
		if err != nil {
			t.Fatal("expected login to succeed")
		}
	})

	t.Run("test with wrong password", func(t *testing.T) {
		hash, _ := authentication.HashPassword("secret")

		repo := &FakeUserRepo{
			User: &database.User{
				Username:     "bob",
				PasswordHash: hash,
			},
		}

		err := authentication.LoginService(context.Background(), repo, "bob", "wrong")
		if err == nil {
			t.Fatal("expected failure")
		}
	})
}
