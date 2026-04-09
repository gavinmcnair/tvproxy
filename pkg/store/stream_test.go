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

func TestStreamStore_PreserveTMDBIDOnUpsert(t *testing.T) {
	log := zerolog.Nop()
	s := NewStreamStore(t.TempDir()+"/streams.gob", log)
	ctx := context.Background()

	s.BulkUpsert(ctx, []models.Stream{
		{ID: "s1", Name: "Movie", M3UAccountID: "acc1", ContentHash: "h1", IsActive: true},
	})
	s.UpdateTMDBID(ctx, "s1", 603)

	s.BulkUpsert(ctx, []models.Stream{
		{ID: "s1", Name: "Movie", M3UAccountID: "acc1", ContentHash: "h1", IsActive: true},
	})

	st, _ := s.GetByID(ctx, "s1")
	if st.TMDBID != 603 {
		t.Errorf("TMDBID lost on upsert: got %d, want 603", st.TMDBID)
	}
}

func TestStreamStore_SetTMDBManual(t *testing.T) {
	log := zerolog.Nop()
	s := NewStreamStore(t.TempDir()+"/streams.gob", log)
	ctx := context.Background()

	s.BulkUpsert(ctx, []models.Stream{
		{ID: "s1", Name: "Movie", M3UAccountID: "acc1", ContentHash: "h1", IsActive: true},
	})

	s.SetTMDBManual(ctx, "s1", 999)

	st, _ := s.GetByID(ctx, "s1")
	if st.TMDBID != 999 || !st.TMDBManual {
		t.Errorf("SetTMDBManual: got TMDBID=%d Manual=%v, want 999/true", st.TMDBID, st.TMDBManual)
	}
}

func TestStreamStore_ClearAutoTMDB(t *testing.T) {
	log := zerolog.Nop()
	s := NewStreamStore(t.TempDir()+"/streams.gob", log)
	ctx := context.Background()

	s.BulkUpsert(ctx, []models.Stream{
		{ID: "s1", Name: "Auto Match", M3UAccountID: "acc1", ContentHash: "h1", IsActive: true},
		{ID: "s2", Name: "Manual Match", M3UAccountID: "acc1", ContentHash: "h2", IsActive: true},
		{ID: "s3", Name: "Other Account", M3UAccountID: "acc2", ContentHash: "h3", IsActive: true},
	})

	s.UpdateTMDBID(ctx, "s1", 100)
	s.SetTMDBManual(ctx, "s2", 200)
	s.UpdateTMDBID(ctx, "s3", 300)

	s.ClearAutoTMDBByAccountID(ctx, "acc1")

	s1, _ := s.GetByID(ctx, "s1")
	if s1.TMDBID != 0 {
		t.Errorf("auto match should be cleared: got %d", s1.TMDBID)
	}

	s2, _ := s.GetByID(ctx, "s2")
	if s2.TMDBID != 200 {
		t.Errorf("manual match should be preserved: got %d", s2.TMDBID)
	}

	s3, _ := s.GetByID(ctx, "s3")
	if s3.TMDBID != 300 {
		t.Errorf("other account should be untouched: got %d", s3.TMDBID)
	}
}
