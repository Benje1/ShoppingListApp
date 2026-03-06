package households

import (
	"net/http"
	"strconv"

	"weekly-shopping-app/authentication"
	"weekly-shopping-app/internal/api/httpx"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterHouseholdRoutes(mux *http.ServeMux, db *pgxpool.Pool, wrap func(httpx.AppHandler) http.HandlerFunc) {
	r := httpx.NewRouter(mux, db, wrap, authentication.RequireAuth, "/households")

	httpx.RegisterEndpoint(r, httpx.EndpointConfig[HouseholdInput]{
		Path:   "/create",
		Method: "POST",
		Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, HouseholdInput) (any, error) {
			return createHouseholdPost(db)
		},
	})

	httpx.RegisterEndpoint(r, httpx.EndpointConfig[struct{}]{
		Path:   "/get",
		Method: "GET",
		Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
			return getHouseholdGet(db)
		},
	})

	httpx.RegisterEndpoint(r, httpx.EndpointConfig[struct{}]{
		Path:   "/delete",
		Method: "DELETE",
		Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
			return deleteHouseholdDelete(db)
		},
	})
}

func createHouseholdPost(db *pgxpool.Pool) func(*http.Request, HouseholdInput) (any, error) {
	return func(r *http.Request, input HouseholdInput) (any, error) {
		household, err := createHousehold(r.Context(), db, input)
		if err != nil {
			return nil, err
		}
		return household, nil
	}
}

func getHouseholdGet(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
	return func(r *http.Request, _ struct{}) (any, error) {
		idStr := r.URL.Query().Get("id")
		id, err := strconv.ParseInt(idStr, 10, 32)
		if err != nil {
			return nil, err
		}
		household, err := getHousehold(r.Context(), db, int32(id))
		if err != nil {
			return nil, err
		}
		return household, nil
	}
}

func deleteHouseholdDelete(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
	return func(r *http.Request, _ struct{}) (any, error) {
		idStr := r.URL.Query().Get("id")
		id, err := strconv.ParseInt(idStr, 10, 32)
		if err != nil {
			return nil, err
		}
		if err := deleteHousehold(r.Context(), db, int32(id)); err != nil {
			return nil, err
		}
		return map[string]string{"status": "household deleted"}, nil
	}
}
