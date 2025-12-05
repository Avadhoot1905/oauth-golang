package utils

import (
	"crypto/rand"
	"encoding/base64"
	"math/big"
)

// GenerateRandomString generates a cryptographically secure random string
func GenerateRandomString(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes)[:length]
}

// GenerateRandomCode generates a random numeric code
func GenerateRandomCode(length int) string {
	const digits = "0123456789"
	code := make([]byte, length)
	for i := range code {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			panic(err)
		}
		code[i] = digits[num.Int64()]
	}
	return string(code)
}

// GenerateSecureToken generates a secure random token
func GenerateSecureToken(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return base64.URLEncoding.EncodeToString(bytes)
}

// GenerateState generates a random state parameter for OAuth
func GenerateState() string {
	return GenerateRandomString(32)
}

// GenerateCodeVerifier generates a PKCE code verifier
func GenerateCodeVerifier() string {
	return GenerateRandomString(64)
}
