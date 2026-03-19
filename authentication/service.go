package authentication

import (
	"context"
	"fmt"

	"weekly-shopping-app/database"
	sqlc "weekly-shopping-app/database/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

type SafeUser struct {
	ID         int32                `json:"id"`
	Name       string               `json:"name"`
	Username   string               `json:"username"`
	CreatedAt  pgtype.Timestamp     `json:"created_at"`
	Households []sqlc.UserHousehold `json:"households"`
}

func LoginService(ctx context.Context, repo database.UserRepository, username, password string) (*SafeUser, error) {
	user, err := repo.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("invalid username or password, (1): %w", err)
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid username or password, (2): %w", err)
	}

	households, err := user.ParseHouseholds()
	if err != nil {
		return nil, fmt.Errorf("parsing households: %w", err)
	}

	return &SafeUser{
		ID:         user.ID,
		Name:       user.Name,
		Username:   user.Username,
		CreatedAt:  user.CreatedAt,
		Households: households,
	}, nil
}
