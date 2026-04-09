package store

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/gavinmcnair/tvproxy/pkg/models"
)

type StreamIndex struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Group         string `json:"group"`
	M3UAccountID  string `json:"m3u_account_id"`
	SatIPSourceID string `json:"satip_source_id,omitempty"`
	VODType       string `json:"vod_type,omitempty"`
	VODSeries     string `json:"vod_series,omitempty"`
	VODCollection string `json:"vod_collection,omitempty"`
	VODSeason     int    `json:"vod_season,omitempty"`
	VODSeasonName string `json:"vod_season_name,omitempty"`
	VODEpisode    int    `json:"vod_episode,omitempty"`
	VODYear       int    `json:"vod_year,omitempty"`
	TMDBID        int    `json:"tmdb_id,omitempty"`
	TMDBManual    bool   `json:"tmdb_manual,omitempty"`
	UseWireGuard  bool   `json:"use_wireguard,omitempty"`
	IsActive      bool   `json:"is_active"`
	Logo          string `json:"logo,omitempty"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type IndexedStreamStore struct {
	mu    sync.RWMutex
	index map[string]StreamIndex
	rev   *Revision

	dataDir  string
	legacyPath string
	log      zerolog.Logger
}

func NewIndexedStreamStore(dataDir string, legacyGobPath string, log zerolog.Logger) *IndexedStreamStore {
	s := &IndexedStreamStore{
		index:      make(map[string]StreamIndex),
		rev:        NewRevision(),
		dataDir:    dataDir,
		legacyPath: legacyGobPath,
		log:        log.With().Str("store", "stream_indexed").Logger(),
	}
	return s
}

func (s *IndexedStreamStore) ETag() string {
	return s.rev.ETag()
}

func (s *IndexedStreamStore) indexFromStream(st models.Stream) StreamIndex {
	return StreamIndex{
		ID:            st.ID,
		Name:          st.Name,
		Group:         st.Group,
		M3UAccountID:  st.M3UAccountID,
		SatIPSourceID: st.SatIPSourceID,
		VODType:       st.VODType,
		VODSeries:     st.VODSeries,
		VODCollection: st.VODCollection,
		VODSeason:     st.VODSeason,
		VODSeasonName: st.VODSeasonName,
		VODEpisode:    st.VODEpisode,
		VODYear:       st.VODYear,
		TMDBID:        st.TMDBID,
		TMDBManual:    st.TMDBManual,
		UseWireGuard:  st.UseWireGuard,
		IsActive:      st.IsActive,
		Logo:          st.Logo,
		CreatedAt:     st.CreatedAt,
		UpdatedAt:     st.UpdatedAt,
	}
}

func (s *IndexedStreamStore) summaryFromIndex(idx StreamIndex) models.StreamSummary {
	return models.StreamSummary{
		ID:            idx.ID,
		M3UAccountID:  idx.M3UAccountID,
		SatIPSourceID: idx.SatIPSourceID,
		Name:          idx.Name,
		Group:         idx.Group,
		Logo:          idx.Logo,
		VODType:       idx.VODType,
		VODSeries:     idx.VODSeries,
		VODSeason:     idx.VODSeason,
		VODSeasonName: idx.VODSeasonName,
		VODEpisode:    idx.VODEpisode,
		VODYear:       idx.VODYear,
		TMDBID:        idx.TMDBID,
	}
}

func (s *IndexedStreamStore) streamPath(id string) string {
	return filepath.Join(s.dataDir, "streams", id+".json")
}

func (s *IndexedStreamStore) indexPath() string {
	return filepath.Join(s.dataDir, "stream_index.json")
}

func (s *IndexedStreamStore) readStream(id string) (*models.Stream, error) {
	data, err := os.ReadFile(s.streamPath(id))
	if err != nil {
		return nil, err
	}
	var st models.Stream
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, err
	}
	return &st, nil
}

func (s *IndexedStreamStore) writeStream(st models.Stream) error {
	dir := filepath.Join(s.dataDir, "streams")
	os.MkdirAll(dir, 0755)
	data, err := json.Marshal(st)
	if err != nil {
		return err
	}
	return os.WriteFile(s.streamPath(st.ID), data, 0644)
}

func (s *IndexedStreamStore) deleteStreamFile(id string) {
	os.Remove(s.streamPath(id))
}

func (s *IndexedStreamStore) List(_ context.Context) ([]models.Stream, error) {
	s.mu.RLock()
	ids := make([]StreamIndex, 0, len(s.index))
	for _, idx := range s.index {
		ids = append(ids, idx)
	}
	s.mu.RUnlock()

	sort.Slice(ids, func(i, j int) bool {
		return ids[i].CreatedAt.Before(ids[j].CreatedAt)
	})

	items := make([]models.Stream, 0, len(ids))
	for _, idx := range ids {
		st, err := s.readStream(idx.ID)
		if err != nil {
			continue
		}
		items = append(items, *st)
	}
	return items, nil
}

func (s *IndexedStreamStore) ListSummaries(_ context.Context) ([]models.StreamSummary, error) {
	s.mu.RLock()
	indices := make([]StreamIndex, 0, len(s.index))
	for _, idx := range s.index {
		indices = append(indices, idx)
	}
	s.mu.RUnlock()

	sort.Slice(indices, func(i, j int) bool {
		return indices[i].Name < indices[j].Name
	})

	summaries := make([]models.StreamSummary, len(indices))
	for i, idx := range indices {
		summaries[i] = s.summaryFromIndex(idx)
	}
	return summaries, nil
}

func (s *IndexedStreamStore) ListByAccountID(_ context.Context, accountID string) ([]models.Stream, error) {
	s.mu.RLock()
	var matching []StreamIndex
	for _, idx := range s.index {
		if idx.M3UAccountID == accountID {
			matching = append(matching, idx)
		}
	}
	s.mu.RUnlock()

	sort.Slice(matching, func(i, j int) bool {
		return matching[i].CreatedAt.Before(matching[j].CreatedAt)
	})

	var items []models.Stream
	for _, idx := range matching {
		if st, err := s.readStream(idx.ID); err == nil {
			items = append(items, *st)
		}
	}
	return items, nil
}

func (s *IndexedStreamStore) ListBySatIPSourceID(_ context.Context, sourceID string) ([]models.Stream, error) {
	s.mu.RLock()
	var matching []StreamIndex
	for _, idx := range s.index {
		if idx.SatIPSourceID == sourceID {
			matching = append(matching, idx)
		}
	}
	s.mu.RUnlock()

	sort.Slice(matching, func(i, j int) bool {
		return matching[i].CreatedAt.Before(matching[j].CreatedAt)
	})

	var items []models.Stream
	for _, idx := range matching {
		if st, err := s.readStream(idx.ID); err == nil {
			items = append(items, *st)
		}
	}
	return items, nil
}

func (s *IndexedStreamStore) GetByID(_ context.Context, id string) (*models.Stream, error) {
	s.mu.RLock()
	_, exists := s.index[id]
	s.mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("stream not found: %s", id)
	}
	st, err := s.readStream(id)
	if err != nil {
		return nil, fmt.Errorf("stream not found: %s", id)
	}
	return st, nil
}

func (s *IndexedStreamStore) BulkUpsert(_ context.Context, streams []models.Stream) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for _, st := range streams {
		if existing, exists := s.index[st.ID]; exists {
			st.CreatedAt = existing.CreatedAt
			if existing.TMDBID > 0 {
				st.TMDBID = existing.TMDBID
				st.TMDBManual = existing.TMDBManual
			}
		} else {
			st.CreatedAt = now
		}
		st.UpdatedAt = now
		s.index[st.ID] = s.indexFromStream(st)
		s.writeStream(st)
	}
	s.rev.Bump()
	return nil
}

func (s *IndexedStreamStore) DeleteStaleByAccountID(_ context.Context, accountID string, keepIDs []string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	keep := make(map[string]struct{}, len(keepIDs))
	for _, id := range keepIDs {
		keep[id] = struct{}{}
	}

	var deleted []string
	for id, idx := range s.index {
		if idx.M3UAccountID != accountID {
			continue
		}
		if _, shouldKeep := keep[id]; !shouldKeep {
			delete(s.index, id)
			s.deleteStreamFile(id)
			deleted = append(deleted, id)
		}
	}
	if len(deleted) > 0 {
		s.rev.Bump()
	}
	return deleted, nil
}

func (s *IndexedStreamStore) DeleteByAccountID(_ context.Context, accountID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, idx := range s.index {
		if idx.M3UAccountID == accountID {
			delete(s.index, id)
			s.deleteStreamFile(id)
		}
	}
	s.rev.Bump()
	return nil
}

func (s *IndexedStreamStore) DeleteStaleBySatIPSourceID(_ context.Context, sourceID string, keepIDs []string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	keep := make(map[string]struct{}, len(keepIDs))
	for _, id := range keepIDs {
		keep[id] = struct{}{}
	}

	var deleted []string
	for id, idx := range s.index {
		if idx.SatIPSourceID != sourceID {
			continue
		}
		if _, shouldKeep := keep[id]; !shouldKeep {
			delete(s.index, id)
			s.deleteStreamFile(id)
			deleted = append(deleted, id)
		}
	}
	if len(deleted) > 0 {
		s.rev.Bump()
	}
	return deleted, nil
}

func (s *IndexedStreamStore) DeleteBySatIPSourceID(_ context.Context, sourceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, idx := range s.index {
		if idx.SatIPSourceID == sourceID {
			delete(s.index, id)
			s.deleteStreamFile(id)
		}
	}
	s.rev.Bump()
	return nil
}

func (s *IndexedStreamStore) DeleteOrphanedM3UStreams(_ context.Context, knownAccountIDs []string) ([]string, error) {
	known := make(map[string]struct{}, len(knownAccountIDs))
	for _, id := range knownAccountIDs {
		known[id] = struct{}{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var deleted []string
	for id, idx := range s.index {
		if idx.M3UAccountID == "" {
			continue
		}
		if _, ok := known[idx.M3UAccountID]; !ok {
			delete(s.index, id)
			s.deleteStreamFile(id)
			deleted = append(deleted, id)
		}
	}
	if len(deleted) > 0 {
		s.rev.Bump()
	}
	return deleted, nil
}

func (s *IndexedStreamStore) DeleteOrphanedSatIPStreams(_ context.Context, knownSourceIDs []string) ([]string, error) {
	known := make(map[string]struct{}, len(knownSourceIDs))
	for _, id := range knownSourceIDs {
		known[id] = struct{}{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var deleted []string
	for id, idx := range s.index {
		if idx.SatIPSourceID == "" {
			continue
		}
		if _, ok := known[idx.SatIPSourceID]; !ok {
			delete(s.index, id)
			s.deleteStreamFile(id)
			deleted = append(deleted, id)
		}
	}
	if len(deleted) > 0 {
		s.rev.Bump()
	}
	return deleted, nil
}

func (s *IndexedStreamStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	delete(s.index, id)
	s.deleteStreamFile(id)
	s.rev.Bump()
	s.mu.Unlock()
	return nil
}

func (s *IndexedStreamStore) UpdateTMDBID(_ context.Context, id string, tmdbID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, ok := s.index[id]
	if !ok {
		return fmt.Errorf("stream not found: %s", id)
	}
	idx.TMDBID = tmdbID
	idx.UpdatedAt = time.Now()
	s.index[id] = idx

	if st, err := s.readStream(id); err == nil {
		st.TMDBID = tmdbID
		st.UpdatedAt = idx.UpdatedAt
		s.writeStream(*st)
	}
	return nil
}

func (s *IndexedStreamStore) SetTMDBManual(_ context.Context, id string, tmdbID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, ok := s.index[id]
	if !ok {
		return fmt.Errorf("stream not found: %s", id)
	}
	idx.TMDBID = tmdbID
	idx.TMDBManual = true
	idx.UpdatedAt = time.Now()
	s.index[id] = idx

	if st, err := s.readStream(id); err == nil {
		st.TMDBID = tmdbID
		st.TMDBManual = true
		st.UpdatedAt = idx.UpdatedAt
		s.writeStream(*st)
	}
	return nil
}

func (s *IndexedStreamStore) ClearAutoTMDBByAccountID(_ context.Context, accountID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, idx := range s.index {
		if idx.M3UAccountID == accountID && idx.TMDBID > 0 && !idx.TMDBManual {
			idx.TMDBID = 0
			s.index[id] = idx

			if st, err := s.readStream(id); err == nil {
				st.TMDBID = 0
				s.writeStream(*st)
			}
		}
	}
	return nil
}

func (s *IndexedStreamStore) UpdateWireGuardByAccountID(_ context.Context, accountID string, useWireGuard bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, idx := range s.index {
		if idx.M3UAccountID == accountID {
			idx.UseWireGuard = useWireGuard
			s.index[id] = idx

			if st, err := s.readStream(id); err == nil {
				st.UseWireGuard = useWireGuard
				s.writeStream(*st)
			}
		}
	}
	return nil
}

func (s *IndexedStreamStore) Clear() error {
	s.mu.Lock()
	for id := range s.index {
		s.deleteStreamFile(id)
	}
	s.index = make(map[string]StreamIndex)
	s.rev.Bump()
	s.mu.Unlock()
	return nil
}

func (s *IndexedStreamStore) Save() error {
	s.mu.RLock()
	indexCopy := make(map[string]StreamIndex, len(s.index))
	for k, v := range s.index {
		indexCopy[k] = v
	}
	s.mu.RUnlock()

	data, err := json.MarshalIndent(indexCopy, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.indexPath() + ".tmp"
	if err := os.MkdirAll(filepath.Dir(s.indexPath()), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, s.indexPath())
}

func (s *IndexedStreamStore) Load() error {
	if data, err := os.ReadFile(s.indexPath()); err == nil {
		s.mu.Lock()
		json.Unmarshal(data, &s.index)
		if s.index == nil {
			s.index = make(map[string]StreamIndex)
		}
		s.mu.Unlock()
		s.log.Info().Int("count", len(s.index)).Msg("loaded stream index")
		return nil
	}

	if s.legacyPath == "" {
		return nil
	}
	f, err := os.Open(s.legacyPath)
	if err != nil {
		return nil
	}
	defer f.Close()

	var legacy map[string]models.Stream
	if err := gob.NewDecoder(f).Decode(&legacy); err != nil {
		s.log.Error().Err(err).Msg("failed to decode legacy gob")
		return nil
	}

	s.mu.Lock()
	s.index = make(map[string]StreamIndex, len(legacy))
	for _, st := range legacy {
		s.index[st.ID] = s.indexFromStream(st)
		s.writeStream(st)
	}
	s.mu.Unlock()

	s.Save()
	s.log.Info().Int("count", len(legacy)).Msg("migrated legacy gob to indexed store")
	return nil
}
