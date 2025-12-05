package oauth

import (
	"sync"
	"time"

	"oauth-golang/internal/models"
)

// AuthSession represents a temporary OAuth session during authorization
type AuthSession struct {
	ClientID            string
	RedirectURI         string
	State               string
	CodeChallenge       string
	CodeChallengeMethod string
	Scope               string
	CreatedAt           time.Time
}

// AuthCode represents an authorization code
type AuthCode struct {
	Code                string
	ClientID            string
	RedirectURI         string
	UserID              string
	CodeChallenge       string
	CodeChallengeMethod string
	Scope               string
	ExpiresAt           time.Time
	UserInfo            *models.GoogleUserInfo
}

// AuthCodeService manages authorization codes and sessions
// In production, use Redis or database with TTL instead of in-memory storage
type AuthCodeService struct {
	sessions  map[string]*AuthSession
	authCodes map[string]*AuthCode
	mu        sync.RWMutex
}

func NewAuthCodeService() *AuthCodeService {
	service := &AuthCodeService{
		sessions:  make(map[string]*AuthSession),
		authCodes: make(map[string]*AuthCode),
	}
	
	// Start cleanup goroutine for expired codes
	go service.cleanupExpired()
	
	return service
}

// StoreSession stores a temporary OAuth session
func (s *AuthCodeService) StoreSession(sessionID string, session *AuthSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sessionID] = session
}

// GetSession retrieves a session by ID
func (s *AuthCodeService) GetSession(sessionID string) *AuthSession {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessions[sessionID]
}

// DeleteSession removes a session
func (s *AuthCodeService) DeleteSession(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
}

// StoreAuthCode stores an authorization code
func (s *AuthCodeService) StoreAuthCode(code string, authCode *AuthCode) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.authCodes[code] = authCode
}

// GetAuthCode retrieves an authorization code
func (s *AuthCodeService) GetAuthCode(code string) *AuthCode {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.authCodes[code]
}

// DeleteAuthCode removes an authorization code (one-time use)
func (s *AuthCodeService) DeleteAuthCode(code string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.authCodes, code)
}

// cleanupExpired removes expired authorization codes and sessions
func (s *AuthCodeService) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()

		// Clean up expired auth codes
		for code, authCode := range s.authCodes {
			if now.After(authCode.ExpiresAt) {
				delete(s.authCodes, code)
			}
		}

		// Clean up old sessions (older than 15 minutes)
		for sessionID, session := range s.sessions {
			if now.Sub(session.CreatedAt) > 15*time.Minute {
				delete(s.sessions, sessionID)
			}
		}

		s.mu.Unlock()
	}
}
