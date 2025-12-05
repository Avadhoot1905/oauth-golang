package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"oauth-golang/internal/config"
	"oauth-golang/internal/oauth"
	"oauth-golang/internal/user"
)

// TokenHandler handles OAuth 2.0 token requests
// API INPUT: Receives token exchange requests from clients
type TokenHandler struct {
	config          *config.Config
	tokenService    *oauth.TokenService
	authCodeService *oauth.AuthCodeService
	clientRegistry  *oauth.ClientRegistry
	pkceValidator   *oauth.PKCEValidator
	userAuth        *user.AuthService
}

func NewTokenHandler(
	cfg *config.Config,
	tokenService *oauth.TokenService,
	authCodeService *oauth.AuthCodeService,
	clientRegistry *oauth.ClientRegistry,
	pkceValidator *oauth.PKCEValidator,
	userAuth *user.AuthService,
) *TokenHandler {
	return &TokenHandler{
		config:          cfg,
		tokenService:    tokenService,
		authCodeService: authCodeService,
		clientRegistry:  clientRegistry,
		pkceValidator:   pkceValidator,
		userAuth:        userAuth,
	}
}

// TokenRequest represents the token exchange request
// API INPUT: Request body from client
type TokenRequest struct {
	GrantType    string `json:"grant_type"`
	Code         string `json:"code"`
	RedirectURI  string `json:"redirect_uri"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	CodeVerifier string `json:"code_verifier"`
	RefreshToken string `json:"refresh_token"`
}

// TokenResponse represents the token exchange response
// API OUTPUT: Response to client
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
}

// Handle processes the /token endpoint
// API INPUT: Form data or JSON body with grant_type, code, client credentials
// OUTPUT TO DB: Stores access token and refresh token via tokenService
func (h *TokenHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, "invalid_request", "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request (API INPUT)
	var req TokenRequest
	if r.Header.Get("Content-Type") == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.writeError(w, "invalid_request", "Invalid JSON body", http.StatusBadRequest)
			return
		}
	} else {
		// Parse form data
		if err := r.ParseForm(); err != nil {
			h.writeError(w, "invalid_request", "Invalid form data", http.StatusBadRequest)
			return
		}
		req.GrantType = r.FormValue("grant_type")
		req.Code = r.FormValue("code")
		req.RedirectURI = r.FormValue("redirect_uri")
		req.ClientID = r.FormValue("client_id")
		req.ClientSecret = r.FormValue("client_secret")
		req.CodeVerifier = r.FormValue("code_verifier")
		req.RefreshToken = r.FormValue("refresh_token")
	}

	// Validate grant type
	switch req.GrantType {
	case "authorization_code":
		h.handleAuthorizationCodeGrant(w, r, &req)
	case "refresh_token":
		h.handleRefreshTokenGrant(w, r, &req)
	default:
		h.writeError(w, "unsupported_grant_type", "Grant type not supported", http.StatusBadRequest)
	}
}

// handleAuthorizationCodeGrant handles the authorization_code grant type
func (h *TokenHandler) handleAuthorizationCodeGrant(w http.ResponseWriter, r *http.Request, req *TokenRequest) {
	// Validate required parameters
	if req.Code == "" || req.RedirectURI == "" || req.ClientID == "" {
		h.writeError(w, "invalid_request", "Missing required parameters", http.StatusBadRequest)
		return
	}

	// Validate client credentials (DB interaction via clientRegistry)
	client, err := h.clientRegistry.GetClient(req.ClientID)
	if err != nil || client == nil {
		h.writeError(w, "invalid_client", "Invalid client credentials", http.StatusUnauthorized)
		return
	}

	// For confidential clients, validate client secret
	if client.IsConfidential() && client.ClientSecret != req.ClientSecret {
		h.writeError(w, "invalid_client", "Invalid client credentials", http.StatusUnauthorized)
		return
	}

	// Retrieve and validate authorization code (DB interaction via authCodeService)
	authCode := h.authCodeService.GetAuthCode(req.Code)
	if authCode == nil {
		h.writeError(w, "invalid_grant", "Invalid authorization code", http.StatusBadRequest)
		return
	}

	// Validate authorization code hasn't expired
	if time.Now().After(authCode.ExpiresAt) {
		h.authCodeService.DeleteAuthCode(req.Code)
		h.writeError(w, "invalid_grant", "Authorization code expired", http.StatusBadRequest)
		return
	}

	// Validate client ID matches
	if authCode.ClientID != req.ClientID {
		h.writeError(w, "invalid_grant", "Client ID mismatch", http.StatusBadRequest)
		return
	}

	// Validate redirect URI matches
	if authCode.RedirectURI != req.RedirectURI {
		h.writeError(w, "invalid_grant", "Redirect URI mismatch", http.StatusBadRequest)
		return
	}

	// Validate PKCE if code_challenge was used
	if authCode.CodeChallenge != "" {
		if req.CodeVerifier == "" {
			h.writeError(w, "invalid_request", "code_verifier required", http.StatusBadRequest)
			return
		}
		if !verifyCodeChallenge(req.CodeVerifier, authCode.CodeChallenge, authCode.CodeChallengeMethod) {
			h.writeError(w, "invalid_grant", "Invalid code_verifier", http.StatusBadRequest)
			return
		}
	}

	// Create or update user in database (OUTPUT TO DB via userAuth)
	user, err := h.userAuth.CreateOrUpdateUser(authCode.UserID, authCode.UserInfo)
	if err != nil {
		h.writeError(w, "server_error", "Failed to create user", http.StatusInternalServerError)
		return
	}

	// Generate tokens (OUTPUT TO DB via tokenService)
	tokens, err := h.tokenService.GenerateTokens(user, authCode.Scope)
	if err != nil {
		h.writeError(w, "server_error", "Failed to generate tokens", http.StatusInternalServerError)
		return
	}

	// Delete authorization code (one-time use)
	h.authCodeService.DeleteAuthCode(req.Code)

	// Return token response (API OUTPUT)
	h.writeTokenResponse(w, tokens)
}

// handleRefreshTokenGrant handles the refresh_token grant type
func (h *TokenHandler) handleRefreshTokenGrant(w http.ResponseWriter, r *http.Request, req *TokenRequest) {
	// Validate required parameters
	if req.RefreshToken == "" || req.ClientID == "" {
		h.writeError(w, "invalid_request", "Missing required parameters", http.StatusBadRequest)
		return
	}

	// Validate client credentials (DB interaction via clientRegistry)
	client, err := h.clientRegistry.GetClient(req.ClientID)
	if err != nil || client == nil {
		h.writeError(w, "invalid_client", "Invalid client credentials", http.StatusUnauthorized)
		return
	}

	// For confidential clients, validate client secret
	if client.IsConfidential() && client.ClientSecret != req.ClientSecret {
		h.writeError(w, "invalid_client", "Invalid client credentials", http.StatusUnauthorized)
		return
	}

	// Refresh tokens (DB interaction via tokenService)
	tokens, err := h.tokenService.RefreshTokens(req.RefreshToken, req.ClientID)
	if err != nil {
		h.writeError(w, "invalid_grant", err.Error(), http.StatusBadRequest)
		return
	}

	// Return token response (API OUTPUT)
	h.writeTokenResponse(w, tokens)
}

// writeTokenResponse writes the token response to the client
func (h *TokenHandler) writeTokenResponse(w http.ResponseWriter, tokens *oauth.TokenPair) {
	response := TokenResponse{
		AccessToken:  tokens.AccessToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(tokens.ExpiresIn.Seconds()),
		RefreshToken: tokens.RefreshToken,
		IDToken:      tokens.IDToken,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	json.NewEncoder(w).Encode(response)
}

// writeError writes an OAuth error response
func (h *TokenHandler) writeError(w http.ResponseWriter, errorCode, description string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error":             errorCode,
		"error_description": description,
	})
}
