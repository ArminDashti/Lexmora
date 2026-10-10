package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ArminDashti/lexmora-api/internal/domain"
)

type SettingsRepository struct {
	pool *pgxpool.Pool
}

func NewSettingsRepository(pool *pgxpool.Pool) *SettingsRepository {
	return &SettingsRepository{pool: pool}
}

func (r *SettingsRepository) Get(ctx context.Context) (*domain.AppSettings, error) {
	var s domain.AppSettings
	err := r.pool.QueryRow(ctx, `
		SELECT openrouter_api_key, gemini_api_key, opencode_go_api_key, api_provider, model_name, updated_at
		FROM app_settings WHERE id = 1
	`).Scan(&s.OpenRouterAPIKey, &s.GeminiAPIKey, &s.OpenCodeGoAPIKey, &s.APIProvider, &s.ModelName, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get settings: %w", err)
	}
	return &s, nil
}

type SettingsUpdate struct {
	OpenRouterAPIKey *string
	GeminiAPIKey     *string
	OpenCodeGoAPIKey *string
	APIProvider      *string
	ModelName        *string
}

func (r *SettingsRepository) Update(ctx context.Context, u SettingsUpdate) (*domain.AppSettings, error) {
	if u.APIProvider != nil {
		p := strings.ToLower(strings.TrimSpace(*u.APIProvider))
		if p != "openrouter" && p != "gemini" && p != "opencode_go" {
			return nil, fmt.Errorf("invalid api_provider: %s", *u.APIProvider)
		}
		*u.APIProvider = p
	}

	_, err := r.pool.Exec(ctx, `
		UPDATE app_settings SET
			openrouter_api_key = COALESCE($1, openrouter_api_key),
			gemini_api_key = COALESCE($2, gemini_api_key),
			opencode_go_api_key = COALESCE($3, opencode_go_api_key),
			api_provider = COALESCE($4, api_provider),
			model_name = COALESCE($5, model_name),
			updated_at = now()
		WHERE id = 1
	`, u.OpenRouterAPIKey, u.GeminiAPIKey, u.OpenCodeGoAPIKey, u.APIProvider, u.ModelName)
	if err != nil {
		return nil, fmt.Errorf("update settings: %w", err)
	}
	return r.Get(ctx)
}

func (r *SettingsRepository) ClearAllData(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM history`)
	return err
}
