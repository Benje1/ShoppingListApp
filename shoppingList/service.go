package shoppinglist

import (
	"context"
	"fmt"
	"net/http"

	"weekly-shopping-app/authentication"
	sqlc "weekly-shopping-app/database/sqlc"
	"weekly-shopping-app/internal/api/httpx"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterShoppingListRoutes(mux *http.ServeMux, db *pgxpool.Pool, wrap func(httpx.AppHandler) http.HandlerFunc) {
	r := httpx.NewRouter(mux, db, wrap, authentication.RequireAuth, "/shopping")

	// GET /shopping/items — all items in the catalogue (for Shopping Items tab)
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[struct{}]{
		Path: "/items", Method: "GET", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
			return func(r *http.Request, _ struct{}) (any, error) {
				return getItemsFromList(r.Context(), db)
			}
		},
	})

	// POST /shopping/items/create — add a new item to the catalogue
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[sqlc.CreateShoppingItemParams]{
		Path: "/items/create", Method: "POST", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, sqlc.CreateShoppingItemParams) (any, error) {
			return func(r *http.Request, input sqlc.CreateShoppingItemParams) (any, error) {
				return addItemToList(r.Context(), db, input)
			}
		},
	})

	// POST /shopping/items/seed
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[struct{}]{
		Path: "/items/seed", Method: "POST", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
			return func(r *http.Request, _ struct{}) (any, error) {
				if err := seedShoppingList(r.Context(), db); err != nil {
					return nil, err
				}
				return map[string]string{"status": "shopping list seeded"}, nil
			}
		},
	})

	// GET /shopping/list — the user's personal + household shopping list
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[struct{}]{
		Path: "/list", Method: "GET", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
			return func(r *http.Request, _ struct{}) (any, error) {
				userID, err := authentication.GetUserID(r)
				if err != nil {
					return nil, err
				}
				householdID := int32(0)
				if hid := r.URL.Query().Get("household_id"); hid != "" {
					fmt.Sscanf(hid, "%d", &householdID)
				}
				return getShoppingList(r.Context(), db, userID, householdID)
			}
		},
	})

	// GET /shopping/list/updated-at — timestamp of most recent change
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[struct{}]{
		Path: "/list/updated-at", Method: "GET", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
			return func(r *http.Request, _ struct{}) (any, error) {
				userID, err := authentication.GetUserID(r)
				if err != nil {
					return nil, err
				}
				householdID := int32(0)
				if hid := r.URL.Query().Get("household_id"); hid != "" {
					fmt.Sscanf(hid, "%d", &householdID)
				}
				return getShoppingListUpdatedAt(r.Context(), db, userID, householdID)
			}
		},
	})

	// POST /shopping/list/add — add a catalogue item to the shopping list
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[AddToListInput]{
		Path: "/list/add", Method: "POST", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, AddToListInput) (any, error) {
			return func(r *http.Request, input AddToListInput) (any, error) {
				userID, err := authentication.GetUserID(r)
				if err != nil {
					return nil, err
				}
				return addToShoppingList(r.Context(), db, userID, input)
			}
		},
	})

	// DELETE /shopping/list/remove?id=<list_entry_id>
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[struct{}]{
		Path: "/list/remove", Method: "DELETE", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
			return func(r *http.Request, _ struct{}) (any, error) {
				var id int32
				fmt.Sscanf(r.URL.Query().Get("id"), "%d", &id)
				if id == 0 {
					return nil, fmt.Errorf("id is required")
				}
				q := sqlc.New(db)
				if err := q.RemoveFromShoppingList(r.Context(), id); err != nil {
					return nil, err
				}
				return map[string]string{"status": "removed"}, nil
			}
		},
	})

	// GET /shopping/mealplan
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[struct{}]{
		Path: "/mealplan", Method: "GET", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
			return func(r *http.Request, _ struct{}) (any, error) {
				userID, err := authentication.GetUserID(r)
				if err != nil {
					return nil, err
				}
				householdID := int32(0)
				if hid := r.URL.Query().Get("household_id"); hid != "" {
					fmt.Sscanf(hid, "%d", &householdID)
				}
				return getMealPlan(r.Context(), db, userID, householdID)
			}
		},
	})

	// POST /shopping/mealplan/save — upsert a single day's meal
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[UpsertMealPlanInput]{
		Path: "/mealplan/save", Method: "POST", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, UpsertMealPlanInput) (any, error) {
			return func(r *http.Request, input UpsertMealPlanInput) (any, error) {
				userID, err := authentication.GetUserID(r)
				if err != nil {
					return nil, err
				}
				return upsertMealPlanDay(r.Context(), db, userID, input)
			}
		},
	})

	// GET /shopping/mealplan/updated-at
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[struct{}]{
		Path: "/mealplan/updated-at", Method: "GET", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
			return func(r *http.Request, _ struct{}) (any, error) {
				userID, err := authentication.GetUserID(r)
				if err != nil {
					return nil, err
				}
				householdID := int32(0)
				if hid := r.URL.Query().Get("household_id"); hid != "" {
					fmt.Sscanf(hid, "%d", &householdID)
				}
				return getMealPlanUpdatedAt(r.Context(), db, userID, householdID)
			}
		},
	})

	// GET /shopping/list/have-it — fetch the full have-it set + updated-at
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[struct{}]{
		Path: "/list/have-it", Method: "GET", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
			return func(r *http.Request, _ struct{}) (any, error) {
				userID, err := authentication.GetUserID(r)
				if err != nil {
					return nil, err
				}
				householdID := int32(0)
				if hid := r.URL.Query().Get("household_id"); hid != "" {
					fmt.Sscanf(hid, "%d", &householdID)
				}
				return getHaveIt(r.Context(), db, userID, householdID)
			}
		},
	})

	// POST /shopping/list/have-it — mark an item as have-it
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[HaveItInput]{
		Path: "/list/have-it/mark", Method: "POST", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, HaveItInput) (any, error) {
			return func(r *http.Request, input HaveItInput) (any, error) {
				userID, err := authentication.GetUserID(r)
				if err != nil {
					return nil, err
				}
				return markHaveIt(r.Context(), db, userID, input)
			}
		},
	})

	// POST /shopping/list/have-it/unmark — unmark an item
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[HaveItInput]{
		Path: "/list/have-it/unmark", Method: "POST", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, HaveItInput) (any, error) {
			return func(r *http.Request, input HaveItInput) (any, error) {
				userID, err := authentication.GetUserID(r)
				if err != nil {
					return nil, err
				}
				return unmarkHaveIt(r.Context(), db, userID, input)
			}
		},
	})
}

// ── Input types ───────────────────────────────────────────────────────────────

type AddToListInput struct {
	ItemID      int32  `json:"item_id"`
	Quantity    int32  `json:"quantity"`
	Scope       string `json:"scope"` // "personal" or "household"
	HouseholdID int32  `json:"household_id"`
}

type UpsertMealPlanInput struct {
	DayName     string `json:"day_name"`
	MealName    string `json:"meal_name"`
	Scope       string `json:"scope"` // "personal" or "household"
	HouseholdID int32  `json:"household_id"`
}

// ── Business logic ────────────────────────────────────────────────────────────

func scopeParams(userID, householdID int32, scope string) (pgtype.Int4, pgtype.Int4) {
	var hid, uid pgtype.Int4
	if scope == "household" && householdID != 0 {
		hid = pgtype.Int4{Int32: householdID, Valid: true}
	} else {
		uid = pgtype.Int4{Int32: userID, Valid: true}
	}
	return hid, uid
}

func listParams(userID, householdID int32) sqlc.GetShoppingListParams {
	p := sqlc.GetShoppingListParams{
		UserID: pgtype.Int4{Int32: userID, Valid: true},
	}
	if householdID != 0 {
		p.HouseholdID = pgtype.Int4{Int32: householdID, Valid: true}
	}
	return p
}

func getShoppingList(ctx context.Context, db *pgxpool.Pool, userID, householdID int32) ([]sqlc.GetShoppingListRow, error) {
	q := sqlc.New(db)
	rows, err := q.GetShoppingList(ctx, listParams(userID, householdID))
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return []sqlc.GetShoppingListRow{}, nil
	}
	return rows, nil
}

func getShoppingListUpdatedAt(ctx context.Context, db *pgxpool.Pool, userID, householdID int32) (map[string]any, error) {
	q := sqlc.New(db)
	t, err := q.GetShoppingListUpdatedAt(ctx, listParams(userID, householdID))
	if err != nil {
		return nil, err
	}
	result := map[string]any{"last_updated": nil}
	if t.Valid {
		result["last_updated"] = t.Time.UTC().Format("2006-01-02T15:04:05Z")
	}
	return result, nil
}

func addToShoppingList(ctx context.Context, db *pgxpool.Pool, userID int32, input AddToListInput) (sqlc.ShoppingListEntry, error) {
	if input.Quantity <= 0 {
		input.Quantity = 1
	}
	hid, uid := scopeParams(userID, input.HouseholdID, input.Scope)
	q := sqlc.New(db)
	return q.AddToShoppingList(ctx, sqlc.AddToShoppingListParams{
		ShoppingItemID: input.ItemID,
		Quantity:       input.Quantity,
		HouseholdID:    hid,
		UserID:         uid,
	})
}

func getMealPlan(ctx context.Context, db *pgxpool.Pool, userID, householdID int32) ([]sqlc.MealPlanRow, error) {
	q := sqlc.New(db)
	rows, err := q.GetMealPlan(ctx, listParams(userID, householdID))
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return []sqlc.MealPlanRow{}, nil
	}
	return rows, nil
}

func getMealPlanUpdatedAt(ctx context.Context, db *pgxpool.Pool, userID, householdID int32) (map[string]any, error) {
	q := sqlc.New(db)
	t, err := q.GetMealPlanUpdatedAt(ctx, listParams(userID, householdID))
	if err != nil {
		return nil, err
	}
	result := map[string]any{"last_updated": nil}
	if t.Valid {
		result["last_updated"] = t.Time.UTC().Format("2006-01-02T15:04:05Z")
	}
	return result, nil
}

func upsertMealPlanDay(ctx context.Context, db *pgxpool.Pool, userID int32, input UpsertMealPlanInput) (sqlc.MealPlanRow, error) {
	hid, uid := scopeParams(userID, input.HouseholdID, input.Scope)
	q := sqlc.New(db)
	return q.UpsertMealPlanDay(ctx, sqlc.UpsertMealPlanDayParams{
		DayName:     input.DayName,
		MealName:    input.MealName,
		HouseholdID: hid,
		UserID:      uid,
	})
}

func getItemsFromList(ctx context.Context, db *pgxpool.Pool) ([]sqlc.ListShoppingItemsRow, error) {
	q := sqlc.New(db)
	return q.ListShoppingItems(ctx)
}

func addItemToList(ctx context.Context, db *pgxpool.Pool, params sqlc.CreateShoppingItemParams) (sqlc.ShoppingItem, error) {
	q := sqlc.New(db)
	return q.CreateShoppingItem(ctx, params)
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

// ── Have-it input ─────────────────────────────────────────────────────────────

type HaveItInput struct {
	ItemID      int32  `json:"item_id"`
	Scope       string `json:"scope"`       // "personal" or "household"
	HouseholdID int32  `json:"household_id"`
}

// ── Have-it business logic ────────────────────────────────────────────────────

type HaveItResponse struct {
	ItemIDs     []int32 `json:"item_ids"`
	LastUpdated *string `json:"last_updated"`
}

func getHaveIt(ctx context.Context, db *pgxpool.Pool, userID, householdID int32) (HaveItResponse, error) {
	q := sqlc.New(db)
	rows, err := q.GetHaveIt(ctx, listParams(userID, householdID))
	if err != nil {
		return HaveItResponse{}, err
	}
	ids := make([]int32, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ShoppingItemID)
	}

	ts, err := q.GetHaveItUpdatedAt(ctx, listParams(userID, householdID))
	if err != nil {
		return HaveItResponse{}, err
	}
	var lastUpdated *string
	if ts.Valid {
		s := ts.Time.UTC().Format("2006-01-02T15:04:05Z")
		lastUpdated = &s
	}
	return HaveItResponse{ItemIDs: ids, LastUpdated: lastUpdated}, nil
}

func markHaveIt(ctx context.Context, db *pgxpool.Pool, userID int32, input HaveItInput) (map[string]any, error) {
	hid, uid := scopeParams(userID, input.HouseholdID, input.Scope)
	q := sqlc.New(db)
	_, err := q.MarkHaveIt(ctx, sqlc.MarkHaveItParams{
		ShoppingItemID: input.ItemID,
		HouseholdID:    hid,
		UserID:         uid,
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"status": "marked"}, nil
}

func unmarkHaveIt(ctx context.Context, db *pgxpool.Pool, userID int32, input HaveItInput) (map[string]any, error) {
	hid, uid := scopeParams(userID, input.HouseholdID, input.Scope)
	q := sqlc.New(db)
	err := q.UnmarkHaveIt(ctx, sqlc.UnmarkHaveItParams{
		ShoppingItemID: input.ItemID,
		HouseholdID:    hid,
		UserID:         uid,
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"status": "unmarked"}, nil
}
