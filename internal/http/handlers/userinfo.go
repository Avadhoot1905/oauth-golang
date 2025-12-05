package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"oauth-golang/internal/security"
	"oauth-golang/internal/storage"
)

// UserInfoHandler handles the /userinfo endpoint
// API INPUT: Receives access token from Authorization header
type UserInfoHandler struct {
	jwtService *security.JWTService
	userRepo   *storage.UserRepository
}

func NewUserInfoHandler(jwtService *security.JWTService, userRepo *storage.UserRepository) *UserInfoHandler {
	return &UserInfoHandler{
		jwtService: jwtService,
		userRepo:   userRepo,
	}
}

// UserInfoResponse represents the user information response
// API OUTPUT: User information returned to client
type UserInfoResponse struct {
	Sub           string `json:"sub"` // Subject (user ID)
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name,omitempty"`
	FamilyName    string `json:"family_name,omitempty"`
	Picture       string `json:"picture,omitempty"`
}

// Handle processes the /userinfo endpoint
// API INPUT: Authorization header with Bearer token
// DB INTERACTION: Retrieves user info from database via userRepo
// API OUTPUT: Returns user information as JSON
func (h *UserInfoHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract access token from Authorization header (API INPUT)
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		h.writeError(w, "invalid_request", "Missing Authorization header", http.StatusUnauthorized)
		return
	}

	// Parse Bearer token
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		h.writeError(w, "invalid_request", "Invalid Authorization header format", http.StatusUnauthorized)
		return
	}

	accessToken := parts[1]

	// Verify and decode JWT token
	claims, err := h.jwtService.VerifyAccessToken(accessToken)
	if err != nil {
		h.writeError(w, "invalid_token", "Invalid or expired token", http.StatusUnauthorized)
		return
	}

	// Retrieve user information from database (DB INTERACTION via userRepo)
	user, err := h.userRepo.GetUserByID(claims.Subject)
	if err != nil {
		h.writeError(w, "server_error", "Failed to retrieve user information", http.StatusInternalServerError)
		return
	}

	if user == nil {
		h.writeError(w, "invalid_token", "User not found", http.StatusUnauthorized)
		return
	}

	// Build user info response (API OUTPUT)
	response := UserInfoResponse{
		Sub:           user.ID,
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
		Name:          user.Name,
		GivenName:     user.GivenName,
		FamilyName:    user.FamilyName,
		Picture:       user.Picture,
	}

	// Return user info as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// writeError writes an error response
func (h *UserInfoHandler) writeError(w http.ResponseWriter, errorCode, description string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("WWW-Authenticate", `Bearer error="`+errorCode+`", error_description="`+description+`"`)
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error":             errorCode,
		"error_description": description,
	})
}
