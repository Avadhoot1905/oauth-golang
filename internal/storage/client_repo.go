package storage

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"
)

// OAuthClient represents an OAuth client application
type OAuthClient struct {
	ClientID     string `gorm:"primaryKey"`
	ClientSecret string
	ClientName   string
	ClientType   string   `gorm:"default:'confidential'"` // "confidential" or "public"
	RedirectURIs []string `gorm:"type:text[]"`
	GrantTypes   []string `gorm:"type:text[]"`
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

// ClientRepository handles database operations for OAuth clients
// DB INTERACTION: All methods interact with the oauth_clients table
type ClientRepository struct {
	db *sql.DB
}

// NewClientRepository creates a new client repository
func NewClientRepository(db *sql.DB) *ClientRepository {
	return &ClientRepository{db: db}
}

// GetClientByID retrieves a client by client ID
// INPUT FROM DB: Queries oauth_clients table
func (r *ClientRepository) GetClientByID(clientID string) (*OAuthClient, error) {
	query := `
		SELECT client_id, client_secret, client_name, client_type, redirect_uris, grant_types, scope, created_at, updated_at
		FROM oauth_clients
		WHERE client_id = $1
	`

	client := &OAuthClient{}
	err := r.db.QueryRow(query, clientID).Scan(
		&client.ClientID,
		&client.ClientSecret,
		&client.ClientName,
		&client.ClientType,
		pq.Array(&client.RedirectURIs),
		pq.Array(&client.GrantTypes),
		&client.Scope,
		&client.CreatedAt,
		&client.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}

	return client, nil
}

// CreateClient creates a new OAuth client
// OUTPUT TO DB: Inserts new client into oauth_clients table
func (r *ClientRepository) CreateClient(client *OAuthClient) error {
	query := `
		INSERT INTO oauth_clients (client_id, client_secret, client_name, client_type, redirect_uris, grant_types, scope, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	now := time.Now()
	_, err := r.db.Exec(
		query,
		client.ClientID,
		client.ClientSecret,
		client.ClientName,
		client.ClientType,
		pq.Array(client.RedirectURIs),
		pq.Array(client.GrantTypes),
		client.Scope,
		now,
		now,
	)

	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	client.CreatedAt = now
	client.UpdatedAt = now

	return nil
}

// UpdateClient updates an existing OAuth client
// OUTPUT TO DB: Updates client in oauth_clients table
func (r *ClientRepository) UpdateClient(client *OAuthClient) error {
	query := `
		UPDATE oauth_clients
		SET client_secret = $2, client_name = $3, client_type = $4, redirect_uris = $5, grant_types = $6, scope = $7, updated_at = $8
		WHERE client_id = $1
	`

	now := time.Now()
	_, err := r.db.Exec(
		query,
		client.ClientID,
		client.ClientSecret,
		client.ClientName,
		client.ClientType,
		pq.Array(client.RedirectURIs),
		pq.Array(client.GrantTypes),
		client.Scope,
		now,
	)

	if err != nil {
		return fmt.Errorf("failed to update client: %w", err)
	}

	client.UpdatedAt = now

	return nil
}

// DeleteClient deletes an OAuth client
// OUTPUT TO DB: Deletes client from oauth_clients table
func (r *ClientRepository) DeleteClient(clientID string) error {
	query := `DELETE FROM oauth_clients WHERE client_id = $1`

	_, err := r.db.Exec(query, clientID)
	if err != nil {
		return fmt.Errorf("failed to delete client: %w", err)
	}

	return nil
}

// ListClients lists all OAuth clients
// INPUT FROM DB: Queries all clients from oauth_clients table
func (r *ClientRepository) ListClients() ([]*OAuthClient, error) {
	query := `
		SELECT client_id, client_secret, client_name, client_type, redirect_uris, grant_types, scope, created_at, updated_at
		FROM oauth_clients
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list clients: %w", err)
	}
	defer rows.Close()

	var clients []*OAuthClient
	for rows.Next() {
		client := &OAuthClient{}
		err := rows.Scan(
			&client.ClientID,
			&client.ClientSecret,
			&client.ClientName,
			&client.ClientType,
			pq.Array(&client.RedirectURIs),
			pq.Array(&client.GrantTypes),
			&client.Scope,
			&client.CreatedAt,
			&client.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan client: %w", err)
		}
		clients = append(clients, client)
	}

	return clients, nil
}
