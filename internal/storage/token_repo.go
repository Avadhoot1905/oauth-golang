package storage

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

// RefreshToken represents a refresh token in the database
type RefreshToken struct {
	Token     string
	UserID    string
	ClientID  string
	Scope     string
	ExpiresAt time.Time
	Revoked   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TokenRepository handles database operations for tokens
// DB INTERACTION: All methods interact with refresh_tokens and revoked_tokens tables
type TokenRepository struct {
	db *sql.DB
}

// NewTokenRepository creates a new token repository
func NewTokenRepository(db *sql.DB) *TokenRepository {
	return &TokenRepository{db: db}
}

// StoreRefreshToken stores a refresh token
// OUTPUT TO DB: Inserts token into refresh_tokens table
func (r *TokenRepository) StoreRefreshToken(token, userID string, expiresAt time.Time) error {
	query := `
		INSERT INTO refresh_tokens (token, user_id, client_id, scope, expires_at, revoked, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	now := time.Now()
	_, err := r.db.Exec(
		query,
		token,
		userID,
		"oauth-service",
		"openid email profile",
		expiresAt,
		false,
		now,
		now,
	)

	if err != nil {
		return fmt.Errorf("failed to store refresh token: %w", err)
	}

	return nil
}

// GetRefreshToken retrieves a refresh token
// INPUT FROM DB: Queries refresh_tokens table
func (r *TokenRepository) GetRefreshToken(token string) (*RefreshToken, error) {
	query := `
		SELECT token, user_id, client_id, scope, expires_at, revoked, created_at, updated_at
		FROM refresh_tokens
		WHERE token = $1
	`

	rt := &RefreshToken{}
	err := r.db.QueryRow(query, token).Scan(
		&rt.Token,
		&rt.UserID,
		&rt.ClientID,
		&rt.Scope,
		&rt.ExpiresAt,
		&rt.Revoked,
		&rt.CreatedAt,
		&rt.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	return rt, nil
}

// RevokeRefreshToken marks a refresh token as revoked
// OUTPUT TO DB: Updates revoked flag in refresh_tokens table
func (r *TokenRepository) RevokeRefreshToken(token string) error {
	query := `
		UPDATE refresh_tokens
		SET revoked = true, updated_at = $2
		WHERE token = $1
	`

	_, err := r.db.Exec(query, token, time.Now())
	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	return nil
}

// DeleteExpiredRefreshTokens deletes expired refresh tokens
// OUTPUT TO DB: Deletes expired tokens from refresh_tokens table
func (r *TokenRepository) DeleteExpiredRefreshTokens() error {
	query := `DELETE FROM refresh_tokens WHERE expires_at < $1`

	_, err := r.db.Exec(query, time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete expired refresh tokens: %w", err)
	}

	return nil
}

// RevokeToken adds a token to the revoked tokens list (blacklist)
// OUTPUT TO DB: Inserts token hash into revoked_tokens table
func (r *TokenRepository) RevokeToken(token string, ttl time.Duration) error {
	// Hash the token before storing for privacy
	tokenHash := hashToken(token)
	expiresAt := time.Now().Add(ttl)

	query := `
		INSERT INTO revoked_tokens (token_hash, expires_at, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (token_hash) DO NOTHING
	`

	_, err := r.db.Exec(query, tokenHash, expiresAt, time.Now())
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	return nil
}

// IsTokenRevoked checks if a token is revoked
// INPUT FROM DB: Queries revoked_tokens table
func (r *TokenRepository) IsTokenRevoked(token string) (bool, error) {
	tokenHash := hashToken(token)

	query := `
		SELECT EXISTS(
			SELECT 1 FROM revoked_tokens 
			WHERE token_hash = $1 AND expires_at > $2
		)
	`

	var exists bool
	err := r.db.QueryRow(query, tokenHash, time.Now()).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if token is revoked: %w", err)
	}

	return exists, nil
}

// DeleteExpiredRevokedTokens deletes expired entries from revoked tokens
// OUTPUT TO DB: Deletes expired tokens from revoked_tokens table
func (r *TokenRepository) DeleteExpiredRevokedTokens() error {
	query := `DELETE FROM revoked_tokens WHERE expires_at < $1`

	_, err := r.db.Exec(query, time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete expired revoked tokens: %w", err)
	}

	return nil
}

// GetUserByID retrieves a user by ID (helper method)
// INPUT FROM DB: Queries users table
func (r *TokenRepository) GetUserByID(userID string) (*User, error) {
	query := `
		SELECT id, email, email_verified, name, given_name, family_name, picture, google_id, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	user := &User{}
	err := r.db.QueryRow(query, userID).Scan(
		&user.ID,
		&user.Email,
		&user.EmailVerified,
		&user.Name,
		&user.GivenName,
		&user.FamilyName,
		&user.Picture,
		&user.GoogleID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// hashToken creates a SHA-256 hash of a token
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
