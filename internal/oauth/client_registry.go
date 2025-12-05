package oauth

import (
	"oauth-golang/internal/storage"
)

// ClientRegistry manages OAuth clients
// DB INTERACTION: Retrieves client information from database
type ClientRegistry struct {
	clientRepo *storage.ClientRepository
}

func NewClientRegistry(clientRepo *storage.ClientRepository) *ClientRegistry {
	return &ClientRegistry{
		clientRepo: clientRepo,
	}
}

// GetClient retrieves a client by client ID
// DB INTERACTION: Queries database via clientRepo
func (r *ClientRegistry) GetClient(clientID string) (*storage.OAuthClient, error) {
	return r.clientRepo.GetClientByID(clientID)
}

// ValidateClient validates client credentials
// DB INTERACTION: Queries database and validates credentials
func (r *ClientRegistry) ValidateClient(clientID, clientSecret string) (*storage.OAuthClient, error) {
	client, err := r.clientRepo.GetClientByID(clientID)
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

// RegisterClient registers a new OAuth client
// OUTPUT TO DB: Creates new client record
func (r *ClientRegistry) RegisterClient(client *storage.OAuthClient) error {
	return r.clientRepo.CreateClient(client)
}
