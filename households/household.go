package households

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"weekly-shopping-app/database"
	sqlc "weekly-shopping-app/database/sqlc"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ── Input types ───────────────────────────────────────────────────────────────

type CreateHouseholdInput struct {
	NumPeople int32  `json:"num_people"`
	Name      string `json:"name"`
}

type RenameHouseholdInput struct {
	Name string `json:"name"`
}

// RequestJoinInput is sent by a user who wants to join a household.
// They submit the invite code they received out-of-band.
type RequestJoinInput struct {
	InviteCode string `json:"invite_code"`
}

// RespondToInviteInput is sent by an existing household member to approve or deny.
type RespondToInviteInput struct {
	InviteID int32  `json:"invite_id"`
	Action   string `json:"action"` // "approve" or "deny"
}

// ── Response types ────────────────────────────────────────────────────────────

type HouseholdResponse struct {
	HouseholdID int32  `json:"household_id"`
	NumPeople   int32  `json:"num_people"`
	Name        string `json:"name"`
}

type InviteResponse struct {
	InviteID    int32  `json:"invite_id"`
	HouseholdID int32  `json:"household_id"`
	InviteCode  string `json:"invite_code"`
	Status      string `json:"status"`
}

type PendingInviteResponse struct {
	InviteID          int32  `json:"invite_id"`
	RequesterName     string `json:"requester_name"`
	RequesterUsername string `json:"requester_username"`
}

type MemberResponse struct {
	ID       int32  `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
}

type HouseholdDetailResponse struct {
	HouseholdID    int32                   `json:"household_id"`
	NumPeople      int32                   `json:"num_people"`
	Name           string                  `json:"name"`
	Members        []MemberResponse        `json:"members"`
	PendingInvites []PendingInviteResponse `json:"pending_invites"`
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func householdResponse(h *sqlc.Household) HouseholdResponse {
	name := ""
	if h.Name.Valid {
		name = h.Name.String
	}
	return HouseholdResponse{HouseholdID: h.HouseholdID, NumPeople: h.NumPeople, Name: name}
}

func generateCode() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating invite code: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func repo(db *pgxpool.Pool) *database.PostgresHouseholdRepo {
	return &database.PostgresHouseholdRepo{DB: db}
}

func userRepo(db *pgxpool.Pool) *database.PostgresUserRepo {
	return &database.PostgresUserRepo{DB: db}
}

// ── Business logic ────────────────────────────────────────────────────────────

func createHousehold(ctx context.Context, db *pgxpool.Pool, creatorUserID int32, input CreateHouseholdInput) (*HouseholdResponse, error) {
	np := input.NumPeople
	if np <= 0 {
		np = 1
	}
	h, err := repo(db).InsertHousehold(ctx, np, input.Name)
	if err != nil {
		return nil, err
	}
	// Automatically add the creator as the first member of the household
	if err := userRepo(db).AddUserToHousehold(ctx, creatorUserID, h.HouseholdID); err != nil {
		return nil, fmt.Errorf("adding creator to household: %w", err)
	}
	r := householdResponse(h)
	return &r, nil
}

func getHousehold(ctx context.Context, db *pgxpool.Pool, id int32) (*HouseholdResponse, error) {
	h, err := repo(db).GetHousehold(ctx, id)
	if err != nil {
		return nil, err
	}
	r := householdResponse(h)
	return &r, nil
}

func getHouseholdDetail(ctx context.Context, db *pgxpool.Pool, id int32) (*HouseholdDetailResponse, error) {
	r := repo(db)
	h, err := r.GetHousehold(ctx, id)
	if err != nil {
		return nil, err
	}
	members, err := r.GetHouseholdMembers(ctx, id)
	if err != nil {
		return nil, err
	}
	pending, err := r.GetPendingInvites(ctx, id)
	if err != nil {
		return nil, err
	}

	ms := make([]MemberResponse, len(members))
	for i, m := range members {
		ms[i] = MemberResponse{ID: m.ID, Name: m.Name, Username: m.Username}
	}
	ps := make([]PendingInviteResponse, len(pending))
	for i, p := range pending {
		ps[i] = PendingInviteResponse{InviteID: p.ID, RequesterName: p.RequesterName, RequesterUsername: p.RequesterUsername}
	}

	name := ""
	if h.Name.Valid {
		name = h.Name.String
	}
	return &HouseholdDetailResponse{
		HouseholdID:    h.HouseholdID,
		NumPeople:      h.NumPeople,
		Name:           name,
		Members:        ms,
		PendingInvites: ps,
	}, nil
}

func renameHousehold(ctx context.Context, db *pgxpool.Pool, id int32, input RenameHouseholdInput) (*HouseholdResponse, error) {
	if input.Name == "" {
		return nil, errors.New("name cannot be empty")
	}
	h, err := repo(db).RenameHousehold(ctx, id, input.Name)
	if err != nil {
		return nil, err
	}
	r := householdResponse(h)
	return &r, nil
}

func deleteHousehold(ctx context.Context, db *pgxpool.Pool, id int32) error {
	return repo(db).DeleteHousehold(ctx, id)
}

// requestJoin: the calling user submits a code to request joining a household.
// The code identifies the household; a new invite record is created with status=pending.
func requestJoin(ctx context.Context, db *pgxpool.Pool, userID int32, input RequestJoinInput) (*InviteResponse, error) {
	if input.InviteCode == "" {
		return nil, errors.New("invite_code is required")
	}

	// Resolve code -> household
	existing, err := repo(db).GetInviteByCode(ctx, input.InviteCode)
	if err != nil {
		return nil, errors.New("invite code not found")
	}

	// Create a new invite request from this user for that household
	code, err := generateCode()
	if err != nil {
		return nil, err
	}
	inv, err := repo(db).CreateInvite(ctx, existing.HouseholdID, code, userID)
	if err != nil {
		return nil, err
	}
	return &InviteResponse{
		InviteID:    inv.ID,
		HouseholdID: inv.HouseholdID,
		InviteCode:  inv.InviteCode,
		Status:      inv.Status,
	}, nil
}

// generateInviteCode: an existing household member generates a shareable code
// that others can use to request joining their household.
func generateInviteCode(ctx context.Context, db *pgxpool.Pool, householdID int32, requestingUserID int32) (*InviteResponse, error) {
	code, err := generateCode()
	if err != nil {
		return nil, err
	}
	inv, err := repo(db).CreateInvite(ctx, householdID, code, requestingUserID)
	if err != nil {
		return nil, err
	}
	return &InviteResponse{
		InviteID:    inv.ID,
		HouseholdID: inv.HouseholdID,
		InviteCode:  inv.InviteCode,
		Status:      inv.Status,
	}, nil
}

// respondToInvite: an existing member approves or denies a pending invite.
// On approval, the requesting user is added to the household.
func respondToInvite(ctx context.Context, db *pgxpool.Pool, input RespondToInviteInput) (*InviteResponse, error) {
	if input.Action != "approve" && input.Action != "deny" {
		return nil, errors.New("action must be 'approve' or 'deny'")
	}

	inv, err := repo(db).GetInviteByID(ctx, input.InviteID)
	if err != nil {
		return nil, errors.New("invite not found")
	}
	if inv.Status != "pending" {
		return nil, errors.New("invite is no longer pending")
	}

	status := "denied"
	if input.Action == "approve" {
		status = "approved"
		// Add the requesting user to the household
		if err := userRepo(db).AddUserToHousehold(ctx, inv.RequestedByUserID, inv.HouseholdID); err != nil {
			return nil, fmt.Errorf("adding user to household: %w", err)
		}
	}

	updated, err := repo(db).RespondToInvite(ctx, inv.ID, status)
	if err != nil {
		return nil, err
	}
	return &InviteResponse{
		InviteID:    updated.ID,
		HouseholdID: updated.HouseholdID,
		InviteCode:  updated.InviteCode,
		Status:      updated.Status,
	}, nil
}
