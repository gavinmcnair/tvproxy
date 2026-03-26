package database

import (
	"context"
	"database/sql"
)

type execContext interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func seedData(_ context.Context, _ execContext) error                    { return nil }
func seedRecordingProfile(_ context.Context, _ *sql.DB) error           { return nil }
func seedCopyProfile(_ context.Context, _ *sql.DB) error                { return nil }
func updateRecordingProfileAV1(_ context.Context, _ *sql.DB) error      { return nil }
