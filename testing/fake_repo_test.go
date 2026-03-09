package authntication_test

import (
	"context"
	"errors"

	"weekly-shopping-app/authentication"
	"weekly-shopping-app/database/sqlc"
)

type FakeUserRepo struct {
	User     *database.User
	GetUser  *database.GetUserByUsernameRow
	SafeUser *authentication.SafeUser
}

func (f *FakeUserRepo) GetUserByUsername(ctx context.Context, username string) (*database.GetUserByUsernameRow, error) {
	if f.User == nil || f.User.Username != username {
		return nil, errors.New("not found")
	}
	return f.GetUser, nil
}

func (f *FakeUserRepo) InsertUser(ctx context.Context, name, username, passwordHash string) (*database.User, error) {
	f.User = &database.User{
		Name:         name,
		Username:     username,
		PasswordHash: passwordHash,
	}
	return f.User, nil
}

func (f *FakeUserRepo) AddUserToHousehold(_ context.Context, _, _ int32) error {
	return nil
}

func (f *FakeUserRepo) UpdateUserName(ctx context.Context, username, name string) (*database.User, error) {
	f.User.Name = name
	return f.User, nil
}

func (f *FakeUserRepo) UpdateUserPassword(ctx context.Context, username, passwordHash string) (*database.User, error) {
	f.User.PasswordHash = passwordHash
	return f.User, nil
}

func (f *FakeUserRepo) UpdateUserHouseholdMemberships(_ context.Context, _, _ int32) error {
	return nil
}
