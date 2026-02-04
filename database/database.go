package database

import (
	"context"
	"errors"
	"os"

	"weekly-shopping-app/database/sqlc"
	"weekly-shopping-app/shoppinglist"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Conn(ctx context.Context) (*pgxpool.Pool, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, errors.New("could not get database url")
	}

	return pgxpool.New(ctx, dbURL)
}

func AddItemsToShoppingList(pool *pgxpool.Pool, ctx context.Context) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	q := database.New(pool)

	for _, item := range shoppinglist.ShoppingList {
		err := q.CreateShoppingItem(ctx, database.CreateShoppingItemParams{
			Name:     item.Name,
			ItemType: database.ShoppingItemType(item.ItemType),
		})
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func GetItemsFromList(pool *pgxpool.Pool, ctx context.Context) ([]database.ShoppingItem, error) {
	q := database.New(pool)

	return q.ListShoppingItems(ctx)
}
