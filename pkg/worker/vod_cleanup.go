package worker

import (
	"context"
	"time"

	"github.com/rs/zerolog"
)

type VODCleaner interface {
	CleanupExpired()
}

type VODCleanupWorker struct {
	service  VODCleaner
	interval time.Duration
	log      zerolog.Logger
}

func NewVODCleanupWorker(service VODCleaner, interval time.Duration, log zerolog.Logger) *VODCleanupWorker {
	return &VODCleanupWorker{
		service:  service,
		interval: interval,
		log:      log,
	}
}

func (w *VODCleanupWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.service.CleanupExpired()
		}
	}
}
