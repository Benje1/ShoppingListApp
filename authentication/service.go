package authentication

import (
	"context"
	"errors"

	"weekly-shopping-app/database"

	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

type SafeUser struct {
	ID           int32            `json:"id"`
	Name         string           `json:"name"`
	Username     string           `json:"username"`
	CreatedAt    pgtype.Timestamp `json:"created_at"`
	HouseholdIds interface{}      `json:"household_ids"`
}

func LoginService(ctx context.Context, repo database.UserRepository, username, password string) (*SafeUser, error) {
	user, err := repo.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, errors.New("invalid username or password")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, errors.New("invalid username or password")
	}

	return &SafeUser{
		ID:           user.ID,
		Name:         user.Name,
		Username:     user.Username,
		CreatedAt:    user.CreatedAt,
		HouseholdIds: user.HouseholdIds,
	}, nil
}
