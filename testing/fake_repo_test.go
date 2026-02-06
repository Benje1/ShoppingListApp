package authntication_test

import (
	"context"
	"errors"

	"weekly-shopping-app/database"
)

type FakeUserRepo struct {
	User *database.User
}

func (f *FakeUserRepo) GetUserByUsername(ctx context.Context, username string) (*database.User, error) {
	if f.User == nil || f.User.Username != username {
		return nil, errors.New("not found")
	}
	return f.User, nil
}

func (f *FakeUserRepo) InsertUser(ctx context.Context, name, username, passwordHash string, household uint) error {
	f.User = &database.User{
		Name: name, Username: username, PasswordHash: passwordHash, Household: int(household),
	}
	return nil
}

func (f *FakeUserRepo) UpdateUser(ctx context.Context, username, name, passwordHash string) error {
	f.User.Name = name
	f.User.PasswordHash = passwordHash
	return nil
}
