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

	// Create a new household — the caller is automatically added as the first member
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[CreateHouseholdInput]{
		Path: "/create", Method: "POST", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, CreateHouseholdInput) (any, error) {
			return func(r *http.Request, input CreateHouseholdInput) (any, error) {
				userID, err := authentication.GetUserID(r)
				if err != nil {
					return nil, err
				}
				return createHousehold(r.Context(), db, userID, input)
			}
		},
	})

	// Get basic household info
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[struct{}]{
		Path: "/get", Method: "GET", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
			return func(r *http.Request, _ struct{}) (any, error) {
				id, err := queryID(r)
				if err != nil {
					return nil, err
				}
				return getHousehold(r.Context(), db, id)
			}
		},
	})

	// Get full household detail: members + pending invites
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[struct{}]{
		Path: "/detail", Method: "GET", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
			return func(r *http.Request, _ struct{}) (any, error) {
				id, err := queryID(r)
				if err != nil {
					return nil, err
				}
				return getHouseholdDetail(r.Context(), db, id)
			}
		},
	})

	// Rename a household
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[RenameHouseholdInput]{
		Path: "/rename", Method: "POST", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, RenameHouseholdInput) (any, error) {
			return func(r *http.Request, input RenameHouseholdInput) (any, error) {
				id, err := queryID(r)
				if err != nil {
					return nil, err
				}
				return renameHousehold(r.Context(), db, id, input)
			}
		},
	})

	// Delete a household
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[struct{}]{
		Path: "/delete", Method: "DELETE", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
			return func(r *http.Request, _ struct{}) (any, error) {
				id, err := queryID(r)
				if err != nil {
					return nil, err
				}
				if err := deleteHousehold(r.Context(), db, id); err != nil {
					return nil, err
				}
				return map[string]string{"status": "household deleted"}, nil
			}
		},
	})

	// Generate a shareable invite code for a household (called by existing member)
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[struct{}]{
		Path: "/invite/generate", Method: "POST", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, struct{}) (any, error) {
			return func(r *http.Request, _ struct{}) (any, error) {
				id, err := queryID(r)
				if err != nil {
					return nil, err
				}
				userID, err := authentication.GetUserID(r)
				if err != nil {
					return nil, err
				}
				return generateInviteCode(r.Context(), db, id, userID)
			}
		},
	})

	// Submit a code to request joining a household (called by the new user)
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[RequestJoinInput]{
		Path: "/invite/request", Method: "POST", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, RequestJoinInput) (any, error) {
			return func(r *http.Request, input RequestJoinInput) (any, error) {
				userID, err := authentication.GetUserID(r)
				if err != nil {
					return nil, err
				}
				return requestJoin(r.Context(), db, userID, input)
			}
		},
	})

	// Approve or deny a pending invite (called by existing household member)
	httpx.RegisterEndpoint(r, httpx.EndpointConfig[RespondToInviteInput]{
		Path: "/invite/respond", Method: "POST", Public: false,
		Handler: func(db *pgxpool.Pool) func(*http.Request, RespondToInviteInput) (any, error) {
			return func(r *http.Request, input RespondToInviteInput) (any, error) {
				return respondToInvite(r.Context(), db, input)
			}
		},
	})
}

func queryID(r *http.Request) (int32, error) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(id), nil
}
