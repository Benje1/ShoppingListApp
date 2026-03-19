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
	InsertHousehold(ctx context.Context, numPeople int32, name string) (*sqlc.Household, error)
	GetHousehold(ctx context.Context, householdID int32) (*sqlc.Household, error)
	RenameHousehold(ctx context.Context, householdID int32, name string) (*sqlc.Household, error)
	DeleteHousehold(ctx context.Context, householdID int32) error
	GetHouseholdMembers(ctx context.Context, householdID int32) ([]sqlc.GetHouseholdMembersRow, error)
	CreateInvite(ctx context.Context, householdID int32, code string, userID int32) (*sqlc.HouseholdInvite, error)
	GetInviteByCode(ctx context.Context, code string) (*sqlc.HouseholdInvite, error)
	GetInviteByID(ctx context.Context, id int32) (*sqlc.HouseholdInvite, error)
	GetPendingInvites(ctx context.Context, householdID int32) ([]sqlc.GetPendingInvitesForHouseholdRow, error)
	RespondToInvite(ctx context.Context, inviteID int32, status string) (*sqlc.HouseholdInvite, error)
}
