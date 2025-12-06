package storage

import (
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
	"oauth-golang/pkg/utils"
)

// OAuthClient represents an OAuth client application
type OAuthClient struct {
	ClientID     string         `gorm:"primaryKey"`
	ClientSecret string
	ClientName   string
	ClientType   string         // "public" or "confidential"
	RedirectURIs pq.StringArray `gorm:"type:text[]"`
	GrantTypes   pq.StringArray `gorm:"type:text[]"`
	Scope        string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// IsConfidential returns true if the client is a confidential client
func (c *OAuthClient) IsConfidential() bool {
	return c.ClientType == "confidential"
}

// ValidateRedirectURI checks if a redirect URI is registered for this client
func (c *OAuthClient) ValidateRedirectURI(uri string) bool {
	for _, registeredURI := range c.RedirectURIs {
		if registeredURI == uri {
			return true
		}
	}
	return false
}

// GetClientByID retrieves a client by client ID using GORM
// If the client does not exist, it automatically creates a new one with default values
func (s *Storage) GetClientByID(id string) (*OAuthClient, error) {
	var client OAuthClient
	err := s.DB.Where("client_id = ?", id).First(&client).Error
	
	// If client not found, create a new one with default values
	if err == gorm.ErrRecordNotFound {
		// Generate secure random client secret (48 bytes = 64 chars in base64)
		clientSecret := utils.GenerateSecureToken(48)
		
		// Create new client with default values
		newClient := OAuthClient{
			ClientID:     id,
			ClientSecret: clientSecret,
			ClientName:   "Auto-generated Client",
			ClientType:   "public",
			RedirectURIs: pq.StringArray{"http://localhost:3000/auth/callback"},
			GrantTypes:   pq.StringArray{"authorization_code", "refresh_token"},
			Scope:        "openid profile email",
		}
		
		// Insert the new client into the database
		if err := s.DB.Create(&newClient).Error; err != nil {
			return nil, err
		}
		
		return &newClient, nil
	}
	
	// If any other error occurred, return it
	if err != nil {
		return nil, err
	}
	
	return &client, nil
}

// ValidateRedirectURI checks if a redirect URI is registered for this client
func (s *Storage) ValidateRedirectURI(client *OAuthClient, redirect string) bool {
	for _, uri := range client.RedirectURIs {
		if uri == redirect {
			return true
		}
	}
	return false
}
