package http

import (
	"net/http"

	"oauth-golang/internal/config"
	"oauth-golang/internal/http/handlers"
	"oauth-golang/internal/oauth"
	"oauth-golang/internal/security"
	"oauth-golang/internal/storage"
	"oauth-golang/internal/user"
)

// NewRouter creates and configures the HTTP router with all endpoints
// This is the main API input layer - handles all HTTP requests
func NewRouter(
	cfg *config.Config,
	userRepo *storage.UserRepository,
	clientRepo *storage.ClientRepository,
	tokenRepo *storage.TokenRepository,
) http.Handler {
	mux := http.NewServeMux()

	// Initialize security components
	jwtService := security.NewJWTService(cfg.JWTSecret)

	// Initialize OAuth components (handles Google OAuth provider interaction)
	authCodeService := oauth.NewAuthCodeService()
	tokenService := oauth.NewTokenService(cfg, jwtService, tokenRepo)
	clientRegistry := oauth.NewClientRegistry(clientRepo)
	pkceValidator := oauth.NewPKCEValidator()

	// Initialize user authentication service
	userAuth := user.NewAuthService(userRepo)

	// Initialize handlers (API input/output layer)
	authorizeHandler := handlers.NewAuthorizeHandler(
		cfg,
		clientRegistry,
		authCodeService,
		pkceValidator,
	)
	tokenHandler := handlers.NewTokenHandler(
		cfg,
		tokenService,
		authCodeService,
		clientRegistry,
		pkceValidator,
		userAuth,
	)
	userinfoHandler := handlers.NewUserInfoHandler(jwtService, userRepo)
	introspectHandler := handlers.NewIntrospectHandler(jwtService, tokenRepo)

	// OAuth 2.0 endpoints - API input layer
	// /authorize - Initiates OAuth flow, redirects to Google
	mux.HandleFunc("/authorize", authorizeHandler.Handle)

	// /callback - Receives authorization code from Google (Google OAuth provider interaction)
	mux.HandleFunc("/callback", authorizeHandler.HandleCallback)

	// /token - Exchanges authorization code for JWT tokens (output to DB via tokenRepo)
	mux.HandleFunc("/token", tokenHandler.Handle)

	// /userinfo - Returns user information from JWT token (input from API, output from DB)
	mux.HandleFunc("/userinfo", userinfoHandler.Handle)

	// /introspect - Validates tokens for other microservices
	mux.HandleFunc("/introspect", introspectHandler.Handle)

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	})

	// Apply middleware
	return loggingMiddleware(corsMiddleware(mux))
}

// loggingMiddleware logs all HTTP requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// log.Printf("%s %s %s", r.Method, r.RequestURI, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}

// corsMiddleware adds CORS headers
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
