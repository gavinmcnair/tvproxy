package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/gavinmcnair/tvproxy/pkg/database"
	"github.com/gavinmcnair/tvproxy/pkg/models"
)

type EPGSourceRepository struct {
	db *database.DB
}

func NewEPGSourceRepository(db *database.DB) *EPGSourceRepository {
	return &EPGSourceRepository{db: db}
}

func (r *EPGSourceRepository) Create(ctx context.Context, source *models.EPGSource) error {
	now := time.Now()
	source.ID = uuid.New().String()
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO epg_sources (id, name, url, is_enabled, last_refreshed, channel_count, program_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		source.ID, source.Name, source.URL, source.IsEnabled, source.LastRefreshed,
		source.ChannelCount, source.ProgramCount, now, now,
	)
	if err != nil {
		return fmt.Errorf("creating epg source: %w", err)
	}
	source.CreatedAt = now
	source.UpdatedAt = now
	return nil
}

func (r *EPGSourceRepository) GetByID(ctx context.Context, id string) (*models.EPGSource, error) {
	source := &models.EPGSource{}
	var lastRefreshed sql.NullTime
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, url, is_enabled, last_refreshed, channel_count, program_count, last_error, created_at, updated_at
		FROM epg_sources WHERE id = ?`, id,
	).Scan(
		&source.ID, &source.Name, &source.URL, &source.IsEnabled,
		&lastRefreshed, &source.ChannelCount, &source.ProgramCount,
		&source.LastError, &source.CreatedAt, &source.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("epg source not found: %w", err)
		}
		return nil, fmt.Errorf("getting epg source by id: %w", err)
	}
	if lastRefreshed.Valid {
		source.LastRefreshed = &lastRefreshed.Time
	}
	return source, nil
}

func (r *EPGSourceRepository) List(ctx context.Context) ([]models.EPGSource, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, url, is_enabled, last_refreshed, channel_count, program_count, last_error, created_at, updated_at
		FROM epg_sources ORDER BY created_at`,
	)
	if err != nil {
		return nil, fmt.Errorf("listing epg sources: %w", err)
	}
	defer rows.Close()

	var sources []models.EPGSource
	for rows.Next() {
		var s models.EPGSource
		var lastRefreshed sql.NullTime
		if err := rows.Scan(
			&s.ID, &s.Name, &s.URL, &s.IsEnabled, &lastRefreshed,
			&s.ChannelCount, &s.ProgramCount, &s.LastError, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning epg source: %w", err)
		}
		if lastRefreshed.Valid {
			s.LastRefreshed = &lastRefreshed.Time
		}
		sources = append(sources, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating epg sources: %w", err)
	}
	return sources, nil
}

func (r *EPGSourceRepository) Update(ctx context.Context, source *models.EPGSource) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		`UPDATE epg_sources SET name = ?, url = ?, is_enabled = ?, last_refreshed = ?, channel_count = ?, program_count = ?, updated_at = ?
		WHERE id = ?`,
		source.Name, source.URL, source.IsEnabled, source.LastRefreshed, source.ChannelCount, source.ProgramCount, now, source.ID,
	)
	if err != nil {
		return fmt.Errorf("updating epg source: %w", err)
	}
	source.UpdatedAt = now
	return nil
}

func (r *EPGSourceRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM epg_sources WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deleting epg source: %w", err)
	}
	return nil
}

func (r *EPGSourceRepository) UpdateLastRefreshed(ctx context.Context, id string, lastRefreshed time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE epg_sources SET last_refreshed = ?, updated_at = ? WHERE id = ?`,
		lastRefreshed, time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("updating last refreshed: %w", err)
	}
	return nil
}

func (r *EPGSourceRepository) UpdateLastError(ctx context.Context, id, lastError string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE epg_sources SET last_error = ?, updated_at = ? WHERE id = ?`,
		lastError, time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("updating last error: %w", err)
	}
	return nil
}

func (r *EPGSourceRepository) UpdateCounts(ctx context.Context, id string, channelCount, programCount int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE epg_sources SET channel_count = ?, program_count = ?, updated_at = ? WHERE id = ?`,
		channelCount, programCount, time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("updating counts: %w", err)
	}
	return nil
}
