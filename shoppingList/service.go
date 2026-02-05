package shoppinglist

import (
	"context"

	sqlc "weekly-shopping-app/database/sqlc"

	"github.com/jackc/pgx/v5/pgxpool"
)

func AddItemsToShoppingList(ctx context.Context, db *pgxpool.Pool) error {
	q := sqlc.New(db)

	for _, item := range ShoppingList {
		err := q.CreateShoppingItem(ctx, sqlc.CreateShoppingItemParams{
			Name:     item.Name,
			ItemType: sqlc.ShoppingItemType(item.ItemType),
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func GetItemsFromList(pool *pgxpool.Pool, ctx context.Context) ([]sqlc.ShoppingItem, error) {
	q := sqlc.New(pool)

	return q.ListShoppingItems(ctx)
}
