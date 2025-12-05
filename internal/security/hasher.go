package security

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// Hasher provides password hashing and verification utilities
type Hasher struct {
	cost int
}

// NewHasher creates a new Hasher with default cost
func NewHasher() *Hasher {
	return &Hasher{
		cost: bcrypt.DefaultCost,
	}
}

// NewHasherWithCost creates a new Hasher with custom cost
func NewHasherWithCost(cost int) *Hasher {
	return &Hasher{
		cost: cost,
	}
}

// HashPassword hashes a password using bcrypt
func (h *Hasher) HashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashedBytes), nil
}

// VerifyPassword verifies a password against a hash
func (h *Hasher) VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateSalt generates a random salt
func (h *Hasher) GenerateSalt(length int) (string, error) {
	salt := make([]byte, length)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}
	return base64.StdEncoding.EncodeToString(salt), nil
}

// HashWithSalt hashes data with a salt
func (h *Hasher) HashWithSalt(data, salt string) (string, error) {
	combined := data + salt
	return h.HashPassword(combined)
}

// VerifyWithSalt verifies data with a salt against a hash
func (h *Hasher) VerifyWithSalt(data, salt, hash string) bool {
	combined := data + salt
	return h.VerifyPassword(combined, hash)
}

// GenerateSecureToken generates a cryptographically secure random token
func GenerateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}
