package security

import (
	"fmt"
	"time"

	"oauth-golang/pkg/utils"

	"github.com/golang-jwt/jwt/v5"
)

// TokenClaims represents JWT token claims
type TokenClaims struct {
	Subject       string    `json:"sub"`
	Email         string    `json:"email"`
	Name          string    `json:"name"`
	GivenName     string    `json:"given_name,omitempty"`
	FamilyName    string    `json:"family_name,omitempty"`
	Picture       string    `json:"picture,omitempty"`
	EmailVerified bool      `json:"email_verified,omitempty"`
	Scope         string    `json:"scope,omitempty"`
	ClientID      string    `json:"client_id,omitempty"`
	Issuer        string    `json:"iss,omitempty"`
	Audience      string    `json:"aud,omitempty"`
	ExpiresAt     time.Time `json:"exp"`
	IssuedAt      time.Time `json:"iat"`
	ID            string    `json:"jti,omitempty"`
}

// JWTService handles JWT token generation and verification
type JWTService struct {
	secret string
	issuer string
}

// NewJWTService creates a new JWT service
func NewJWTService(secret string) *JWTService {
	return &JWTService{
		secret: secret,
		issuer: "oauth-service",
	}
}

// GenerateAccessToken generates a new access token (short-lived)
func (s *JWTService) GenerateAccessToken(claims *TokenClaims) (string, error) {
	now := time.Now()
	expiresAt := now.Add(1 * time.Hour) // Access tokens expire in 1 hour

	jwtClaims := jwt.MapClaims{
		"sub":       claims.Subject,
		"email":     claims.Email,
		"name":      claims.Name,
		"scope":     claims.Scope,
		"client_id": claims.ClientID,
		"iss":       s.issuer,
		"aud":       "oauth-service",
		"exp":       expiresAt.Unix(),
		"iat":       now.Unix(),
		"jti":       utils.GenerateRandomString(16),
		"type":      "access",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaims)
	return token.SignedString([]byte(s.secret))
}

// GenerateRefreshToken generates a new refresh token (long-lived)
func (s *JWTService) GenerateRefreshToken(userID string) (string, error) {
	now := time.Now()
	expiresAt := now.Add(30 * 24 * time.Hour) // Refresh tokens expire in 30 days

	jwtClaims := jwt.MapClaims{
		"sub":  userID,
		"iss":  s.issuer,
		"aud":  "oauth-service",
		"exp":  expiresAt.Unix(),
		"iat":  now.Unix(),
		"jti":  utils.GenerateRandomString(16),
		"type": "refresh",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaims)
	return token.SignedString([]byte(s.secret))
}

// GenerateIDToken generates an OpenID Connect ID token
func (s *JWTService) GenerateIDToken(claims *TokenClaims) (string, error) {
	now := time.Now()
	expiresAt := now.Add(1 * time.Hour)

	jwtClaims := jwt.MapClaims{
		"sub":            claims.Subject,
		"email":          claims.Email,
		"email_verified": claims.EmailVerified,
		"name":           claims.Name,
		"iss":            s.issuer,
		"aud":            "oauth-service",
		"exp":            expiresAt.Unix(),
		"iat":            now.Unix(),
		"type":           "id",
	}

	// Add optional claims
	if claims.GivenName != "" {
		jwtClaims["given_name"] = claims.GivenName
	}
	if claims.FamilyName != "" {
		jwtClaims["family_name"] = claims.FamilyName
	}
	if claims.Picture != "" {
		jwtClaims["picture"] = claims.Picture
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaims)
	return token.SignedString([]byte(s.secret))
}

// VerifyAccessToken verifies and decodes an access token
func (s *JWTService) VerifyAccessToken(tokenString string) (*TokenClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Verify token type
	if tokenType, ok := claims["type"].(string); !ok || tokenType != "access" {
		return nil, fmt.Errorf("not an access token")
	}

	return s.mapClaimsToTokenClaims(claims), nil
}

// VerifyRefreshToken verifies and decodes a refresh token
func (s *JWTService) VerifyRefreshToken(tokenString string) (*TokenClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Verify token type
	if tokenType, ok := claims["type"].(string); !ok || tokenType != "refresh" {
		return nil, fmt.Errorf("not a refresh token")
	}

	return s.mapClaimsToTokenClaims(claims), nil
}

// mapClaimsToTokenClaims converts JWT claims to TokenClaims struct
func (s *JWTService) mapClaimsToTokenClaims(claims jwt.MapClaims) *TokenClaims {
	tokenClaims := &TokenClaims{}

	if sub, ok := claims["sub"].(string); ok {
		tokenClaims.Subject = sub
	}
	if email, ok := claims["email"].(string); ok {
		tokenClaims.Email = email
	}
	if name, ok := claims["name"].(string); ok {
		tokenClaims.Name = name
	}
	if givenName, ok := claims["given_name"].(string); ok {
		tokenClaims.GivenName = givenName
	}
	if familyName, ok := claims["family_name"].(string); ok {
		tokenClaims.FamilyName = familyName
	}
	if picture, ok := claims["picture"].(string); ok {
		tokenClaims.Picture = picture
	}
	if emailVerified, ok := claims["email_verified"].(bool); ok {
		tokenClaims.EmailVerified = emailVerified
	}
	if scope, ok := claims["scope"].(string); ok {
		tokenClaims.Scope = scope
	}
	if clientID, ok := claims["client_id"].(string); ok {
		tokenClaims.ClientID = clientID
	}
	if iss, ok := claims["iss"].(string); ok {
		tokenClaims.Issuer = iss
	}
	if aud, ok := claims["aud"].(string); ok {
		tokenClaims.Audience = aud
	}
	if jti, ok := claims["jti"].(string); ok {
		tokenClaims.ID = jti
	}

	// Parse time fields
	if exp, ok := claims["exp"].(float64); ok {
		tokenClaims.ExpiresAt = time.Unix(int64(exp), 0)
	}
	if iat, ok := claims["iat"].(float64); ok {
		tokenClaims.IssuedAt = time.Unix(int64(iat), 0)
	}

	return tokenClaims
}

// ExtractToken extracts the token from Authorization header
func ExtractToken(authHeader string) (string, error) {
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		return "", fmt.Errorf("invalid authorization header")
	}
	return authHeader[7:], nil
}
