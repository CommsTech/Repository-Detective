package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// GetPlatformSettings returns operator platform overrides (empty if none saved).
func (s *SQLiteStore) GetPlatformSettings(ctx context.Context) (PlatformSettings, error) {
	var raw, updatedAt, updatedBy string
	err := s.db.QueryRowContext(ctx, `
		SELECT settings_json, updated_at, updated_by FROM platform_settings WHERE id = 1
	`).Scan(&raw, &updatedAt, &updatedBy)
	if err == sql.ErrNoRows {
		return PlatformSettings{}, nil
	}
	if err != nil {
		return PlatformSettings{}, fmt.Errorf("get platform settings: %w", err)
	}
	var out PlatformSettings
	if strings.TrimSpace(raw) != "" && raw != "{}" {
		if err := json.Unmarshal([]byte(raw), &out); err != nil {
			return PlatformSettings{}, fmt.Errorf("decode platform settings: %w", err)
		}
	}
	out.UpdatedAt = updatedAt
	out.UpdatedBy = updatedBy
	return out, nil
}

// SavePlatformSettings upserts the single platform settings row.
func (s *SQLiteStore) SavePlatformSettings(ctx context.Context, settings PlatformSettings) error {
	if err := ValidatePlatformSettings(settings); err != nil {
		return err
	}
	payload, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("encode platform settings: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	updatedBy := strings.TrimSpace(settings.UpdatedBy)
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO platform_settings (id, settings_json, updated_at, updated_by)
		VALUES (1, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			settings_json = excluded.settings_json,
			updated_at = excluded.updated_at,
			updated_by = excluded.updated_by
	`, string(payload), now, updatedBy)
	if err != nil {
		return fmt.Errorf("save platform settings: %w", err)
	}
	return nil
}
