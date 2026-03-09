package database

import (
	"context"

	sqlc "weekly-shopping-app/database/sqlc"
)

type UserRepository interface {
	InsertUser(ctx context.Context, name, username, passwordHash string) (*sqlc.User, error)
	AddUserToHousehold(ctx context.Context, userID, householdID int32) error
	UpdateUserName(ctx context.Context, username, name string) (*sqlc.User, error)
	UpdateUserPassword(ctx context.Context, username, passwordHash string) (*sqlc.User, error)
	UpdateUserHouseholdMemberships(ctx context.Context, userID, householdID int32) error
	GetUserByUsername(ctx context.Context, username string) (*sqlc.GetUserByUsernameRow, error)
}

type HouseholdRepository interface {
	InsertHousehold(ctx context.Context, householdID int32) (*sqlc.Household, error)
	GetHousehold(ctx context.Context, householdID int32) (*sqlc.Household, error)
	DeleteHousehold(ctx context.Context, householdID int32) error
}
