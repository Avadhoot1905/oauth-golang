package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"oauth-golang/internal/security"
	"oauth-golang/internal/storage"
)

// IntrospectHandler handles the /introspect endpoint
// API INPUT: Receives token introspection requests from other microservices
type IntrospectHandler struct {
	jwtService *security.JWTService
	tokenRepo  *storage.TokenRepository
}

func NewIntrospectHandler(jwtService *security.JWTService, tokenRepo *storage.TokenRepository) *IntrospectHandler {
	return &IntrospectHandler{
		jwtService: jwtService,
		tokenRepo:  tokenRepo,
	}
}

// IntrospectRequest represents the token introspection request
// API INPUT: Request body from other microservices
type IntrospectRequest struct {
	Token         string `json:"token"`
	TokenTypeHint string `json:"token_type_hint,omitempty"` // "access_token" or "refresh_token"
}

// IntrospectResponse represents the token introspection response (RFC 7662)
// API OUTPUT: Token validation result for other microservices
type IntrospectResponse struct {
	Active    bool   `json:"active"`
	Scope     string `json:"scope,omitempty"`
	ClientID  string `json:"client_id,omitempty"`
	Username  string `json:"username,omitempty"`
	TokenType string `json:"token_type,omitempty"`
	Exp       int64  `json:"exp,omitempty"`
	Iat       int64  `json:"iat,omitempty"`
	Sub       string `json:"sub,omitempty"`
	Aud       string `json:"aud,omitempty"`
	Iss       string `json:"iss,omitempty"`
	Jti       string `json:"jti,omitempty"`
}

// Handle processes the /introspect endpoint
// API INPUT: POST request with token to validate
// DB INTERACTION: Checks if token is revoked via tokenRepo
// API OUTPUT: Returns token validation status and metadata
func (h *IntrospectHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request (API INPUT)
	var req IntrospectRequest
	if r.Header.Get("Content-Type") == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.writeInactiveResponse(w)
			return
		}
	} else {
		if err := r.ParseForm(); err != nil {
			h.writeInactiveResponse(w)
			return
		}
		req.Token = r.FormValue("token")
		req.TokenTypeHint = r.FormValue("token_type_hint")
	}

	if req.Token == "" {
		h.writeInactiveResponse(w)
		return
	}

	// Verify JWT token
	claims, err := h.jwtService.VerifyAccessToken(req.Token)
	if err != nil {
		// Token is invalid or expired
		h.writeInactiveResponse(w)
		return
	}

	// Check if token has been revoked (DB INTERACTION via tokenRepo)
	isRevoked, err := h.tokenRepo.IsTokenRevoked(req.Token)
	if err != nil || isRevoked {
		h.writeInactiveResponse(w)
		return
	}

	// Build active response (API OUTPUT)
	response := IntrospectResponse{
		Active:    true,
		Scope:     claims.Scope,
		ClientID:  claims.ClientID,
		Username:  claims.Email,
		TokenType: "Bearer",
		Exp:       claims.ExpiresAt.Unix(),
		Iat:       claims.IssuedAt.Unix(),
		Sub:       claims.Subject,
		Aud:       claims.Audience,
		Iss:       claims.Issuer,
		Jti:       claims.ID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// writeInactiveResponse writes an inactive token response
func (h *IntrospectHandler) writeInactiveResponse(w http.ResponseWriter) {
	response := IntrospectResponse{
		Active: false,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Additional helper for token revocation endpoint (optional)
type RevokeHandler struct {
	jwtService *security.JWTService
	tokenRepo  *storage.TokenRepository
}

func NewRevokeHandler(jwtService *security.JWTService, tokenRepo *storage.TokenRepository) *RevokeHandler {
	return &RevokeHandler{
		jwtService: jwtService,
		tokenRepo:  tokenRepo,
	}
}

// Handle processes token revocation requests
// API INPUT: Token to revoke
// OUTPUT TO DB: Marks token as revoked via tokenRepo
func (rh *RevokeHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	token := r.FormValue("token")
	if token == "" {
		http.Error(w, "Missing token parameter", http.StatusBadRequest)
		return
	}

	// Verify token format
	claims, err := rh.jwtService.VerifyAccessToken(token)
	if err != nil {
		// Even if token is invalid, return success per RFC 7009
		w.WriteHeader(http.StatusOK)
		return
	}

	// Revoke token (OUTPUT TO DB via tokenRepo)
	expiresAt := time.Until(claims.ExpiresAt)
	if err := rh.tokenRepo.RevokeToken(token, expiresAt); err != nil {
		http.Error(w, "Failed to revoke token", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
