package authentication

import (
	"context"
	"errors"

	"weekly-shopping-app/database"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func LoginService(ctx context.Context, db *pgxpool.Pool, username, password string) error {
	user, err := database.GetUserByUsername(ctx, db, username)
	if err != nil {
		return errors.New("invalid username or password")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return errors.New("invalid username or password")
	}

	return nil
}
