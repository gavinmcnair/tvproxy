package dash

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	"github.com/rs/zerolog"
)

func TempDir() string {
	return filepath.Join(os.TempDir(), "tvproxy-dash")
}

func ChannelDir(channelID string) string {
	return filepath.Join(TempDir(), channelID)
}

type Manager struct {
	remuxers   map[string]*Remuxer
	segmenters map[string]*MP4Segmenter
	mu         sync.Mutex
	log        zerolog.Logger
}

func NewManager(log zerolog.Logger) *Manager {
	return &Manager{
		remuxers:   make(map[string]*Remuxer),
		segmenters: make(map[string]*MP4Segmenter),
		log:        log.With().Str("component", "dash_manager").Logger(),
	}
}

func (m *Manager) GetOrStart(ctx context.Context, channelID, inputPath, outputDir string, isVOD bool, duration float64) (*Remuxer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if r, ok := m.remuxers[channelID]; ok {
		if !r.IsDone() && r.inputPath == inputPath {
			return r, nil
		}
		r.Stop()
		os.RemoveAll(r.OutputDir())
		delete(m.remuxers, channelID)
	}

	r := NewRemuxer(inputPath, outputDir, isVOD, duration, m.log)
	if err := r.Start(ctx); err != nil {
		return nil, err
	}

	m.remuxers[channelID] = r
	m.log.Info().Str("channel_id", channelID).Msg("dash remuxer started")
	return r, nil
}

func (m *Manager) GetOrStartMP4(ctx context.Context, channelID, filePath string, duration float64) (*MP4Segmenter, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if s, ok := m.segmenters[channelID]; ok {
		if !s.IsDone() {
			return s, nil
		}
		s.Stop()
		delete(m.segmenters, channelID)
	}

	s := NewMP4Segmenter(filePath, duration, m.log)
	if err := s.Start(ctx); err != nil {
		return nil, err
	}

	m.segmenters[channelID] = s
	m.log.Info().Str("channel_id", channelID).Msg("mp4 segmenter started")
	return s, nil
}

func (m *Manager) GetMP4Segmenter(channelID string) *MP4Segmenter {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.segmenters[channelID]
}

func (m *Manager) Stop(channelID string) {
	m.mu.Lock()
	r, rOK := m.remuxers[channelID]
	if rOK {
		delete(m.remuxers, channelID)
	}
	s, sOK := m.segmenters[channelID]
	if sOK {
		delete(m.segmenters, channelID)
	}
	m.mu.Unlock()

	if rOK {
		r.Stop()
		os.RemoveAll(r.OutputDir())
		m.log.Info().Str("channel_id", channelID).Msg("dash remuxer stopped")
	}
	if sOK {
		s.Stop()
		m.log.Info().Str("channel_id", channelID).Msg("mp4 segmenter stopped")
	}
}

func (m *Manager) Shutdown() {
	m.mu.Lock()
	allRemuxers := make(map[string]*Remuxer, len(m.remuxers))
	for k, v := range m.remuxers {
		allRemuxers[k] = v
	}
	m.remuxers = make(map[string]*Remuxer)

	allSegmenters := make(map[string]*MP4Segmenter, len(m.segmenters))
	for k, v := range m.segmenters {
		allSegmenters[k] = v
	}
	m.segmenters = make(map[string]*MP4Segmenter)
	m.mu.Unlock()

	for _, r := range allRemuxers {
		r.Stop()
		os.RemoveAll(r.OutputDir())
	}
	for _, s := range allSegmenters {
		s.Stop()
	}
	m.log.Info().
		Int("remuxers", len(allRemuxers)).
		Int("segmenters", len(allSegmenters)).
		Msg("dash manager shutdown complete")
}
