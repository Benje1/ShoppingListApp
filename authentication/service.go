package authentication

import (
	"context"
	"encoding/json"
	"fmt"

	"weekly-shopping-app/database"
	"weekly-shopping-app/internal/logger"

	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

type SafeUser struct {
	ID         int32                    `json:"id"`
	Name       string                   `json:"name"`
	Username   string                   `json:"username"`
	CreatedAt  pgtype.Timestamp         `json:"created_at"`
	Households []database.UserHousehold `json:"households"`
}

func LoginService(ctx context.Context, repo database.UserRepository, username, password string) (*SafeUser, error) {
	user, err := repo.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, logger.WithStack(fmt.Errorf("invalid username or password: %w", err))
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, logger.WithStack(fmt.Errorf("invalid username or password: %w", err))
	}

	// Households is returned from the DB as a JSON []byte (pgx JSON column).
	var households []database.UserHousehold
	if user.Households != nil {
		raw, ok := user.Households.([]byte)
		if !ok {
			// Fallback: re-marshal if pgx decoded it to something else (e.g. during tests).
			var err error
			if raw, err = json.Marshal(user.Households); err != nil {
				return nil, logger.WithStack(fmt.Errorf("failed to encode households: %w", err))
			}
		}
		if err := json.Unmarshal(raw, &households); err != nil {
			return nil, logger.WithStack(fmt.Errorf("failed to decode households: %w", err))
		}
	}

	return &SafeUser{
		ID:         user.ID,
		Name:       user.Name,
		Username:   user.Username,
		CreatedAt:  user.CreatedAt,
		Households: households,
	}, nil
}
