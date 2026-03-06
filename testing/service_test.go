package authntication_test

import (
	"context"
	"testing"

	"weekly-shopping-app/authentication"
	"weekly-shopping-app/database/sqlc"
)

func TestLoginService(t *testing.T) {
	hash, err := authentication.HashPassword("secret")
	if err != nil {
		t.Fatalf("could not encrypt password: %v", err.Error())
	}

	repo := &FakeUserRepo{
		User: &database.User{
			Username:     "bob",
			Name:         "bob",
			PasswordHash: hash,
		},
	}

	t.Run("test with right password", func(t *testing.T) {
		var user *database.User
		user, err = authentication.LoginService(context.Background(), repo, "bob", "secret")
		if err != nil {
			t.Fatal("expected login to succeed")
		}
		if user.Username != "bob" {
			t.Fatal("user not expected")
		}
		if user.Name != "bob" {
			t.Fatal("user name not expected")
		}
	})

	t.Run("test with wrong password", func(t *testing.T) {
		_, err = authentication.LoginService(context.Background(), repo, "bob", "wrong")
		if err == nil {
			t.Fatal("expected failure")
		}
	})
}
