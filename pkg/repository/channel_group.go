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

type ChannelGroupRepository struct {
	db *database.DB
}

func NewChannelGroupRepository(db *database.DB) *ChannelGroupRepository {
	return &ChannelGroupRepository{db: db}
}

func (r *ChannelGroupRepository) Create(ctx context.Context, group *models.ChannelGroup) error {
	now := time.Now()
	group.ID = uuid.New().String()
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO channel_groups (id, user_id, name, is_enabled, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		group.ID, group.UserID, group.Name, group.IsEnabled, group.SortOrder, now, now,
	)
	if err != nil {
		return fmt.Errorf("creating channel group: %w", err)
	}
	group.CreatedAt = now
	group.UpdatedAt = now
	return nil
}

func (r *ChannelGroupRepository) GetByID(ctx context.Context, id string) (*models.ChannelGroup, error) {
	group := &models.ChannelGroup{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, name, is_enabled, sort_order, created_at, updated_at
		FROM channel_groups WHERE id = ?`, id,
	).Scan(&group.ID, &group.UserID, &group.Name, &group.IsEnabled, &group.SortOrder, &group.CreatedAt, &group.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("channel group not found: %w", err)
		}
		return nil, fmt.Errorf("getting channel group by id: %w", err)
	}
	return group, nil
}

func (r *ChannelGroupRepository) GetByName(ctx context.Context, name string) (*models.ChannelGroup, error) {
	group := &models.ChannelGroup{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, name, is_enabled, sort_order, created_at, updated_at
		FROM channel_groups WHERE name = ?`, name,
	).Scan(&group.ID, &group.UserID, &group.Name, &group.IsEnabled, &group.SortOrder, &group.CreatedAt, &group.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("channel group not found: %w", err)
		}
		return nil, fmt.Errorf("getting channel group by name: %w", err)
	}
	return group, nil
}

func (r *ChannelGroupRepository) List(ctx context.Context) ([]models.ChannelGroup, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, name, is_enabled, sort_order, created_at, updated_at
		FROM channel_groups ORDER BY sort_order, name`,
	)
	if err != nil {
		return nil, fmt.Errorf("listing channel groups: %w", err)
	}
	defer rows.Close()

	return r.scanRows(rows)
}

func (r *ChannelGroupRepository) ListByUserID(ctx context.Context, userID string) ([]models.ChannelGroup, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, name, is_enabled, sort_order, created_at, updated_at
		FROM channel_groups WHERE user_id = ? ORDER BY sort_order, name`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing channel groups by user: %w", err)
	}
	defer rows.Close()

	return r.scanRows(rows)
}

func (r *ChannelGroupRepository) GetByIDForUser(ctx context.Context, id, userID string) (*models.ChannelGroup, error) {
	group := &models.ChannelGroup{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, name, is_enabled, sort_order, created_at, updated_at
		FROM channel_groups WHERE id = ? AND user_id = ?`, id, userID,
	).Scan(&group.ID, &group.UserID, &group.Name, &group.IsEnabled, &group.SortOrder, &group.CreatedAt, &group.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("channel group not found: %w", err)
		}
		return nil, fmt.Errorf("getting channel group by id for user: %w", err)
	}
	return group, nil
}

func (r *ChannelGroupRepository) GetByNameForUser(ctx context.Context, name, userID string) (*models.ChannelGroup, error) {
	group := &models.ChannelGroup{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, name, is_enabled, sort_order, created_at, updated_at
		FROM channel_groups WHERE name = ? AND user_id = ?`, name, userID,
	).Scan(&group.ID, &group.UserID, &group.Name, &group.IsEnabled, &group.SortOrder, &group.CreatedAt, &group.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("channel group not found: %w", err)
		}
		return nil, fmt.Errorf("getting channel group by name for user: %w", err)
	}
	return group, nil
}

func (r *ChannelGroupRepository) UpdateForUser(ctx context.Context, group *models.ChannelGroup, userID string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		`UPDATE channel_groups SET name = ?, is_enabled = ?, sort_order = ?, updated_at = ?
		WHERE id = ? AND user_id = ?`,
		group.Name, group.IsEnabled, group.SortOrder, now, group.ID, userID,
	)
	if err != nil {
		return fmt.Errorf("updating channel group for user: %w", err)
	}
	group.UpdatedAt = now
	return nil
}

func (r *ChannelGroupRepository) DeleteForUser(ctx context.Context, id, userID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM channel_groups WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return fmt.Errorf("deleting channel group for user: %w", err)
	}
	return nil
}

func (r *ChannelGroupRepository) Update(ctx context.Context, group *models.ChannelGroup) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		`UPDATE channel_groups SET name = ?, is_enabled = ?, sort_order = ?, updated_at = ?
		WHERE id = ?`,
		group.Name, group.IsEnabled, group.SortOrder, now, group.ID,
	)
	if err != nil {
		return fmt.Errorf("updating channel group: %w", err)
	}
	group.UpdatedAt = now
	return nil
}

func (r *ChannelGroupRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM channel_groups WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deleting channel group: %w", err)
	}
	return nil
}

func (r *ChannelGroupRepository) scanRows(rows *sql.Rows) ([]models.ChannelGroup, error) {
	var groups []models.ChannelGroup
	for rows.Next() {
		var g models.ChannelGroup
		if err := rows.Scan(&g.ID, &g.UserID, &g.Name, &g.IsEnabled, &g.SortOrder, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning channel group: %w", err)
		}
		groups = append(groups, g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating channel groups: %w", err)
	}
	return groups, nil
}
