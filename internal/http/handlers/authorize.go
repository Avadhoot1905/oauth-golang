package handlers

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"oauth-golang/internal/config"
	"oauth-golang/internal/models"
	"oauth-golang/internal/oauth"
	"oauth-golang/pkg/utils"
)

// AuthorizeHandler handles OAuth 2.0 authorization requests
// API INPUT: Receives authorization requests from clients
type AuthorizeHandler struct {
	config          *config.Config
	clientRegistry  *oauth.ClientRegistry
	authCodeService *oauth.AuthCodeService
	pkceValidator   *oauth.PKCEValidator
}

func NewAuthorizeHandler(
	cfg *config.Config,
	clientRegistry *oauth.ClientRegistry,
	authCodeService *oauth.AuthCodeService,
	pkceValidator *oauth.PKCEValidator,
) *AuthorizeHandler {
	return &AuthorizeHandler{
		config:          cfg,
		clientRegistry:  clientRegistry,
		authCodeService: authCodeService,
		pkceValidator:   pkceValidator,
	}
}

// Handle processes the /authorize endpoint
// API INPUT: Query params (client_id, redirect_uri, response_type, state, code_challenge, code_challenge_method)
// OUTPUT: Redirects user to Google OAuth provider
func (h *AuthorizeHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters (API INPUT)
	clientID := r.URL.Query().Get("client_id")
	redirectURI := r.URL.Query().Get("redirect_uri")
	responseType := r.URL.Query().Get("response_type")
	state := r.URL.Query().Get("state")
	codeChallenge := r.URL.Query().Get("code_challenge")
	codeChallengeMethod := r.URL.Query().Get("code_challenge_method")
	scope := r.URL.Query().Get("scope")

	// Validate request parameters
	if clientID == "" || redirectURI == "" || responseType != "code" {
		http.Error(w, "Invalid request parameters", http.StatusBadRequest)
		return
	}

	// Validate client (DB interaction via clientRegistry)
	client, err := h.clientRegistry.GetClient(clientID)
	if err != nil || client == nil {
		http.Error(w, "Invalid client_id", http.StatusBadRequest)
		return
	}

	// Validate redirect URI
	if !client.ValidateRedirectURI(redirectURI) {
		http.Error(w, "Invalid redirect_uri", http.StatusBadRequest)
		return
	}

	// Validate PKCE parameters
	if codeChallenge != "" {
		if !h.pkceValidator.ValidateCodeChallenge(codeChallenge, codeChallengeMethod) {
			http.Error(w, "Invalid PKCE parameters", http.StatusBadRequest)
			return
		}
	}

	// Generate state for CSRF protection if not provided
	if state == "" {
		state = utils.GenerateRandomString(32)
	}

	// Store PKCE challenge and redirect info temporarily
	// In production, use Redis or database with TTL
	sessionID := utils.GenerateRandomString(32)
	h.authCodeService.StoreSession(sessionID, &oauth.AuthSession{
		ClientID:            clientID,
		RedirectURI:         redirectURI,
		State:               state,
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: codeChallengeMethod,
		Scope:               scope,
		CreatedAt:           time.Now(),
	})

	// Build Google OAuth URL (INTERACTION WITH GOOGLE OAUTH PROVIDER)
	googleAuthURL := h.buildGoogleAuthURL(sessionID, scope)

	// Redirect user to Google login
	http.Redirect(w, r, googleAuthURL, http.StatusFound)
}

// HandleCallback processes the callback from Google OAuth
// GOOGLE OAUTH PROVIDER INTERACTION: Receives authorization code from Google
// OUTPUT TO DB: Stores authorization code and user info
func (h *AuthorizeHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get authorization code and state from Google (GOOGLE OAUTH PROVIDER OUTPUT)
	code := r.URL.Query().Get("code")
	stateParam := r.URL.Query().Get("state")
	errorParam := r.URL.Query().Get("error")

	if errorParam != "" {
		http.Error(w, fmt.Sprintf("OAuth error: %s", errorParam), http.StatusBadRequest)
		return
	}

	if code == "" {
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	// Retrieve session by state parameter
	session := h.authCodeService.GetSession(stateParam)
	if session == nil {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	// Exchange code for access token with Google (GOOGLE OAUTH PROVIDER INTERACTION)
	googleToken, err := h.exchangeGoogleCode(code)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to exchange code: %v", err), http.StatusInternalServerError)
		return
	}

	// Get user info from Google (GOOGLE OAUTH PROVIDER INTERACTION)
	userInfo, err := h.getGoogleUserInfo(googleToken.AccessToken)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get user info: %v", err), http.StatusInternalServerError)
		return
	}

	// Generate authorization code for client
	authCode := utils.GenerateRandomString(32)

	// Store authorization code with user info (OUTPUT TO DB via authCodeService)
	h.authCodeService.StoreAuthCode(authCode, &oauth.AuthCode{
		Code:                authCode,
		ClientID:            session.ClientID,
		RedirectURI:         session.RedirectURI,
		UserID:              userInfo.Email, // Using email as user ID
		CodeChallenge:       session.CodeChallenge,
		CodeChallengeMethod: session.CodeChallengeMethod,
		Scope:               session.Scope,
		ExpiresAt:           time.Now().Add(10 * time.Minute),
		UserInfo:            userInfo,
	})

	// Build redirect URL with authorization code
	redirectURL, _ := url.Parse(session.RedirectURI)
	q := redirectURL.Query()
	q.Set("code", authCode)
	q.Set("state", session.State)
	redirectURL.RawQuery = q.Encode()

	// Redirect back to client application
	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}

// buildGoogleAuthURL constructs the Google OAuth authorization URL
func (h *AuthorizeHandler) buildGoogleAuthURL(state, scope string) string {
	if scope == "" {
		scope = "openid email profile"
	}

	params := url.Values{}
	params.Set("client_id", h.config.GoogleClientID)
	params.Set("redirect_uri", h.config.GoogleRedirectURL)
	params.Set("response_type", "code")
	params.Set("scope", scope)
	params.Set("state", state)
	params.Set("access_type", "offline")
	params.Set("prompt", "consent")

	return "https://accounts.google.com/o/oauth2/v2/auth?" + params.Encode()
}

// GoogleTokenResponse represents the response from Google's token endpoint
type GoogleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	IDToken      string `json:"id_token"`
}

// exchangeGoogleCode exchanges authorization code for access token with Google
func (h *AuthorizeHandler) exchangeGoogleCode(code string) (*GoogleTokenResponse, error) {
	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", h.config.GoogleClientID)
	data.Set("client_secret", h.config.GoogleClientSecret)
	data.Set("redirect_uri", h.config.GoogleRedirectURL)
	data.Set("grant_type", "authorization_code")

	resp, err := http.PostForm("https://oauth2.googleapis.com/token", data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Google token exchange failed: %s", string(body))
	}

	var tokenResp GoogleTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

// getGoogleUserInfo retrieves user information from Google
func (h *AuthorizeHandler) getGoogleUserInfo(accessToken string) (*models.GoogleUserInfo, error) {
	req, err := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Failed to get user info: %s", string(body))
	}

	var userInfo models.GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

// Helper function to verify code_verifier against code_challenge (for PKCE)
func verifyCodeChallenge(codeVerifier, codeChallenge, method string) bool {
	if method == "" || method == "plain" {
		return codeVerifier == codeChallenge
	}

	if method == "S256" {
		hash := sha256.Sum256([]byte(codeVerifier))
		computed := base64.RawURLEncoding.EncodeToString(hash[:])
		return computed == codeChallenge
	}

	return false
}
