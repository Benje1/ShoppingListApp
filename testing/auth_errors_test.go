package authntication_test

// auth_errors_test.go
// Tests for authentication error paths:
//   - Login with wrong password returns an error
//   - Login with non-existent username returns an error
//   - Both cases return the same generic message (no user enumeration)
//   - ClientError is set so the HTTP layer returns 400, not 500

import (
	"context"
	"errors"
	"testing"

	"weekly-shopping-app/authentication"
	"weekly-shopping-app/internal/api/httpx"
	sqlc "weekly-shopping-app/database/sqlc"
)

func TestLoginService_WrongPassword_ReturnsError(t *testing.T) {
	hash, err := authentication.HashPassword("correct")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	repo := &FakeUserRepo{User: &sqlc.User{Username: "alice", PasswordHash: hash}}

	_, err = authentication.LoginService(context.Background(), repo, "alice", "wrong")
	if err == nil {
		t.Fatal("expected error for wrong password, got nil")
	}
}

func TestLoginService_WrongPassword_IsClientError(t *testing.T) {
	hash, _ := authentication.HashPassword("correct")
	repo := &FakeUserRepo{User: &sqlc.User{Username: "alice", PasswordHash: hash}}

	_, err := authentication.LoginService(context.Background(), repo, "alice", "wrong")

	var ce httpx.ClientError
	if !errors.As(err, &ce) {
		t.Fatalf("expected ClientError (maps to HTTP 400), got %T: %v", err, err)
	}
}

func TestLoginService_NonExistentUser_ReturnsError(t *testing.T) {
	repo := &FakeUserRepo{} // no user stored

	_, err := authentication.LoginService(context.Background(), repo, "nobody", "pass")
	if err == nil {
		t.Fatal("expected error for non-existent user, got nil")
	}
}

func TestLoginService_NonExistentUser_IsClientError(t *testing.T) {
	repo := &FakeUserRepo{}

	_, err := authentication.LoginService(context.Background(), repo, "nobody", "pass")

	var ce httpx.ClientError
	if !errors.As(err, &ce) {
		t.Fatalf("expected ClientError (maps to HTTP 400), got %T: %v", err, err)
	}
}

func TestLoginService_NoUserEnumeration(t *testing.T) {
	// Wrong password and non-existent user must return the same message so
	// callers cannot distinguish between the two cases.
	hash, _ := authentication.HashPassword("correct")
	repoWithUser := &FakeUserRepo{User: &sqlc.User{Username: "alice", PasswordHash: hash}}
	repoEmpty := &FakeUserRepo{}

	_, errWrongPass := authentication.LoginService(context.Background(), repoWithUser, "alice", "wrong")
	_, errNoUser := authentication.LoginService(context.Background(), repoEmpty, "alice", "anything")

	if errWrongPass == nil || errNoUser == nil {
		t.Fatal("both cases should return errors")
	}
	if errWrongPass.Error() != errNoUser.Error() {
		t.Errorf("error messages differ — user enumeration possible:\n  wrong password: %q\n  no user:        %q",
			errWrongPass.Error(), errNoUser.Error())
	}
}

func TestLoginService_CorrectCredentials_Succeeds(t *testing.T) {
	hash, _ := authentication.HashPassword("secret")
	repo := &FakeUserRepo{User: &sqlc.User{Username: "bob", Name: "Bob", PasswordHash: hash}}

	user, err := authentication.LoginService(context.Background(), repo, "bob", "secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Username != "bob" {
		t.Errorf("expected Username=bob, got %q", user.Username)
	}
}
