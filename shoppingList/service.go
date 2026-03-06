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
		return getItemsFromList(db, r.Context())
	}
}

func createItemPost(db *pgxpool.Pool) func(*http.Request, sqlc.CreateShoppingItemParams) (any, error) {
	return func(r *http.Request, input sqlc.CreateShoppingItemParams) (any, error) {
		if err := addItemToList(r.Context(), db, input); err != nil {
			return nil, err
		}
		return map[string]string{"status": "item created"}, nil
	}
}

func seedItemsPost(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
	return func(r *http.Request, _ struct{}) (any, error) {
		if err := addItemsToShoppingList(r.Context(), db); err != nil {
			return nil, err
		}
		return map[string]string{"status": "shopping list seeded"}, nil
	}
}

func addItemsToShoppingList(ctx context.Context, db *pgxpool.Pool) error {
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

func addItemToList(ctx context.Context, db *pgxpool.Pool, params sqlc.CreateShoppingItemParams) error {
	q := sqlc.New(db)
	return q.CreateShoppingItem(ctx, params)
}

func getItemsFromList(db *pgxpool.Pool, ctx context.Context) ([]sqlc.ShoppingItem, error) {
	q := sqlc.New(db)
	return q.ListShoppingItems(ctx)
}
