package database

import (
	"context"

	sqlc "weekly-shopping-app/database/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresHouseholdRepo struct {
	DB *pgxpool.Pool
}

func (p *PostgresHouseholdRepo) InsertHousehold(ctx context.Context, numPeople int32, name string) (*sqlc.Household, error) {
	q := sqlc.New(p.DB)
	var pgName pgtype.Text
	if name != "" {
		pgName = pgtype.Text{String: name, Valid: true}
	}
	h, err := q.InsertHousehold(ctx, sqlc.InsertHouseholdParams{NumPeople: numPeople, Name: pgName})
	if err != nil {
		return nil, err
	}
	return &h, nil
}

func (p *PostgresHouseholdRepo) GetHousehold(ctx context.Context, householdID int32) (*sqlc.Household, error) {
	q := sqlc.New(p.DB)
	h, err := q.GetHousehold(ctx, householdID)
	if err != nil {
		return nil, err
	}
	return &h, nil
}

func (p *PostgresHouseholdRepo) RenameHousehold(ctx context.Context, householdID int32, name string) (*sqlc.Household, error) {
	q := sqlc.New(p.DB)
	h, err := q.RenameHousehold(ctx, sqlc.RenameHouseholdParams{
		HouseholdID: householdID,
		Name:        pgtype.Text{String: name, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	return &h, nil
}

func (p *PostgresHouseholdRepo) DeleteHousehold(ctx context.Context, householdID int32) error {
	q := sqlc.New(p.DB)
	return q.DeleteHousehold(ctx, householdID)
}

func (p *PostgresHouseholdRepo) GetHouseholdMembers(ctx context.Context, householdID int32) ([]sqlc.GetHouseholdMembersRow, error) {
	q := sqlc.New(p.DB)
	return q.GetHouseholdMembers(ctx, householdID)
}

func (p *PostgresHouseholdRepo) CreateInvite(ctx context.Context, householdID int32, code string, userID int32) (*sqlc.HouseholdInvite, error) {
	q := sqlc.New(p.DB)
	inv, err := q.CreateInvite(ctx, sqlc.CreateInviteParams{
		HouseholdID:       householdID,
		InviteCode:        code,
		RequestedByUserID: userID,
	})
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

func (p *PostgresHouseholdRepo) GetInviteByCode(ctx context.Context, code string) (*sqlc.HouseholdInvite, error) {
	q := sqlc.New(p.DB)
	inv, err := q.GetInviteByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

func (p *PostgresHouseholdRepo) GetInviteByID(ctx context.Context, id int32) (*sqlc.HouseholdInvite, error) {
	q := sqlc.New(p.DB)
	inv, err := q.GetInviteByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

func (p *PostgresHouseholdRepo) GetPendingInvites(ctx context.Context, householdID int32) ([]sqlc.GetPendingInvitesForHouseholdRow, error) {
	q := sqlc.New(p.DB)
	return q.GetPendingInvitesForHousehold(ctx, householdID)
}

func (p *PostgresHouseholdRepo) RespondToInvite(ctx context.Context, inviteID int32, status string) (*sqlc.HouseholdInvite, error) {
	q := sqlc.New(p.DB)
	inv, err := q.RespondToInvite(ctx, sqlc.RespondToInviteParams{ID: inviteID, Status: status})
	if err != nil {
		return nil, err
	}
	return &inv, nil
}
