package worker

import (
	"context"
	"time"

	"github.com/rs/zerolog"
)

type MultiWireGuardSyncer interface {
	SyncProfiles(ctx context.Context)
	RunHealthChecks(ctx context.Context)
}

type MultiWireGuardWorker struct {
	syncer          MultiWireGuardSyncer
	syncInterval    time.Duration
	healthInterval  time.Duration
	log             zerolog.Logger
}

func NewMultiWireGuardWorker(syncer MultiWireGuardSyncer, syncInterval, healthInterval time.Duration, log zerolog.Logger) *MultiWireGuardWorker {
	if syncInterval <= 0 {
		syncInterval = 30 * time.Second
	}
	if healthInterval <= 0 {
		healthInterval = 60 * time.Second
	}
	return &MultiWireGuardWorker{
		syncer:         syncer,
		syncInterval:   syncInterval,
		healthInterval: healthInterval,
		log:            log.With().Str("worker", "wireguard_multi").Logger(),
	}
}

func (w *MultiWireGuardWorker) Run(ctx context.Context) {
	syncTicker := time.NewTicker(w.syncInterval)
	healthTicker := time.NewTicker(w.healthInterval)
	defer syncTicker.Stop()
	defer healthTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-syncTicker.C:
			w.syncer.SyncProfiles(ctx)
		case <-healthTicker.C:
			w.syncer.RunHealthChecks(ctx)
		}
	}
}
