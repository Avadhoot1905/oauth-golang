package storage

import (
	"database/sql"
	"fmt"
	"time"
)

// User represents a user in the database
type User struct {
	ID            string    `gorm:"primaryKey;type:uuid"`
	Email         string    `gorm:"uniqueIndex;not null"`
	EmailVerified bool
	Name          string
	GivenName     string
	FamilyName    string
	GoogleID      string    `gorm:"uniqueIndex"`
	Picture       string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// UserRepository handles database operations for users
// DB INTERACTION: All methods interact with the users table
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// GetUserByID retrieves a user by ID
// INPUT FROM DB: Queries users table by ID
func (r *UserRepository) GetUserByID(id string) (*User, error) {
	query := `
		SELECT id, email, email_verified, name, given_name, family_name, picture, google_id, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	user := &User{}
	err := r.db.QueryRow(query, id).Scan(
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

// GetUserByEmail retrieves a user by email
// INPUT FROM DB: Queries users table by email
func (r *UserRepository) GetUserByEmail(email string) (*User, error) {
	query := `
		SELECT id, email, email_verified, name, given_name, family_name, picture, google_id, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	user := &User{}
	err := r.db.QueryRow(query, email).Scan(
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
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return user, nil
}

// GetUserByGoogleID retrieves a user by Google ID
// INPUT FROM DB: Queries users table by google_id
func (r *UserRepository) GetUserByGoogleID(googleID string) (*User, error) {
	query := `
		SELECT id, email, email_verified, name, given_name, family_name, picture, google_id, created_at, updated_at
		FROM users
		WHERE google_id = $1
	`

	user := &User{}
	err := r.db.QueryRow(query, googleID).Scan(
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
		return nil, fmt.Errorf("failed to get user by Google ID: %w", err)
	}

	return user, nil
}

// CreateUser creates a new user
// OUTPUT TO DB: Inserts new user into users table
func (r *UserRepository) CreateUser(user *User) error {
	query := `
		INSERT INTO users (id, email, email_verified, name, given_name, family_name, picture, google_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	now := time.Now()
	_, err := r.db.Exec(
		query,
		user.ID,
		user.Email,
		user.EmailVerified,
		user.Name,
		user.GivenName,
		user.FamilyName,
		user.Picture,
		user.GoogleID,
		now,
		now,
	)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	user.CreatedAt = now
	user.UpdatedAt = now

	return nil
}

// UpdateUser updates an existing user
// OUTPUT TO DB: Updates user record in users table
func (r *UserRepository) UpdateUser(user *User) error {
	query := `
		UPDATE users
		SET email = $2, email_verified = $3, name = $4, given_name = $5, family_name = $6, picture = $7, google_id = $8, updated_at = $9
		WHERE id = $1
	`

	now := time.Now()
	_, err := r.db.Exec(
		query,
		user.ID,
		user.Email,
		user.EmailVerified,
		user.Name,
		user.GivenName,
		user.FamilyName,
		user.Picture,
		user.GoogleID,
		now,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	user.UpdatedAt = now

	return nil
}

// DeleteUser deletes a user
// OUTPUT TO DB: Deletes user from users table
func (r *UserRepository) DeleteUser(id string) error {
	query := `DELETE FROM users WHERE id = $1`

	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

// ListUsers lists all users with pagination
// INPUT FROM DB: Queries users table with limit and offset
func (r *UserRepository) ListUsers(limit, offset int) ([]*User, error) {
	query := `
		SELECT id, email, email_verified, name, given_name, family_name, picture, google_id, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user := &User{}
		err := rows.Scan(
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
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}
