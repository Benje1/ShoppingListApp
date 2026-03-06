package httpx

import (
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// EndpointConfig holds all configuration for a single route registration.
type EndpointConfig[T any] struct {
	// Path is the sub-path for this endpoint (e.g. "/create").
	// The full registered path will be the router's prefix + Path.
	Path string

	// Method is the HTTP method ("GET", "POST", etc.).
	// Defaults to http.MethodPost when empty.
	Method string

	// Public marks the endpoint as accessible without authentication.
	// Defaults to false — omitting this field means the route is protected.
	// The Router's authMiddleware is injected automatically for protected routes,
	// so there is no separate list of public paths to keep in sync.
	Public bool

	// DB overrides the router-level pool for this specific endpoint.
	// Leave nil to use the router's pool.
	DB *pgxpool.Pool

	// Handler is a factory that receives a DB pool and returns the typed
	// handler. Use this for handlers that don't need to write to the response
	// directly (no cookies, no custom headers).
	// Exactly one of Handler or HandlerWithWriter must be non-nil.
	Handler func(db *pgxpool.Pool) func(*http.Request, T) (any, error)

	// HandlerWithWriter is like Handler but also receives the ResponseWriter,
	// needed when the handler must set cookies or custom headers (e.g. /login).
	HandlerWithWriter func(db *pgxpool.Pool) func(http.ResponseWriter, *http.Request, T) (any, error)

	// Middleware is an optional slice of additional per-route middleware
	// applied after auth (e.g. rate-limiting, role checks).
	// Auth middleware must not be added here — use Public: false instead.
	Middleware []func(http.Handler) http.Handler
}

// Router groups endpoints under a common URL prefix and shares a mux, db pool,
// wrap function, and auth middleware so individual RegisterEndpoint calls stay concise.
//
// Example:
//
//	r := httpx.NewRouter(mux, db, wrap, authentication.RequireAuth, "/users")
//	httpx.RegisterEndpoint(r, httpx.EndpointConfig[UserInput]{
//	    Path:    "/create",
//	    Public:  true,
//	    Handler: createUserPost,
//	})
type Router struct {
	mux            *http.ServeMux
	db             *pgxpool.Pool
	wrap           func(AppHandler) http.HandlerFunc
	prefix         string
	authMiddleware func(http.Handler) http.Handler
}

// NewRouter creates a Router scoped to the given URL prefix (e.g. "/users").
// Pass an empty prefix for top-level routes such as "/login".
//
// authMiddleware is the function applied to any route with Public: false.
// Passing nil disables automatic auth injection (useful in tests).
func NewRouter(
	mux *http.ServeMux,
	db *pgxpool.Pool,
	wrap func(AppHandler) http.HandlerFunc,
	authMiddleware func(http.Handler) http.Handler,
	prefix string,
) *Router {
	return &Router{
		mux:            mux,
		db:             db,
		wrap:           wrap,
		prefix:         strings.TrimRight(prefix, "/"),
		authMiddleware: authMiddleware,
	}
}

// RegisterEndpoint wires a typed endpoint into the router's mux.
//
// Type parameter T is the request-body struct (use struct{} for GET/DELETE).
//
// Auth behaviour:
//   - Public: false (default) → the router's authMiddleware is applied automatically.
//   - Public: true            → no auth middleware; the route is open.
func RegisterEndpoint[T any](r *Router, cfg EndpointConfig[T]) {
	fullPath := r.prefix + cfg.Path

	method := cfg.Method
	if method == "" {
		method = http.MethodPost
	}

	db := cfg.DB
	if db == nil {
		db = r.db
	}

	// Build the core AppHandler from whichever factory was provided.
	var appHandler AppHandler
	switch {
	case cfg.HandlerWithWriter != nil:
		appHandler = EndpointWithWriter(method, cfg.HandlerWithWriter(db))
	case cfg.Handler != nil:
		appHandler = Endpoint(method, cfg.Handler(db))
	default:
		panic("httpx.RegisterEndpoint: one of Handler or HandlerWithWriter must be set for " + fullPath)
	}

	// Wrap into a standard http.Handler.
	var h http.Handler = r.wrap(appHandler)

	// Apply auth first (innermost), then any extra per-route middleware.
	if !cfg.Public && r.authMiddleware != nil {
		h = r.authMiddleware(h)
	}

	for i := len(cfg.Middleware) - 1; i >= 0; i-- {
		h = cfg.Middleware[i](h)
	}

	r.mux.Handle(fullPath, h)
}
