package shoppinglist

import (
	"context"
	"net/http"

	"weekly-shopping-app/authentication"
	sqlc "weekly-shopping-app/database/sqlc"
	"weekly-shopping-app/internal/api/httpx"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterShoppingListRoutes(mux *http.ServeMux, db *pgxpool.Pool, wrap func(httpx.AppHandler) http.HandlerFunc) {
	r := httpx.NewRouter(mux, db, wrap, authentication.RequireAuth, "/shopping")

	httpx.RegisterEndpoint(r, httpx.EndpointConfig[struct{}]{
		Path:   "/items",
		Method: "GET",
		Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
			return listItemsGet(db)
		},
	})

	httpx.RegisterEndpoint(r, httpx.EndpointConfig[sqlc.CreateShoppingItemParams]{
		Path:   "/items/create",
		Method: "POST",
		Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, sqlc.CreateShoppingItemParams) (any, error) {
			return createItemPost(db)
		},
	})

	httpx.RegisterEndpoint(r, httpx.EndpointConfig[struct{}]{
		Path:   "/items/seed",
		Method: "POST",
		Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
			return seedItemsPost(db)
		},
	})
}

func listItemsGet(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
	return func(r *http.Request, _ struct{}) (any, error) {
		return getItemsFromList(r.Context(), db)
	}
}

func createItemPost(db *pgxpool.Pool) func(*http.Request, sqlc.CreateShoppingItemParams) (any, error) {
	return func(r *http.Request, input sqlc.CreateShoppingItemParams) (any, error) {
		item, err := addItemToList(r.Context(), db, input)
		if err != nil {
			return nil, err
		}
		return item, nil
	}
}

func seedItemsPost(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
	return func(r *http.Request, _ struct{}) (any, error) {
		if err := seedShoppingList(r.Context(), db); err != nil {
			return nil, err
		}
		return map[string]string{"status": "shopping list seeded"}, nil
	}
}

func seedShoppingList(ctx context.Context, db *pgxpool.Pool) error {
	q := sqlc.New(db)
	for _, item := range ShoppingList {
		_, err := q.CreateShoppingItem(ctx, sqlc.CreateShoppingItemParams{
			Name:     item.Name,
			ItemType: sqlc.ShoppingItemType(item.ItemType),
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func addItemToList(ctx context.Context, db *pgxpool.Pool, params sqlc.CreateShoppingItemParams) (sqlc.ShoppingItem, error) {
	q := sqlc.New(db)
	return q.CreateShoppingItem(ctx, params)
}

func getItemsFromList(ctx context.Context, db *pgxpool.Pool) ([]sqlc.ListShoppingItemsRow, error) {
	q := sqlc.New(db)
	return q.ListShoppingItems(ctx)
}
