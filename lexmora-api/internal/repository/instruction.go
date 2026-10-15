package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ArminDashti/lexmora-api/internal/domain"
)

type InstructionRepository struct {
	conn *sql.DB
}

func NewInstructionRepository(conn *sql.DB) *InstructionRepository {
	return &InstructionRepository{conn: conn}
}

func (r *InstructionRepository) List(ctx context.Context) ([]domain.Instruction, error) {
	rows, err := r.conn.QueryContext(ctx, `
		SELECT key, content, updated_at FROM instructions ORDER BY key ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list instructions: %w", err)
	}
	defer rows.Close()

	var items []domain.Instruction
	for rows.Next() {
		var i domain.Instruction
		if err := rows.Scan(&i.Key, &i.Content, &i.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

func (r *InstructionRepository) Get(ctx context.Context, key string) (*domain.Instruction, error) {
	var i domain.Instruction
	err := r.conn.QueryRowContext(ctx, `
		SELECT key, content, updated_at FROM instructions WHERE key = ?
	`, key).Scan(&i.Key, &i.Content, &i.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get instruction: %w", err)
	}
	return &i, nil
}

func (r *InstructionRepository) Upsert(ctx context.Context, key, content string) (*domain.Instruction, error) {
	var i domain.Instruction
	err := r.conn.QueryRowContext(ctx, `
		INSERT INTO instructions (key, content) VALUES (?, ?)
		ON CONFLICT (key) DO UPDATE SET content = excluded.content,
			updated_at = strftime('%Y-%m-%d %H:%M:%f', 'now')
		RETURNING key, content, updated_at
	`, key, content).Scan(&i.Key, &i.Content, &i.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("upsert instruction: %w", err)
	}
	return &i, nil
}

func (r *InstructionRepository) Delete(ctx context.Context, key string) error {
	res, err := r.conn.ExecContext(ctx, `DELETE FROM instructions WHERE key = ?`, key)
	if err != nil {
		return fmt.Errorf("delete instruction: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete instruction: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
