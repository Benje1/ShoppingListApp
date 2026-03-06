package authentication

import (
	"context"
	"errors"

	"weekly-shopping-app/database"
	sqlc "weekly-shopping-app/database/sqlc"

	"golang.org/x/crypto/bcrypt"
)

func LoginService(ctx context.Context, repo database.UserRepository, username, password string) (*sqlc.User, error) {
	user, err := repo.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, errors.New("invalid username or password, could not get user")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, errors.New("invalid username or password")
	}

	return user, nil
}
