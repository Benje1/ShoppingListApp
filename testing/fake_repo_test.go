package authntication_test

import (
	"context"
	"encoding/json"
	"errors"

	sqlc "weekly-shopping-app/database/sqlc"
)

type FakeUserRepo struct {
	User *sqlc.User
}

func (f *FakeUserRepo) GetUserByUsername(_ context.Context, username string) (*sqlc.GetUserByUsernameRow, error) {
	if f.User == nil || f.User.Username != username {
		return nil, errors.New("not found")
	}
	// Return empty households array — tests only care about auth, not household data
	emptyHouseholds, _ := json.Marshal([]sqlc.UserHousehold{})
	return &sqlc.GetUserByUsernameRow{
		ID:           f.User.ID,
		Name:         f.User.Name,
		Username:     f.User.Username,
		PasswordHash: f.User.PasswordHash,
		CreatedAt:    f.User.CreatedAt,
		Households:   json.RawMessage(emptyHouseholds),
	}, nil
}

func (f *FakeUserRepo) InsertUser(_ context.Context, name, username, passwordHash string) (*sqlc.User, error) {
	f.User = &sqlc.User{Name: name, Username: username, PasswordHash: passwordHash}
	return f.User, nil
}

func (f *FakeUserRepo) AddUserToHousehold(_ context.Context, _, _ int32) error { return nil }

func (f *FakeUserRepo) UpdateUserName(_ context.Context, _, name string) (*sqlc.User, error) {
	f.User.Name = name
	return f.User, nil
}

func (f *FakeUserRepo) UpdateUserPassword(_ context.Context, _, passwordHash string) (*sqlc.User, error) {
	f.User.PasswordHash = passwordHash
	return f.User, nil
}

func (f *FakeUserRepo) UpdateUserHouseholdMemberships(_ context.Context, _, _ int32) error {
	return nil
}
