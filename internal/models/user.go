package models

import (
	"database/sql"
	"fmt"
	"time"
)

type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	IsActive     bool      `json:"isActive"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type UserProfile struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

func (u *User) ToProfile() UserProfile {
	return UserProfile{
		ID:       u.ID,
		Username: u.Username,
		Email:    u.Email,
		Role:     u.Role,
	}
}

type UserRepository struct {
	DB *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{DB: db}
}

func (r *UserRepository) CountUsers() (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM users
	`

	var total int64
	if err := r.DB.QueryRow(query).Scan(&total); err != nil {
		return 0, fmt.Errorf("error counting users: %w", err)
	}

	return total, nil
}

func (r *UserRepository) Create(username, email, passwordHash, role string) (*User, error) {
	query := `
		INSERT INTO users (username, email, password_hash, role, is_active)
		VALUES (?, ?, ?, ?, 1)
	`

	result, err := r.DB.Exec(query, username, email, passwordHash, role)
	if err != nil {
		return nil, fmt.Errorf("error creating user: %w", err)
	}

	userID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("error getting user id: %w", err)
	}

	return r.GetByID(userID)
}

func (r *UserRepository) GetByID(id int64) (*User, error) {
	query := `
		SELECT id, username, email, password_hash, role, is_active, created_at, updated_at
		FROM users
		WHERE id = ?
	`

	var user User
	err := r.DB.QueryRow(query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("error querying user: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) GetByEmail(email string) (*User, error) {
	query := `
		SELECT id, username, email, password_hash, role, is_active, created_at, updated_at
		FROM users
		WHERE email = ?
	`

	var user User
	err := r.DB.QueryRow(query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("error querying user: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) UpdateProfile(userID int64, username, email string) (*User, error) {
	query := `
		UPDATE users
		SET username = ?, email = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	result, err := r.DB.Exec(query, username, email, userID)
	if err != nil {
		return nil, fmt.Errorf("error updating user profile: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("error getting affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return nil, fmt.Errorf("user not found")
	}

	return r.GetByID(userID)
}

func (r *UserRepository) ListOperationalUsers() ([]UserProfile, error) {
	query := `
		SELECT id, username, email, role
		FROM users
		WHERE is_active = 1
		  AND LOWER(TRIM(role)) NOT IN ('admin', 'chi_huy_khoa', 'truong_khoa')
		ORDER BY role, username, id
	`

	rows, err := r.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error listing operational users: %w", err)
	}
	defer rows.Close()

	profiles := make([]UserProfile, 0)
	for rows.Next() {
		var profile UserProfile
		if err := rows.Scan(&profile.ID, &profile.Username, &profile.Email, &profile.Role); err != nil {
			return nil, fmt.Errorf("error scanning operational user: %w", err)
		}
		profiles = append(profiles, profile)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating operational users: %w", err)
	}

	return profiles, nil
}

func (r *UserRepository) ListActiveUsers() ([]UserProfile, error) {
	query := `
		SELECT id, username, email, role
		FROM users
		WHERE is_active = 1
		ORDER BY username, id
	`

	rows, err := r.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error listing users: %w", err)
	}
	defer rows.Close()

	profiles := make([]UserProfile, 0)
	for rows.Next() {
		var profile UserProfile
		if err := rows.Scan(&profile.ID, &profile.Username, &profile.Email, &profile.Role); err != nil {
			return nil, fmt.Errorf("error scanning user: %w", err)
		}
		profiles = append(profiles, profile)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return profiles, nil
}

func (r *UserRepository) UpdateRole(userID int64, role string) (*User, error) {
	query := `
		UPDATE users
		SET role = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND is_active = 1
	`

	result, err := r.DB.Exec(query, role, userID)
	if err != nil {
		return nil, fmt.Errorf("error updating user role: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("error getting affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return nil, fmt.Errorf("user not found")
	}

	return r.GetByID(userID)
}

func (r *UserRepository) DeactivateByID(userID int64) error {
	query := `
		UPDATE users
		SET is_active = 0, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND is_active = 1
	`

	result, err := r.DB.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("error deactivating user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}
