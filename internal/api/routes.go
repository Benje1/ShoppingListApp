package api

import (
	"net/http"

	"weekly-shopping-app/authentication"
)

func RegisterRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	authentication.RegisterRoutes(mux)

	return mux
}
