package oauth

import (
	"fmt"
	"time"

	"oauth-golang/internal/config"
	"oauth-golang/internal/security"
	"oauth-golang/internal/storage"
)

// TokenPair represents an access token and refresh token pair
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	IDToken      string
	ExpiresIn    time.Duration
}

// TokenService handles token generation and refresh
// OUTPUT TO DB: Stores refresh tokens in database via tokenRepo
type TokenService struct {
	config     *config.Config
	jwtService *security.JWTService
	tokenRepo  *storage.TokenRepository
}

func NewTokenService(
	cfg *config.Config,
	jwtService *security.JWTService,
	tokenRepo *storage.TokenRepository,
) *TokenService {
	return &TokenService{
		config:     cfg,
		jwtService: jwtService,
		tokenRepo:  tokenRepo,
	}
}

// GenerateTokens creates a new access token and refresh token pair
// OUTPUT TO DB: Stores refresh token in database
func (s *TokenService) GenerateTokens(user *storage.User, scope string) (*TokenPair, error) {
	// Generate access token (short-lived, 1 hour)
	accessToken, err := s.jwtService.GenerateAccessToken(&security.TokenClaims{
		Subject:  user.ID,
		Email:    user.Email,
		Name:     user.Name,
		Scope:    scope,
		ClientID: "oauth-service",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token (long-lived, 30 days)
	refreshToken, err := s.jwtService.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Generate ID token (contains user identity information)
	idToken, err := s.jwtService.GenerateIDToken(&security.TokenClaims{
		Subject:       user.ID,
		Email:         user.Email,
		Name:          user.Name,
		EmailVerified: user.EmailVerified,
		Picture:       user.Picture,
		GivenName:     user.GivenName,
		FamilyName:    user.FamilyName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate ID token: %w", err)
	}

	// Store refresh token in database (OUTPUT TO DB)
	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	if err := s.tokenRepo.StoreRefreshToken(refreshToken, user.ID, expiresAt); err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		IDToken:      idToken,
		ExpiresIn:    1 * time.Hour,
	}, nil
}

// RefreshTokens generates new tokens using a refresh token
// DB INTERACTION: Validates refresh token from database, stores new refresh token
func (s *TokenService) RefreshTokens(refreshToken, clientID string) (*TokenPair, error) {
	// Verify refresh token
	claims, err := s.jwtService.VerifyRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token")
	}

	// Check if refresh token exists and is valid in database (DB INTERACTION)
	storedToken, err := s.tokenRepo.GetRefreshToken(refreshToken)
	if err != nil || storedToken == nil {
		return nil, fmt.Errorf("refresh token not found or expired")
	}

	// Check if token is revoked
	if storedToken.Revoked {
		return nil, fmt.Errorf("refresh token has been revoked")
	}

	// Check if token has expired
	if time.Now().After(storedToken.ExpiresAt) {
		return nil, fmt.Errorf("refresh token has expired")
	}

	// Get user information
	user, err := s.tokenRepo.GetUserByID(storedToken.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// Generate new tokens
	tokens, err := s.GenerateTokens(user, claims.Scope)
	if err != nil {
		return nil, err
	}

	// Optionally: Revoke old refresh token (refresh token rotation)
	// s.tokenRepo.RevokeRefreshToken(refreshToken)

	return tokens, nil
}

// RevokeToken revokes an access or refresh token
// OUTPUT TO DB: Marks token as revoked
func (s *TokenService) RevokeToken(token string) error {
	// Verify token to get expiration time
	claims, err := s.jwtService.VerifyAccessToken(token)
	if err != nil {
		// Try as refresh token
		claims, err = s.jwtService.VerifyRefreshToken(token)
		if err != nil {
			return fmt.Errorf("invalid token")
		}
	}

	// Store revoked token until it expires (OUTPUT TO DB)
	expiresAt := time.Until(claims.ExpiresAt)
	return s.tokenRepo.RevokeToken(token, expiresAt)
}
