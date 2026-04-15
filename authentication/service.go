package authentication

import (
	"context"
	"encoding/json"
	"fmt"

	"weekly-shopping-app/database"

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
		return nil, fmt.Errorf("invalid username or password, (1): %w", err)
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid username or password, (2): %w", err)
	}

	// Households is returned from the DB as a JSON array (interface{}).
	// Marshal it back to bytes so we can unmarshal into the typed slice.
	var households []database.UserHousehold
	if user.Households != nil {
		raw, err := json.Marshal(user.Households)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal households: %w", err)
		}
		if err := json.Unmarshal(raw, &households); err != nil {
			return nil, fmt.Errorf("failed to unmarshal households: %w", err)
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
