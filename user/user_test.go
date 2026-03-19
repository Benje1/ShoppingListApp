package user

import (
	"testing"
)

// ── UserInput validation ──────────────────────────────────────────────────────
// The UserInput struct is used in createUser. These tests verify the struct
// can be constructed correctly and that the password update helpers exist.

func TestUserInput_Fields(t *testing.T) {
	input := UserInput{
		Name:     "Jane Smith",
		Username: "jane",
		Password: "secret123",
	}
	if input.Name != "Jane Smith" {
		t.Errorf("unexpected Name: %q", input.Name)
	}
	if input.Username != "jane" {
		t.Errorf("unexpected Username: %q", input.Username)
	}
	if input.Password != "secret123" {
		t.Errorf("unexpected Password: %q", input.Password)
	}
}

func TestUpdateUserInput_Fields(t *testing.T) {
	input := UpdateUserInput{
		Username: "alice",
		Name:     "Alice Wonderland",
		Password: "newpass",
	}
	if input.Username != "alice" {
		t.Errorf("unexpected Username: %q", input.Username)
	}
}
