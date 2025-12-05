package oauth

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
)

// PKCEValidator validates PKCE (Proof Key for Code Exchange) parameters
// Used for public clients (mobile apps, SPAs) to prevent authorization code interception
type PKCEValidator struct{}

func NewPKCEValidator() *PKCEValidator {
	return &PKCEValidator{}
}

// ValidateCodeChallenge validates the code_challenge parameter
func (v *PKCEValidator) ValidateCodeChallenge(codeChallenge, method string) bool {
	if codeChallenge == "" {
		return false
	}

	// Validate code challenge format (base64url encoded)
	if !isBase64URL(codeChallenge) {
		return false
	}

	// Validate code challenge method
	if method != "" && method != "plain" && method != "S256" {
		return false
	}

	// code_challenge must be between 43-128 characters
	if len(codeChallenge) < 43 || len(codeChallenge) > 128 {
		return false
	}

	return true
}

// VerifyCodeChallenge verifies the code_verifier against the code_challenge
func (v *PKCEValidator) VerifyCodeChallenge(codeVerifier, codeChallenge, method string) bool {
	if codeVerifier == "" || codeChallenge == "" {
		return false
	}

	// code_verifier must be between 43-128 characters
	if len(codeVerifier) < 43 || len(codeVerifier) > 128 {
		return false
	}

	// Verify based on method
	if method == "" || method == "plain" {
		return codeVerifier == codeChallenge
	}

	if method == "S256" {
		// Compute SHA256 hash of code_verifier
		hash := sha256.Sum256([]byte(codeVerifier))
		computed := base64.RawURLEncoding.EncodeToString(hash[:])
		return computed == codeChallenge
	}

	return false
}

// GenerateCodeChallenge generates a code_challenge from a code_verifier
// This is typically done by the client, but useful for testing
func (v *PKCEValidator) GenerateCodeChallenge(codeVerifier, method string) string {
	if method == "" || method == "plain" {
		return codeVerifier
	}

	if method == "S256" {
		hash := sha256.Sum256([]byte(codeVerifier))
		return base64.RawURLEncoding.EncodeToString(hash[:])
	}

	return ""
}

// isBase64URL checks if a string is valid base64url encoding
func isBase64URL(s string) bool {
	// Base64URL uses A-Z, a-z, 0-9, -, _ (no padding)
	for _, c := range s {
		if !((c >= 'A' && c <= 'Z') ||
			(c >= 'a' && c <= 'z') ||
			(c >= '0' && c <= '9') ||
			c == '-' || c == '_') {
			return false
		}
	}
	return true
}

// ValidateCodeVerifier validates the code_verifier parameter
func (v *PKCEValidator) ValidateCodeVerifier(codeVerifier string) bool {
	// code_verifier must be between 43-128 characters
	if len(codeVerifier) < 43 || len(codeVerifier) > 128 {
		return false
	}

	// Must contain only unreserved characters: A-Z, a-z, 0-9, -, ., _, ~
	for _, c := range codeVerifier {
		if !((c >= 'A' && c <= 'Z') ||
			(c >= 'a' && c <= 'z') ||
			(c >= '0' && c <= '9') ||
			c == '-' || c == '.' || c == '_' || c == '~') {
			return false
		}
	}

	return true
}

// ShouldUsePKCE determines if PKCE should be required for a client
func (v *PKCEValidator) ShouldUsePKCE(clientType string) bool {
	// Public clients (SPAs, mobile apps) should always use PKCE
	return strings.ToLower(clientType) == "public"
}
