package worker

import (
	"context"
	"time"

	"github.com/rs/zerolog"
)

type RecordingScheduler interface {
	Tick(ctx context.Context)
}

type SchedulerWorker struct {
	service  RecordingScheduler
	interval time.Duration
	log      zerolog.Logger
}

func NewSchedulerWorker(service RecordingScheduler, interval time.Duration, log zerolog.Logger) *SchedulerWorker {
	return &SchedulerWorker{
		service:  service,
		interval: interval,
		log:      log,
	}
}

func (w *SchedulerWorker) Run(ctx context.Context) {
	w.service.Tick(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.service.Tick(ctx)
		}
	}
}
