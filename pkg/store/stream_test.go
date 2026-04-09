package store

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/gavinmcnair/tvproxy/pkg/models"
)

type storeFactory func(t *testing.T) StreamStore

func newMapStore(t *testing.T) StreamStore {
	return NewStreamStore(t.TempDir()+"/streams.gob", zerolog.Nop())
}

func newIndexedStore(t *testing.T) StreamStore {
	dir := t.TempDir()
	return NewIndexedStreamStore(dir, "", zerolog.Nop())
}

func seedStore(t *testing.T, s StreamStore) {
	t.Helper()
	ctx := context.Background()
	s.BulkUpsert(ctx, []models.Stream{
		{ID: "m3u-1", Name: "BBC One", M3UAccountID: "acc1", Group: "Freeview", ContentHash: "h1", IsActive: true},
		{ID: "m3u-2", Name: "ITV", M3UAccountID: "acc1", Group: "Freeview", ContentHash: "h2", IsActive: true},
		{ID: "m3u-3", Name: "Sky Sports", M3UAccountID: "acc2", Group: "Sports", ContentHash: "h3", IsActive: true},
		{ID: "sat-1", Name: "Channel 4", SatIPSourceID: "src1", Group: "DVB-T", ContentHash: "h4", IsActive: true},
		{ID: "sat-2", Name: "Channel 5", SatIPSourceID: "src1", Group: "DVB-T", ContentHash: "h5", IsActive: true},
		{ID: "sat-3", Name: "Dave", SatIPSourceID: "src2", Group: "DVB-T", ContentHash: "h6", IsActive: true},
		{ID: "vod-1", Name: "The Matrix (1999)", M3UAccountID: "acc3", VODType: "movie", ContentHash: "h7", IsActive: true},
		{ID: "vod-2", Name: "Breaking Bad S01E01", M3UAccountID: "acc3", VODType: "series", VODSeries: "Breaking Bad", VODSeason: 1, VODEpisode: 1, ContentHash: "h8", IsActive: true},
	})
}

func runStreamStoreTests(t *testing.T, factory storeFactory) {
	t.Run("List", func(t *testing.T) {
		s := factory(t)
		seedStore(t, s)
		items, err := s.List(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 8 {
			t.Errorf("got %d items, want 8", len(items))
		}
		for i := 1; i < len(items); i++ {
			if items[i].CreatedAt.Before(items[i-1].CreatedAt) {
				t.Error("List should be sorted by CreatedAt")
				break
			}
		}
	})

	t.Run("ListSummaries", func(t *testing.T) {
		s := factory(t)
		seedStore(t, s)
		summaries, err := s.ListSummaries(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if len(summaries) != 8 {
			t.Errorf("got %d summaries, want 8", len(summaries))
		}
		for i := 1; i < len(summaries); i++ {
			if summaries[i].Name < summaries[i-1].Name {
				t.Error("ListSummaries should be sorted by Name")
				break
			}
		}
		found := false
		for _, sm := range summaries {
			if sm.ID == "vod-2" {
				found = true
				if sm.VODSeries != "Breaking Bad" {
					t.Errorf("summary VODSeries = %q, want Breaking Bad", sm.VODSeries)
				}
				if sm.VODSeason != 1 || sm.VODEpisode != 1 {
					t.Errorf("summary season/episode = %d/%d, want 1/1", sm.VODSeason, sm.VODEpisode)
				}
			}
		}
		if !found {
			t.Error("vod-2 not found in summaries")
		}
	})

	t.Run("GetByID", func(t *testing.T) {
		s := factory(t)
		seedStore(t, s)
		ctx := context.Background()
		st, err := s.GetByID(ctx, "m3u-1")
		if err != nil {
			t.Fatal(err)
		}
		if st.Name != "BBC One" {
			t.Errorf("got name %q, want BBC One", st.Name)
		}
		_, err = s.GetByID(ctx, "nonexistent")
		if err == nil {
			t.Error("expected error for nonexistent ID")
		}
	})

	t.Run("ListByAccountID", func(t *testing.T) {
		s := factory(t)
		seedStore(t, s)
		items, err := s.ListByAccountID(context.Background(), "acc1")
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 2 {
			t.Errorf("got %d items for acc1, want 2", len(items))
		}
		for _, item := range items {
			if item.M3UAccountID != "acc1" {
				t.Errorf("got account %q, want acc1", item.M3UAccountID)
			}
		}
		empty, _ := s.ListByAccountID(context.Background(), "nonexistent")
		if len(empty) != 0 {
			t.Errorf("got %d items for nonexistent account, want 0", len(empty))
		}
	})

	t.Run("ListBySatIPSourceID", func(t *testing.T) {
		s := factory(t)
		seedStore(t, s)
		items, err := s.ListBySatIPSourceID(context.Background(), "src1")
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 2 {
			t.Errorf("got %d items for src1, want 2", len(items))
		}
	})

	t.Run("BulkUpsert_NewItems", func(t *testing.T) {
		s := factory(t)
		ctx := context.Background()
		s.BulkUpsert(ctx, []models.Stream{
			{ID: "s1", Name: "Stream 1", ContentHash: "h1"},
			{ID: "s2", Name: "Stream 2", ContentHash: "h2"},
		})
		items, _ := s.List(ctx)
		if len(items) != 2 {
			t.Fatalf("got %d items, want 2", len(items))
		}
		if items[0].CreatedAt.IsZero() {
			t.Error("CreatedAt should be set")
		}
	})

	t.Run("BulkUpsert_UpdateExisting", func(t *testing.T) {
		s := factory(t)
		ctx := context.Background()
		s.BulkUpsert(ctx, []models.Stream{{ID: "s1", Name: "Original", ContentHash: "h1"}})
		st1, _ := s.GetByID(ctx, "s1")
		origCreated := st1.CreatedAt
		time.Sleep(time.Millisecond)
		s.BulkUpsert(ctx, []models.Stream{{ID: "s1", Name: "Updated", ContentHash: "h1"}})
		st2, _ := s.GetByID(ctx, "s1")
		if st2.Name != "Updated" {
			t.Errorf("name not updated: got %q", st2.Name)
		}
		if !st2.CreatedAt.Equal(origCreated) {
			t.Error("CreatedAt should be preserved on update")
		}
		if !st2.UpdatedAt.After(origCreated) {
			t.Error("UpdatedAt should be newer")
		}
	})

	t.Run("BulkUpsert_PreservesTMDBID", func(t *testing.T) {
		s := factory(t)
		ctx := context.Background()
		s.BulkUpsert(ctx, []models.Stream{{ID: "s1", Name: "Movie", ContentHash: "h1"}})
		s.UpdateTMDBID(ctx, "s1", 603)
		s.BulkUpsert(ctx, []models.Stream{{ID: "s1", Name: "Movie", ContentHash: "h1"}})
		st, _ := s.GetByID(ctx, "s1")
		if st.TMDBID != 603 {
			t.Errorf("TMDBID lost: got %d, want 603", st.TMDBID)
		}
	})

	t.Run("BulkUpsert_PreservesTMDBManual", func(t *testing.T) {
		s := factory(t)
		ctx := context.Background()
		s.BulkUpsert(ctx, []models.Stream{{ID: "s1", Name: "Movie", ContentHash: "h1"}})
		s.SetTMDBManual(ctx, "s1", 999)
		s.BulkUpsert(ctx, []models.Stream{{ID: "s1", Name: "Movie", ContentHash: "h1"}})
		st, _ := s.GetByID(ctx, "s1")
		if st.TMDBID != 999 || !st.TMDBManual {
			t.Errorf("TMDBManual lost: %d/%v", st.TMDBID, st.TMDBManual)
		}
	})

	t.Run("DeleteStaleByAccountID", func(t *testing.T) {
		s := factory(t)
		seedStore(t, s)
		ctx := context.Background()
		deleted, _ := s.DeleteStaleByAccountID(ctx, "acc1", []string{"m3u-1"})
		if len(deleted) != 1 || deleted[0] != "m3u-2" {
			t.Errorf("deleted = %v, want [m3u-2]", deleted)
		}
		if _, err := s.GetByID(ctx, "m3u-1"); err != nil {
			t.Error("m3u-1 should still exist")
		}
		if _, err := s.GetByID(ctx, "m3u-2"); err == nil {
			t.Error("m3u-2 should be deleted")
		}
		if _, err := s.GetByID(ctx, "m3u-3"); err != nil {
			t.Error("m3u-3 (acc2) should still exist")
		}
	})

	t.Run("DeleteByAccountID", func(t *testing.T) {
		s := factory(t)
		seedStore(t, s)
		ctx := context.Background()
		s.DeleteByAccountID(ctx, "acc1")
		items, _ := s.ListByAccountID(ctx, "acc1")
		if len(items) != 0 {
			t.Errorf("acc1 should have 0, got %d", len(items))
		}
		items, _ = s.ListByAccountID(ctx, "acc2")
		if len(items) != 1 {
			t.Errorf("acc2 should be unaffected, got %d", len(items))
		}
	})

	t.Run("DeleteStaleBySatIPSourceID", func(t *testing.T) {
		s := factory(t)
		seedStore(t, s)
		ctx := context.Background()
		deleted, _ := s.DeleteStaleBySatIPSourceID(ctx, "src1", []string{"sat-1"})
		if len(deleted) != 1 || deleted[0] != "sat-2" {
			t.Errorf("deleted = %v, want [sat-2]", deleted)
		}
	})

	t.Run("DeleteBySatIPSourceID", func(t *testing.T) {
		s := factory(t)
		seedStore(t, s)
		ctx := context.Background()
		s.DeleteBySatIPSourceID(ctx, "src1")
		items, _ := s.ListBySatIPSourceID(ctx, "src1")
		if len(items) != 0 {
			t.Errorf("src1 should have 0, got %d", len(items))
		}
	})

	t.Run("DeleteOrphanedM3UStreams", func(t *testing.T) {
		s := factory(t)
		seedStore(t, s)
		ctx := context.Background()
		deleted, _ := s.DeleteOrphanedM3UStreams(ctx, []string{"acc1"})
		deletedSet := make(map[string]bool)
		for _, id := range deleted {
			deletedSet[id] = true
		}
		if !deletedSet["m3u-3"] {
			t.Error("m3u-3 (acc2) should be deleted")
		}
		if deletedSet["m3u-1"] || deletedSet["m3u-2"] {
			t.Error("acc1 streams should NOT be deleted")
		}
		if deletedSet["sat-1"] || deletedSet["sat-2"] || deletedSet["sat-3"] {
			t.Error("SAT>IP streams should NOT be deleted")
		}
	})

	t.Run("DeleteOrphanedSatIPStreams", func(t *testing.T) {
		s := factory(t)
		seedStore(t, s)
		ctx := context.Background()
		deleted, _ := s.DeleteOrphanedSatIPStreams(ctx, []string{"src1"})
		if len(deleted) != 1 || deleted[0] != "sat-3" {
			t.Errorf("deleted = %v, want [sat-3]", deleted)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		s := factory(t)
		seedStore(t, s)
		ctx := context.Background()
		s.Delete(ctx, "m3u-1")
		if _, err := s.GetByID(ctx, "m3u-1"); err == nil {
			t.Error("m3u-1 should be deleted")
		}
		all, _ := s.List(ctx)
		if len(all) != 7 {
			t.Errorf("got %d after delete, want 7", len(all))
		}
	})

	t.Run("Clear", func(t *testing.T) {
		s := factory(t)
		seedStore(t, s)
		s.Clear()
		items, _ := s.List(context.Background())
		if len(items) != 0 {
			t.Errorf("got %d after clear, want 0", len(items))
		}
	})

	t.Run("UpdateTMDBID", func(t *testing.T) {
		s := factory(t)
		seedStore(t, s)
		ctx := context.Background()
		s.UpdateTMDBID(ctx, "vod-1", 603)
		st, _ := s.GetByID(ctx, "vod-1")
		if st.TMDBID != 603 {
			t.Errorf("TMDBID = %d, want 603", st.TMDBID)
		}
		if err := s.UpdateTMDBID(ctx, "nonexistent", 123); err == nil {
			t.Error("expected error for nonexistent")
		}
	})

	t.Run("SetTMDBManual", func(t *testing.T) {
		s := factory(t)
		seedStore(t, s)
		ctx := context.Background()
		s.SetTMDBManual(ctx, "vod-1", 999)
		st, _ := s.GetByID(ctx, "vod-1")
		if st.TMDBID != 999 || !st.TMDBManual {
			t.Errorf("got %d/%v, want 999/true", st.TMDBID, st.TMDBManual)
		}
	})

	t.Run("ClearAutoTMDBByAccountID", func(t *testing.T) {
		s := factory(t)
		seedStore(t, s)
		ctx := context.Background()
		s.UpdateTMDBID(ctx, "m3u-1", 100)
		s.SetTMDBManual(ctx, "m3u-2", 200)
		s.UpdateTMDBID(ctx, "m3u-3", 300)
		s.ClearAutoTMDBByAccountID(ctx, "acc1")
		s1, _ := s.GetByID(ctx, "m3u-1")
		if s1.TMDBID != 0 {
			t.Errorf("auto should be cleared: got %d", s1.TMDBID)
		}
		s2, _ := s.GetByID(ctx, "m3u-2")
		if s2.TMDBID != 200 {
			t.Errorf("manual should be preserved: got %d", s2.TMDBID)
		}
		s3, _ := s.GetByID(ctx, "m3u-3")
		if s3.TMDBID != 300 {
			t.Errorf("other account untouched: got %d", s3.TMDBID)
		}
	})

	t.Run("UpdateWireGuardByAccountID", func(t *testing.T) {
		s := factory(t)
		seedStore(t, s)
		ctx := context.Background()
		s.UpdateWireGuardByAccountID(ctx, "acc1", true)
		s1, _ := s.GetByID(ctx, "m3u-1")
		if !s1.UseWireGuard {
			t.Error("m3u-1 should have UseWireGuard=true")
		}
		s3, _ := s.GetByID(ctx, "m3u-3")
		if s3.UseWireGuard {
			t.Error("m3u-3 (acc2) should NOT have UseWireGuard")
		}
	})

	t.Run("SaveAndLoad", func(t *testing.T) {
		dir := t.TempDir()
		log := zerolog.Nop()

		var s1 StreamStore
		if _, ok := factory(t).(*StreamStoreImpl); ok {
			path := dir + "/streams.gob"
			store := NewStreamStore(path, log)
			s1 = store
		} else {
			store := NewIndexedStreamStore(dir, "", log)
			s1 = store
		}

		ctx := context.Background()
		s1.BulkUpsert(ctx, []models.Stream{
			{ID: "s1", Name: "Saved", M3UAccountID: "acc1", ContentHash: "h1"},
			{ID: "s2", Name: "Another", SatIPSourceID: "src1", ContentHash: "h2"},
		})
		s1.UpdateTMDBID(ctx, "s1", 603)
		s1.Save()

		var s2 StreamStore
		if _, ok := factory(t).(*StreamStoreImpl); ok {
			path := dir + "/streams.gob"
			store := NewStreamStore(path, log)
			store.Load()
			s2 = store
		} else {
			store := NewIndexedStreamStore(dir, "", log)
			store.Load()
			s2 = store
		}

		items, _ := s2.List(ctx)
		if len(items) != 2 {
			t.Fatalf("loaded %d, want 2", len(items))
		}
		st, _ := s2.GetByID(ctx, "s1")
		if st.Name != "Saved" || st.TMDBID != 603 {
			t.Errorf("loaded: name=%q tmdb=%d", st.Name, st.TMDBID)
		}
	})

	t.Run("ETagBumpsOnWrite", func(t *testing.T) {
		s := factory(t)
		ctx := context.Background()
		e1 := s.(interface{ ETag() string }).ETag()
		s.BulkUpsert(ctx, []models.Stream{{ID: "s1", Name: "Test", ContentHash: "h1"}})
		e2 := s.(interface{ ETag() string }).ETag()
		if e2 == e1 {
			t.Error("ETag should change after BulkUpsert")
		}
		s.Delete(ctx, "s1")
		e3 := s.(interface{ ETag() string }).ETag()
		if e3 == e2 {
			t.Error("ETag should change after Delete")
		}
	})

	t.Run("ListEmpty", func(t *testing.T) {
		s := factory(t)
		items, _ := s.List(context.Background())
		if len(items) != 0 {
			t.Errorf("empty store: got %d", len(items))
		}
		summaries, _ := s.ListSummaries(context.Background())
		if len(summaries) != 0 {
			t.Errorf("empty summaries: got %d", len(summaries))
		}
	})
}

func TestMapStreamStore(t *testing.T) {
	runStreamStoreTests(t, newMapStore)
}

func TestIndexedStreamStore(t *testing.T) {
	runStreamStoreTests(t, newIndexedStore)
}
