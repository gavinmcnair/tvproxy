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

const programDataCols = `id, epg_data_id, title, description, start, stop, category, episode_num, icon,
	subtitle, date, language, is_new, is_previously_shown, credits, rating, rating_icon, star_rating, sub_categories, episode_num_system`

var sqliteTimeFormats = []string{
	"2006-01-02 15:04:05 -0700 -0700",
	"2006-01-02 15:04:05 -0700",
	"2006-01-02T15:04:05Z",
	"2006-01-02 15:04:05Z",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02 15:04:05-07:00",
	"2006-01-02T15:04:05-07:00",
	time.RFC3339,
	time.RFC3339Nano,
}

func parseSQLiteTime(s string) (time.Time, error) {
	for _, f := range sqliteTimeFormats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time: %q", s)
}

type GuideProgram struct {
	ChannelID   string    `json:"channel_id"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Start       time.Time `json:"start"`
	Stop        time.Time `json:"stop"`
	Category    string    `json:"category,omitempty"`
}

type ProgramDataRepository struct {
	db *database.DB
}

func NewProgramDataRepository(db *database.DB) *ProgramDataRepository {
	return &ProgramDataRepository{db: db}
}

func (r *ProgramDataRepository) Checkpoint(ctx context.Context) {
	r.db.Checkpoint(ctx)
}

func scanProgramData(scanner interface{ Scan(...any) error }, p *models.ProgramData) error {
	return scanner.Scan(
		&p.ID, &p.EPGDataID, &p.Title, &p.Description,
		&p.Start, &p.Stop, &p.Category, &p.EpisodeNum, &p.Icon,
		&p.Subtitle, &p.Date, &p.Language, &p.IsNew, &p.IsPreviouslyShown,
		&p.Credits, &p.Rating, &p.RatingIcon, &p.StarRating, &p.SubCategories, &p.EpisodeNumSystem,
	)
}

func (r *ProgramDataRepository) Create(ctx context.Context, program *models.ProgramData) error {
	program.ID = uuid.New().String()
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO program_data (`+programDataCols+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		program.ID, program.EPGDataID, program.Title, program.Description,
		program.Start, program.Stop, program.Category,
		program.EpisodeNum, program.Icon,
		program.Subtitle, program.Date, program.Language, program.IsNew, program.IsPreviouslyShown,
		program.Credits, program.Rating, program.RatingIcon, program.StarRating, program.SubCategories, program.EpisodeNumSystem,
	)
	if err != nil {
		return fmt.Errorf("creating program data: %w", err)
	}
	return nil
}

func (r *ProgramDataRepository) List(ctx context.Context) ([]models.ProgramData, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+programDataCols+` FROM program_data ORDER BY start`,
	)
	if err != nil {
		return nil, fmt.Errorf("listing program data: %w", err)
	}
	defer rows.Close()

	var programs []models.ProgramData
	for rows.Next() {
		var p models.ProgramData
		if err := scanProgramData(rows, &p); err != nil {
			return nil, fmt.Errorf("scanning program data: %w", err)
		}
		programs = append(programs, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating program data: %w", err)
	}
	return programs, nil
}

func (r *ProgramDataRepository) ListByEPGDataID(ctx context.Context, epgDataID string) ([]models.ProgramData, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+programDataCols+` FROM program_data WHERE epg_data_id = ? ORDER BY start`, epgDataID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing program data by epg data id: %w", err)
	}
	defer rows.Close()

	var programs []models.ProgramData
	for rows.Next() {
		var p models.ProgramData
		if err := scanProgramData(rows, &p); err != nil {
			return nil, fmt.Errorf("scanning program data: %w", err)
		}
		programs = append(programs, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating program data: %w", err)
	}
	return programs, nil
}

func (r *ProgramDataRepository) ListByTimeRange(ctx context.Context, start, stop time.Time) ([]models.ProgramData, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+programDataCols+` FROM program_data WHERE start < ? AND stop > ? ORDER BY start`, stop, start,
	)
	if err != nil {
		return nil, fmt.Errorf("listing program data by time range: %w", err)
	}
	defer rows.Close()

	var programs []models.ProgramData
	for rows.Next() {
		var p models.ProgramData
		if err := scanProgramData(rows, &p); err != nil {
			return nil, fmt.Errorf("scanning program data: %w", err)
		}
		programs = append(programs, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating program data: %w", err)
	}
	return programs, nil
}

func (r *ProgramDataRepository) GetNowByChannelID(ctx context.Context, channelID string, now time.Time) (*models.ProgramData, error) {
	var p models.ProgramData
	err := r.db.QueryRowContext(ctx,
		`SELECT p.id, p.epg_data_id, p.title, p.description, p.start, p.stop, p.category, p.episode_num, p.icon,
			p.subtitle, p.date, p.language, p.is_new, p.is_previously_shown, p.credits, p.rating, p.rating_icon, p.star_rating, p.sub_categories, p.episode_num_system
		FROM program_data p
		JOIN epg_data e ON e.id = p.epg_data_id
		WHERE e.channel_id = ? AND p.start <= ? AND p.stop > ?
		ORDER BY p.start DESC LIMIT 1`,
		channelID, now, now,
	).Scan(&p.ID, &p.EPGDataID, &p.Title, &p.Description, &p.Start, &p.Stop, &p.Category, &p.EpisodeNum, &p.Icon,
		&p.Subtitle, &p.Date, &p.Language, &p.IsNew, &p.IsPreviouslyShown,
		&p.Credits, &p.Rating, &p.RatingIcon, &p.StarRating, &p.SubCategories, &p.EpisodeNumSystem)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProgramDataRepository) ListNowPlaying(ctx context.Context, now time.Time) (map[string]string, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT e.channel_id, p.title
		FROM program_data p
		JOIN epg_data e ON e.id = p.epg_data_id
		WHERE p.start <= ? AND p.stop > ?`,
		now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("listing now playing: %w", err)
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var channelID, title string
		if err := rows.Scan(&channelID, &title); err != nil {
			return nil, fmt.Errorf("scanning now playing: %w", err)
		}
		if _, exists := result[channelID]; !exists {
			result[channelID] = title
		}
	}
	return result, rows.Err()
}

func (r *ProgramDataRepository) ListForGuide(ctx context.Context, start, stop time.Time) ([]GuideProgram, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT e.channel_id, p.title, p.description, p.start, p.stop, p.category
		FROM program_data p
		JOIN epg_data e ON e.id = p.epg_data_id
		WHERE p.start < ? AND p.stop > ?
		ORDER BY e.channel_id, p.start`,
		stop, start,
	)
	if err != nil {
		return nil, fmt.Errorf("listing guide programs: %w", err)
	}
	defer rows.Close()

	var programs []GuideProgram
	for rows.Next() {
		var g GuideProgram
		var startStr, stopStr string
		if err := rows.Scan(&g.ChannelID, &g.Title, &g.Description, &startStr, &stopStr, &g.Category); err != nil {
			return nil, fmt.Errorf("scanning guide program: %w", err)
		}
		var parseErr error
		if g.Start, parseErr = parseSQLiteTime(startStr); parseErr != nil {
			return nil, fmt.Errorf("parsing guide start time: %w", parseErr)
		}
		if g.Stop, parseErr = parseSQLiteTime(stopStr); parseErr != nil {
			return nil, fmt.Errorf("parsing guide stop time: %w", parseErr)
		}
		programs = append(programs, g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating guide programs: %w", err)
	}
	return programs, nil
}

func (r *ProgramDataRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM program_data WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deleting program data: %w", err)
	}
	return nil
}

func (r *ProgramDataRepository) DeleteByEPGDataID(ctx context.Context, epgDataID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM program_data WHERE epg_data_id = ?`, epgDataID)
	if err != nil {
		return fmt.Errorf("deleting program data by epg data id: %w", err)
	}
	return nil
}

func (r *ProgramDataRepository) BulkCreate(ctx context.Context, programs []models.ProgramData) error {
	return r.db.InTx(ctx, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx,
			`INSERT INTO program_data (`+programDataCols+`)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		)
		if err != nil {
			return fmt.Errorf("preparing statement: %w", err)
		}
		defer stmt.Close()

		for i := range programs {
			programs[i].ID = uuid.New().String()
			if _, err := stmt.ExecContext(ctx,
				programs[i].ID, programs[i].EPGDataID, programs[i].Title, programs[i].Description,
				programs[i].Start, programs[i].Stop, programs[i].Category,
				programs[i].EpisodeNum, programs[i].Icon,
				programs[i].Subtitle, programs[i].Date, programs[i].Language,
				programs[i].IsNew, programs[i].IsPreviouslyShown,
				programs[i].Credits, programs[i].Rating, programs[i].RatingIcon,
				programs[i].StarRating, programs[i].SubCategories, programs[i].EpisodeNumSystem,
			); err != nil {
				return fmt.Errorf("inserting program data %d: %w", i, err)
			}
		}
		return nil
	})
}
