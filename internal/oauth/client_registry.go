package oauth

import (
	"oauth-golang/internal/storage"
)

// ClientRegistry manages OAuth clients
// DB INTERACTION: Retrieves client information from database
type ClientRegistry struct {
	storage *storage.Storage
}

func NewClientRegistry(storage *storage.Storage) *ClientRegistry {
	return &ClientRegistry{
		storage: storage,
	}
}

// GetClient retrieves a client by client ID
// DB INTERACTION: Queries database via storage
func (r *ClientRegistry) GetClient(clientID string) (*storage.OAuthClient, error) {
	return r.storage.GetClientByID(clientID)
}

// ValidateClient validates client credentials
// DB INTERACTION: Queries database and validates credentials
func (r *ClientRegistry) ValidateClient(clientID, clientSecret string) (*storage.OAuthClient, error) {
	client, err := r.storage.GetClientByID(clientID)
	if err != nil {
		return nil, err
	}

	if client == nil {
		return nil, nil
	}

	// For confidential clients, validate secret
	if client.IsConfidential() && client.ClientSecret != clientSecret {
		return nil, nil
	}

	return client, nil
}
