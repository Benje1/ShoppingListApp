package integration_test

import (
	"context"
	"testing"

	"weekly-shopping-app/authentication"
	"weekly-shopping-app/database"
)

func newUserRepo() *database.PostgresUserRepo {
	return &database.PostgresUserRepo{DB: sharedPool()}
}

// ── InsertUser ────────────────────────────────────────────────────────────────

func TestIntegration_InsertUser_Succeeds(t *testing.T) {
	repo := newUserRepo()
	_, username, _ := makeUser(t)

	// Verify we can fetch back what was just created.
	row, err := repo.GetUserByUsername(context.Background(), username)
	if err != nil {
		t.Fatalf("GetUserByUsername after insert: %v", err)
	}
	if row.Username != username {
		t.Errorf("expected Username=%q, got %q", username, row.Username)
	}
	if row.Name != "Test User" {
		t.Errorf("expected Name=Test User, got %q", row.Name)
	}
	if row.ID == 0 {
		t.Error("expected a non-zero ID")
	}
}

func TestIntegration_InsertUser_DuplicateUsername_Fails(t *testing.T) {
	repo := newUserRepo()
	// SeedUser.Username already exists — inserting it again must fail.
	hash, _ := authentication.HashPassword("any")
	_, err := repo.InsertUser(context.Background(), "Duplicate", SeedUser.Username, hash)
	if err == nil {
		t.Fatal("expected an error inserting a duplicate username, got nil")
	}
}

// ── GetUserByUsername ─────────────────────────────────────────────────────────

func TestIntegration_GetUserByUsername_ReturnsSeededUser(t *testing.T) {
	row, err := newUserRepo().GetUserByUsername(context.Background(), SeedUser.Username)
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}
	if row.ID != SeedUser.ID {
		t.Errorf("expected ID=%d, got %d", SeedUser.ID, row.ID)
	}
}

func TestIntegration_GetUserByUsername_NotFound_ReturnsError(t *testing.T) {
	_, err := newUserRepo().GetUserByUsername(context.Background(), "does_not_exist_xyz")
	if err == nil {
		t.Fatal("expected an error for unknown username, got nil")
	}
}

// ── LoginService ──────────────────────────────────────────────────────────────

func TestIntegration_LoginService_CorrectPassword_Succeeds(t *testing.T) {
	_, username, password := makeUser(t)

	safe, err := authentication.LoginService(context.Background(), newUserRepo(), username, password)
	if err != nil {
		t.Fatalf("LoginService: %v", err)
	}
	if safe.Username != username {
		t.Errorf("expected Username=%q, got %q", username, safe.Username)
	}
}

func TestIntegration_LoginService_WrongPassword_Fails(t *testing.T) {
	_, username, _ := makeUser(t)

	_, err := authentication.LoginService(context.Background(), newUserRepo(), username, "wrong-pass")
	if err == nil {
		t.Fatal("expected an error for wrong password, got nil")
	}
}

func TestIntegration_LoginService_UnknownUser_Fails(t *testing.T) {
	_, err := authentication.LoginService(context.Background(), newUserRepo(), "ghost_user_xyz", "any")
	if err == nil {
		t.Fatal("expected an error for unknown user, got nil")
	}
}

// ── UpdateUserName ────────────────────────────────────────────────────────────

func TestIntegration_UpdateUserName_Succeeds(t *testing.T) {
	repo := newUserRepo()
	_, username, _ := makeUser(t)

	updated, err := repo.UpdateUserName(context.Background(), username, "Updated Name")
	if err != nil {
		t.Fatalf("UpdateUserName: %v", err)
	}
	if updated.Name != "Updated Name" {
		t.Errorf("expected Name=Updated Name, got %q", updated.Name)
	}
}

// ── UpdateUserPassword ────────────────────────────────────────────────────────

func TestIntegration_UpdateUserPassword_NewPasswordWorks(t *testing.T) {
	repo := newUserRepo()
	ctx := context.Background()
	_, username, _ := makeUser(t)

	newHash, err := authentication.HashPassword("new-pass")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if _, err := repo.UpdateUserPassword(ctx, username, newHash); err != nil {
		t.Fatalf("UpdateUserPassword: %v", err)
	}

	if _, err := authentication.LoginService(ctx, repo, username, "test_password"); err == nil {
		t.Fatal("expected old password to fail after update")
	}
	if _, err := authentication.LoginService(ctx, repo, username, "new-pass"); err != nil {
		t.Fatalf("expected new password to succeed: %v", err)
	}
}
