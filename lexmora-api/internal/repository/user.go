package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ArminDashti/lexmora-api/internal/domain"
	"github.com/google/uuid"
)

type UserRepository struct {
	conn *sql.DB
}

func NewUserRepository(conn *sql.DB) *UserRepository {
	return &UserRepository{conn: conn}
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	var u domain.User
	err := r.conn.QueryRowContext(ctx, `
		SELECT id, username, password_hash, created_at
		FROM users WHERE username = ?
	`, username).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) Create(ctx context.Context, username, passwordHash string) (*domain.User, error) {
	var u domain.User
	err := r.conn.QueryRowContext(ctx, `
		INSERT INTO users (id, username, password_hash)
		VALUES (?, ?, ?)
		RETURNING id, username, password_hash, created_at
	`, uuid.New(), username, passwordHash).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}
