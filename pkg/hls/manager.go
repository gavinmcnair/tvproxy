package hls

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	baseDir  string
	log      zerolog.Logger
}

func NewManager(baseDir string, log zerolog.Logger) *Manager {
	return &Manager{
		sessions: make(map[string]*Session),
		baseDir:  baseDir,
		log:      log.With().Str("component", "hls").Logger(),
	}
}

func (m *Manager) GetOrCreateSession(sessionID, streamURL string, segmentLength int, durationTicks int64, isLive bool, profile ProfileSettings) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	if sess, ok := m.sessions[sessionID]; ok {
		sess.Touch()
		return sess
	}

	outputDir := filepath.Join(m.baseDir, sessionID)
	sess := NewSession(sessionID, streamURL, outputDir, segmentLength, durationTicks, isLive, profile, m.log)
	m.sessions[sessionID] = sess
	return sess
}

func (m *Manager) GetSession(sessionID string) *Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if sess, ok := m.sessions[sessionID]; ok {
		sess.Touch()
		return sess
	}
	return nil
}

func (m *Manager) RequestSegment(ctx context.Context, sess *Session, segmentIndex int, runtimeTicks int64) error {
	sess.Touch()

	segPath := sess.SegmentPath(segmentIndex)
	if _, err := os.Stat(segPath); err == nil {
		nextPath := sess.SegmentPath(segmentIndex + 1)
		if sess.IsDone() {
			return nil
		}
		if _, err := os.Stat(nextPath); err == nil {
			return nil
		}
	}

	currentIndex := sess.CurrentTranscodeIndex()
	segmentGap := 24 / sess.SegmentLength

	needsNewTranscode := currentIndex == -1 ||
		segmentIndex < currentIndex ||
		segmentIndex-currentIndex > segmentGap

	if needsNewTranscode {
		m.log.Info().
			Str("session", sess.ID).
			Int("segment", segmentIndex).
			Int("current_index", currentIndex).
			Int64("runtime_ticks", runtimeTicks).
			Msg("starting new transcode for segment")

		if err := sess.StartTranscode(context.Background(), segmentIndex, runtimeTicks); err != nil {
			return err
		}
	}

	return sess.WaitForSegment(segmentIndex, 30*time.Second)
}

func (m *Manager) StopSession(sessionID string) {
	m.mu.Lock()
	sess, ok := m.sessions[sessionID]
	if ok {
		delete(m.sessions, sessionID)
	}
	m.mu.Unlock()

	if ok {
		sess.Stop()
		os.RemoveAll(sess.OutputDir)
		m.log.Info().Str("session", sessionID).Msg("hls session stopped")
	}
}

func (m *Manager) StopAll() {
	m.mu.Lock()
	all := make(map[string]*Session, len(m.sessions))
	for k, v := range m.sessions {
		all[k] = v
	}
	m.sessions = make(map[string]*Session)
	m.mu.Unlock()

	for id, sess := range all {
		sess.Stop()
		os.RemoveAll(sess.OutputDir)
		m.log.Info().Str("session", id).Msg("hls session stopped")
	}
}

func (m *Manager) CleanupIdle(maxIdle time.Duration) {
	m.mu.Lock()
	var toRemove []string
	for id, sess := range m.sessions {
		if sess.IdleSince() > maxIdle {
			toRemove = append(toRemove, id)
		}
	}
	var stopped []*Session
	for _, id := range toRemove {
		if sess, ok := m.sessions[id]; ok {
			stopped = append(stopped, sess)
			delete(m.sessions, id)
		}
	}
	m.mu.Unlock()

	for _, sess := range stopped {
		sess.Stop()
		os.RemoveAll(sess.OutputDir)
		m.log.Info().Str("session", sess.ID).Msg("hls idle session cleaned")
	}
}

func (m *Manager) StartCleanupWorker(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			m.StopAll()
			return
		case <-ticker.C:
			m.CleanupIdle(5 * time.Minute)
		}
	}
}

func TempDir() string {
	return filepath.Join(os.TempDir(), "tvproxy-hls")
}
