package store

import (
	"context"
	"testing"

	"github.com/rs/zerolog"

	"github.com/gavinmcnair/tvproxy/pkg/models"
)

func TestStreamStore_UpdateTMDBID(t *testing.T) {
	log := zerolog.Nop()
	s := NewStreamStore(t.TempDir()+"/streams.gob", log)

	ctx := context.Background()

	err := s.BulkUpsert(ctx, []models.Stream{
		{ID: "stream-1", Name: "The Matrix", VODType: "movie", ContentHash: "h1", IsActive: true},
		{ID: "stream-2", Name: "Breaking Bad S01E01", VODType: "series", ContentHash: "h2", IsActive: true},
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := s.UpdateTMDBID(ctx, "stream-1", 603); err != nil {
		t.Fatal(err)
	}

	st, err := s.GetByID(ctx, "stream-1")
	if err != nil {
		t.Fatal(err)
	}
	if st.TMDBID != 603 {
		t.Errorf("got TMDBID %d, want 603", st.TMDBID)
	}

	st2, _ := s.GetByID(ctx, "stream-2")
	if st2.TMDBID != 0 {
		t.Errorf("stream-2 TMDBID should be 0, got %d", st2.TMDBID)
	}

	if err := s.UpdateTMDBID(ctx, "nonexistent", 123); err == nil {
		t.Fatal("expected error for nonexistent stream")
	}
}
