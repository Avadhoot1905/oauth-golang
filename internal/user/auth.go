package user

import (
	"fmt"

	"oauth-golang/internal/models"
	"oauth-golang/internal/storage"
	"oauth-golang/pkg/utils"
)

// AuthService handles user authentication operations
type AuthService struct {
	userRepo *storage.UserRepository
}

// NewAuthService creates a new authentication service
func NewAuthService(userRepo *storage.UserRepository) *AuthService {
	return &AuthService{
		userRepo: userRepo,
	}
}

// CreateOrUpdateUser creates a new user or updates an existing one from Google OAuth info
// OUTPUT TO DB: Creates or updates user via userRepo
func (s *AuthService) CreateOrUpdateUser(userID string, googleUserInfo *models.GoogleUserInfo) (*storage.User, error) {
	// Check if user exists by Google ID
	existingUser, err := s.userRepo.GetUserByGoogleID(googleUserInfo.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	// If user doesn't exist by Google ID, check by email
	if existingUser == nil {
		existingUser, err = s.userRepo.GetUserByEmail(googleUserInfo.Email)
		if err != nil {
			return nil, fmt.Errorf("failed to check user by email: %w", err)
		}
	}

	// Build user object
	user := &storage.User{
		Email:         googleUserInfo.Email,
		EmailVerified: googleUserInfo.VerifiedEmail,
		Name:          googleUserInfo.Name,
		GivenName:     googleUserInfo.GivenName,
		FamilyName:    googleUserInfo.FamilyName,
		Picture:       googleUserInfo.Picture,
		GoogleID:      googleUserInfo.ID,
	}

	if existingUser != nil {
		// Update existing user
		user.ID = existingUser.ID
		if err := s.userRepo.UpdateUser(user); err != nil {
			return nil, fmt.Errorf("failed to update user: %w", err)
		}
	} else {
		// Create new user with generated ID
		user.ID = utils.GenerateRandomString(16)
		if err := s.userRepo.CreateUser(user); err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
	}

	return user, nil
}

// GetUser retrieves a user by ID
// INPUT FROM DB: Queries user via userRepo
func (s *AuthService) GetUser(userID string) (*storage.User, error) {
	return s.userRepo.GetUserByID(userID)
}

// GetUserByEmail retrieves a user by email
// INPUT FROM DB: Queries user via userRepo
func (s *AuthService) GetUserByEmail(email string) (*storage.User, error) {
	return s.userRepo.GetUserByEmail(email)
}

// AuthenticateUser authenticates a user (placeholder for future password-based auth)
func (s *AuthService) AuthenticateUser(email, password string) (*storage.User, error) {
	// This would be used for password-based authentication
	// For now, we only support Google OAuth
	return nil, fmt.Errorf("password authentication not implemented")
}
