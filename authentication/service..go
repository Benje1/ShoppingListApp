package authentication

import (
	"context"
	"errors"

	"weekly-shopping-app/database"

	"golang.org/x/crypto/bcrypt"
)

func LoginService(ctx context.Context, repo database.UserRepository, username, password string) error {
	user, err := repo.GetUserByUsername(ctx, username)
	if err != nil {
		return errors.New("invalid username or password, could not get user")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return errors.New("invalid username or password")
	}

	return nil
}
