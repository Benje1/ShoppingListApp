package shoppinglist

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"weekly-shopping-app/authentication"
	sqlc "weekly-shopping-app/database/sqlc"
	"weekly-shopping-app/internal/allergens"
	"weekly-shopping-app/internal/api/httpx"
	"weekly-shopping-app/internal/logger"

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

	// POST /shopping/items/update — edit name, category, or portions of a catalogue item
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[sqlc.UpdateShoppingItemParams]{
		Path: "/items/update", Method: "POST", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, sqlc.UpdateShoppingItemParams) (any, error) {
			return func(r *http.Request, input sqlc.UpdateShoppingItemParams) (any, error) {
				idStr := r.URL.Query().Get("item_id")
				if idStr == "" {
					return nil, fmt.Errorf("missing item_id query param")
				}
				var id int32
				if _, err := fmt.Sscan(idStr, &id); err != nil {
					return nil, fmt.Errorf("invalid item_id: %w", err)
				}
				input.ID = id
				return updateItemInList(r.Context(), db, input)
			}
		},
	})

	// GET /shopping/items/sub-ingredients?item_id= — sub-ingredients for one item
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[struct{}]{
		Path: "/items/sub-ingredients", Method: "GET", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
			return func(r *http.Request, _ struct{}) (any, error) {
				var itemID int32
				if _, err := fmt.Sscanf(r.URL.Query().Get("item_id"), "%d", &itemID); err != nil || itemID <= 0 {
					return nil, httpx.NewClientError(fmt.Errorf("valid item_id query parameter is required"))
				}
				return listSubIngredients(r.Context(), db, itemID)
			}
		},
	})

	// POST /shopping/items/sub-ingredients/add — attach a sub-ingredient to an item
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[SubIngredientInput]{
		Path: "/items/sub-ingredients/add", Method: "POST", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, SubIngredientInput) (any, error) {
			return func(r *http.Request, input SubIngredientInput) (any, error) {
				return addSubIngredient(r.Context(), db, input)
			}
		},
	})

	// POST /shopping/items/sub-ingredients/update — edit a sub-ingredient
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[SubIngredientInput]{
		Path: "/items/sub-ingredients/update", Method: "POST", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, SubIngredientInput) (any, error) {
			return func(r *http.Request, input SubIngredientInput) (any, error) {
				return updateSubIngredient(r.Context(), db, input)
			}
		},
	})

	// DELETE /shopping/items/sub-ingredients/remove?id= — remove a sub-ingredient
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[struct{}]{
		Path: "/items/sub-ingredients/remove", Method: "DELETE", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
			return func(r *http.Request, _ struct{}) (any, error) {
				var id int32
				if _, err := fmt.Sscanf(r.URL.Query().Get("id"), "%d", &id); err != nil || id <= 0 {
					return nil, httpx.NewClientError(fmt.Errorf("valid id query parameter is required"))
				}
				q := sqlc.New(db)
				if err := q.RemoveSubIngredient(r.Context(), id); err != nil {
					return nil, err
				}
				return map[string]string{"status": "removed"}, nil
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

	// GET /shopping/list
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[struct{}]{
		Path: "/list", Method: "GET", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
			return func(r *http.Request, _ struct{}) (any, error) {
				sess, err := authentication.SessionFromContext(r)
				if err != nil {
					return nil, err
				}
				return getShoppingList(r.Context(), db, sess.UserID, sess.GetAllHouseholdsID())
			}
		},
	})

	// GET /shopping/list/updated-at
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[struct{}]{
		Path: "/list/updated-at", Method: "GET", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
			return func(r *http.Request, _ struct{}) (any, error) {
				sess, err := authentication.SessionFromContext(r)
				if err != nil {
					return nil, err
				}
				return getShoppingListUpdatedAt(r.Context(), db, sess.UserID, sess.GetAllHouseholdsID())
			}
		},
	})

	// POST /shopping/list/add
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[AddToListInput]{
		Path: "/list/add", Method: "POST", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, AddToListInput) (any, error) {
			return func(r *http.Request, input AddToListInput) (any, error) {
				sess, err := authentication.SessionFromContext(r)
				if err != nil {
					return nil, err
				}
				// Validate household scope: only allow if session owns that household
				if input.Scope == "household" && !sess.HasHousehold(input.HouseholdID) {
					return nil, fmt.Errorf("not a member of that household")
				}
				return addToShoppingList(r.Context(), db, sess.UserID, input)
			}
		},
	})

	// DELETE /shopping/list/remove?id=
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[struct{}]{
		Path: "/list/remove", Method: "DELETE", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
			return func(r *http.Request, _ struct{}) (any, error) {
				var id int32
				if _, err := fmt.Sscanf(r.URL.Query().Get("id"), "%d", &id); err != nil || id <= 0 {
					return nil, fmt.Errorf("valid id query parameter is required")
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
				sess, err := authentication.SessionFromContext(r)
				if err != nil {
					return nil, err
				}
				return getMealPlan(r.Context(), db, sess.UserID, sess.FirstHouseholdID())
			}
		},
	})

	// POST /shopping/mealplan/save
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[UpsertMealPlanInput]{
		Path: "/mealplan/save", Method: "POST", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, UpsertMealPlanInput) (any, error) {
			return func(r *http.Request, input UpsertMealPlanInput) (any, error) {
				sess, err := authentication.SessionFromContext(r)
				if err != nil {
					return nil, err
				}
				if input.Scope == "household" && !sess.HasHousehold(input.HouseholdID) {
					return nil, fmt.Errorf("not a member of that household")
				}
				return upsertMealPlanDay(r.Context(), db, sess.UserID, input)
			}
		},
	})

	// GET /shopping/mealplan/updated-at
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[struct{}]{
		Path: "/mealplan/updated-at", Method: "GET", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
			return func(r *http.Request, _ struct{}) (any, error) {
				sess, err := authentication.SessionFromContext(r)
				if err != nil {
					return nil, err
				}
				return getMealPlanUpdatedAt(r.Context(), db, sess.UserID, sess.FirstHouseholdID())
			}
		},
	})

	// GET /shopping/list/have-it
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[struct{}]{
		Path: "/list/have-it", Method: "GET", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
			return func(r *http.Request, _ struct{}) (any, error) {
				sess, err := authentication.SessionFromContext(r)
				if err != nil {
					return nil, err
				}
				return getHaveIt(r.Context(), db, sess.UserID, sess.FirstHouseholdID())
			}
		},
	})

	// POST /shopping/list/have-it/mark
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[HaveItInput]{
		Path: "/list/have-it/mark", Method: "POST", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, HaveItInput) (any, error) {
			return func(r *http.Request, input HaveItInput) (any, error) {
				sess, err := authentication.SessionFromContext(r)
				if err != nil {
					return nil, err
				}
				if input.Scope == "household" && !sess.HasHousehold(input.HouseholdID) {
					return nil, fmt.Errorf("not a member of that household")
				}
				return markHaveIt(r.Context(), db, sess.UserID, input)
			}
		},
	})

	// POST /shopping/list/have-it/unmark
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[HaveItInput]{
		Path: "/list/have-it/unmark", Method: "POST", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, HaveItInput) (any, error) {
			return func(r *http.Request, input HaveItInput) (any, error) {
				sess, err := authentication.SessionFromContext(r)
				if err != nil {
					return nil, err
				}
				if input.Scope == "household" && !sess.HasHousehold(input.HouseholdID) {
					return nil, fmt.Errorf("not a member of that household")
				}
				return unmarkHaveIt(r.Context(), db, sess.UserID, input)
			}
		},
	})
}

// ── Input types ───────────────────────────────────────────────────────────────

type AddToListInput struct {
	ItemID      int32  `json:"item_id"`
	Quantity    int32  `json:"quantity"`
	Scope       string `json:"scope"`
	HouseholdID int32  `json:"household_id"`
}

type UpsertMealPlanInput struct {
	DayName     string `json:"day_name"`
	MealName    string `json:"meal_name"`
	Scope       string `json:"scope"`
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

// listParams builds a GetShoppingListParams for scalar (single-household) queries.
// When householdID is 0 the household field is left as a NULL pgtype.Int4.
func listParams(userID, householdID int32) sqlc.GetShoppingListParams {
	p := sqlc.GetShoppingListParams{
		UserID: pgtype.Int4{Int32: userID, Valid: true},
	}
	if householdID != 0 {
		p.HouseholdID = pgtype.Int4{Int32: householdID, Valid: true}
	}
	return p
}

func mealPlanParams(userID, householdID int32) sqlc.GetMealPlanParams {
	p := sqlc.GetMealPlanParams{
		UserID: pgtype.Int4{Int32: userID, Valid: true},
	}
	if householdID != 0 {
		p.HouseholdID = pgtype.Int4{Int32: householdID, Valid: true}
	}
	return p
}

func mealPlanUpdatedAtParams(userID, householdID int32) sqlc.GetMealPlanUpdatedAtParams {
	p := sqlc.GetMealPlanUpdatedAtParams{
		UserID: pgtype.Int4{Int32: userID, Valid: true},
	}
	if householdID != 0 {
		p.HouseholdID = pgtype.Int4{Int32: householdID, Valid: true}
	}
	return p
}

func haveItParams(userID, householdID int32) sqlc.GetHaveItParams {
	p := sqlc.GetHaveItParams{
		UserID: pgtype.Int4{Int32: userID, Valid: true},
	}
	if householdID != 0 {
		p.HouseholdID = pgtype.Int4{Int32: householdID, Valid: true}
	}
	return p
}

func haveItUpdatedAtParams(userID, householdID int32) sqlc.GetHaveItUpdatedAtParams {
	p := sqlc.GetHaveItUpdatedAtParams{
		UserID: pgtype.Int4{Int32: userID, Valid: true},
	}
	if householdID != 0 {
		p.HouseholdID = pgtype.Int4{Int32: householdID, Valid: true}
	}
	return p
}

func getShoppingList(ctx context.Context, db *pgxpool.Pool, userID int32, householdIDs []int32) ([]sqlc.GetShoppingListRow, error) {
	// GetShoppingList uses ANY($1) in SQL which requires an array, not a scalar pgtype.Int4.
	// We query directly to pass []int32 correctly via pgx.
	const query = `SELECT
    sl.id,
    si.id   AS item_id,
    si.name,
    si.item_type,
    si.portions_per_unit,
    sl.quantity,
    sl.updated_at,
    CASE WHEN sl.household_id IS NOT NULL THEN 'household' ELSE 'personal' END AS scope
FROM shopping_list sl
JOIN shopping_items si ON sl.shopping_item_id = si.id
WHERE sl.household_id = ANY($1) OR sl.user_id = $2
ORDER BY si.item_type, si.name`

	if len(householdIDs) == 0 {
		householdIDs = []int32{}
	}
	rows, err := db.Query(ctx, query, householdIDs, userID)
	if err != nil {
		return nil, logger.WithStack(err)
	}
	defer rows.Close()
	var items []sqlc.GetShoppingListRow
	for rows.Next() {
		var i sqlc.GetShoppingListRow
		if err := rows.Scan(&i.ID, &i.ItemID, &i.Name, &i.ItemType, &i.PortionsPerUnit, &i.Quantity, &i.UpdatedAt, &i.Scope); err != nil {
			return nil, logger.WithStack(err)
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, logger.WithStack(err)
	}
	if items == nil {
		return []sqlc.GetShoppingListRow{}, nil
	}
	return items, nil
}

func getShoppingListUpdatedAt(ctx context.Context, db *pgxpool.Pool, userID int32, householdIDs []int32) (map[string]any, error) {
	const query = `SELECT MAX(updated_at) AS last_updated FROM shopping_list WHERE household_id = ANY($1) OR user_id = $2`
	if len(householdIDs) == 0 {
		householdIDs = []int32{}
	}
	var raw interface{}
	if err := db.QueryRow(ctx, query, householdIDs, userID).Scan(&raw); err != nil {
		return nil, logger.WithStack(err)
	}
	result := map[string]any{"last_updated": nil}
	if t, ok := raw.(pgtype.Timestamp); ok && t.Valid {
		result["last_updated"] = t.Time.UTC().Format("2006-01-02T15:04:05Z")
	}
	return result, nil
}

func addToShoppingList(ctx context.Context, db *pgxpool.Pool, userID int32, input AddToListInput) (sqlc.ShoppingList, error) {
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

func getMealPlan(ctx context.Context, db *pgxpool.Pool, userID, householdID int32) ([]sqlc.GetMealPlanRow, error) {
	q := sqlc.New(db)
	rows, err := q.GetMealPlan(ctx, mealPlanParams(userID, householdID))
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return []sqlc.GetMealPlanRow{}, nil
	}
	return rows, nil
}

func getMealPlanUpdatedAt(ctx context.Context, db *pgxpool.Pool, userID, householdID int32) (map[string]any, error) {
	q := sqlc.New(db)
	raw, err := q.GetMealPlanUpdatedAt(ctx, mealPlanUpdatedAtParams(userID, householdID))
	if err != nil {
		return nil, err
	}
	result := map[string]any{"last_updated": nil}
	if t, ok := raw.(pgtype.Timestamp); ok && t.Valid {
		result["last_updated"] = t.Time.UTC().Format("2006-01-02T15:04:05Z")
	}
	return result, nil
}

func upsertMealPlanDay(ctx context.Context, db *pgxpool.Pool, userID int32, input UpsertMealPlanInput) (sqlc.MealPlan, error) {
	hid, uid := scopeParams(userID, input.HouseholdID, input.Scope)
	q := sqlc.New(db)
	mealName := pgtype.Text{Valid: false}
	if input.MealName != "" {
		mealName = pgtype.Text{String: input.MealName, Valid: true}
	}
	if hid.Valid {
		return q.UpsertMealPlanDayHousehold(ctx, sqlc.UpsertMealPlanDayHouseholdParams{
			DayName:     input.DayName,
			MealName:    mealName,
			HouseholdID: hid,
		})
	}
	return q.UpsertMealPlanDayUser(ctx, sqlc.UpsertMealPlanDayUserParams{
		DayName:  input.DayName,
		MealName: mealName,
		UserID:   uid,
	})
}

func getItemsFromList(ctx context.Context, db *pgxpool.Pool) ([]sqlc.ListShoppingItemsRow, error) {
	q := sqlc.New(db)
	return q.ListShoppingItems(ctx)
}

func addItemToList(ctx context.Context, db *pgxpool.Pool, params sqlc.CreateShoppingItemParams) (sqlc.ShoppingItem, error) {
	clean, err := allergens.Sanitize(params.Allergens)
	if err != nil {
		return sqlc.ShoppingItem{}, httpx.NewClientError(err)
	}
	params.Allergens = clean
	q := sqlc.New(db)
	return q.CreateShoppingItem(ctx, params)
}

func updateItemInList(ctx context.Context, db *pgxpool.Pool, params sqlc.UpdateShoppingItemParams) (sqlc.ShoppingItem, error) {
	// Partial update: a nil slice means "leave allergens unchanged", so only
	// validate when the caller actually supplied a (possibly empty) list.
	if params.Allergens != nil {
		clean, err := allergens.Sanitize(params.Allergens)
		if err != nil {
			return sqlc.ShoppingItem{}, httpx.NewClientError(err)
		}
		params.Allergens = clean
	}
	q := sqlc.New(db)
	return q.UpdateShoppingItem(ctx, params)
}

// ── Sub-ingredients ─────────────────────────────────────────────────────────

// SubIngredientInput carries the payload for creating or updating a
// sub-ingredient. ID is only used on update; ShoppingItemID only on create.
type SubIngredientInput struct {
	ID             int32    `json:"id"`
	ShoppingItemID int32    `json:"shopping_item_id"`
	Name           string   `json:"name"`
	Allergens      []string `json:"allergens"`
	SortOrder      int32    `json:"sort_order"`
}

func listSubIngredients(ctx context.Context, db *pgxpool.Pool, itemID int32) ([]sqlc.SubIngredient, error) {
	q := sqlc.New(db)
	rows, err := q.ListSubIngredientsForItem(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return []sqlc.SubIngredient{}, nil
	}
	return rows, nil
}

func addSubIngredient(ctx context.Context, db *pgxpool.Pool, input SubIngredientInput) (sqlc.SubIngredient, error) {
	if input.ShoppingItemID <= 0 {
		return sqlc.SubIngredient{}, httpx.NewClientError(fmt.Errorf("shopping_item_id is required"))
	}
	if strings.TrimSpace(input.Name) == "" {
		return sqlc.SubIngredient{}, httpx.NewClientError(fmt.Errorf("name is required"))
	}
	clean, err := allergens.Sanitize(input.Allergens)
	if err != nil {
		return sqlc.SubIngredient{}, httpx.NewClientError(err)
	}
	q := sqlc.New(db)
	return q.AddSubIngredient(ctx, sqlc.AddSubIngredientParams{
		ShoppingItemID: input.ShoppingItemID,
		Name:           strings.TrimSpace(input.Name),
		Allergens:      clean,
		SortOrder:      input.SortOrder,
	})
}

func updateSubIngredient(ctx context.Context, db *pgxpool.Pool, input SubIngredientInput) (sqlc.SubIngredient, error) {
	if input.ID <= 0 {
		return sqlc.SubIngredient{}, httpx.NewClientError(fmt.Errorf("id is required"))
	}
	if strings.TrimSpace(input.Name) == "" {
		return sqlc.SubIngredient{}, httpx.NewClientError(fmt.Errorf("name is required"))
	}
	clean, err := allergens.Sanitize(input.Allergens)
	if err != nil {
		return sqlc.SubIngredient{}, httpx.NewClientError(err)
	}
	q := sqlc.New(db)
	return q.UpdateSubIngredient(ctx, sqlc.UpdateSubIngredientParams{
		ID:        input.ID,
		Name:      strings.TrimSpace(input.Name),
		Allergens: clean,
		SortOrder: input.SortOrder,
	})
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

// ── Have-it ───────────────────────────────────────────────────────────────────

type HaveItInput struct {
	ItemID      int32  `json:"item_id"`
	Scope       string `json:"scope"`
	HouseholdID int32  `json:"household_id"`
}

type HaveItResponse struct {
	ItemIDs     []int32 `json:"item_ids"`
	LastUpdated *string `json:"last_updated"`
}

func getHaveIt(ctx context.Context, db *pgxpool.Pool, userID, householdID int32) (HaveItResponse, error) {
	q := sqlc.New(db)
	rows, err := q.GetHaveIt(ctx, haveItParams(userID, householdID))
	if err != nil {
		return HaveItResponse{}, logger.WithStack(err)
	}
	ids := make([]int32, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ShoppingItemID)
	}
	ts, err := q.GetHaveItUpdatedAt(ctx, haveItUpdatedAtParams(userID, householdID))
	if err != nil {
		return HaveItResponse{}, logger.WithStack(err)
	}
	var lastUpdated *string
	if t, ok := ts.(pgtype.Timestamp); ok && t.Valid {
		s := t.Time.UTC().Format("2006-01-02T15:04:05Z")
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
		return nil, logger.WithStack(err)
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
		return nil, logger.WithStack(err)
	}
	return map[string]any{"status": "unmarked"}, nil
}
