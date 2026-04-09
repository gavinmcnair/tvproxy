package store

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/gavinmcnair/tvproxy/pkg/models"
)

func newTestStreamStore(t *testing.T) *StreamStoreImpl {
	return NewStreamStore(t.TempDir()+"/streams.gob", zerolog.Nop())
}

func seedStreams(t *testing.T, s *StreamStoreImpl) {
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

func TestList(t *testing.T) {
	s := newTestStreamStore(t)
	seedStreams(t, s)

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
}

func TestListSummaries(t *testing.T) {
	s := newTestStreamStore(t)
	seedStreams(t, s)

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
}

func TestGetByID(t *testing.T) {
	s := newTestStreamStore(t)
	seedStreams(t, s)
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
}

func TestListByAccountID(t *testing.T) {
	s := newTestStreamStore(t)
	seedStreams(t, s)

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
}

func TestListBySatIPSourceID(t *testing.T) {
	s := newTestStreamStore(t)
	seedStreams(t, s)

	items, err := s.ListBySatIPSourceID(context.Background(), "src1")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Errorf("got %d items for src1, want 2", len(items))
	}
	for _, item := range items {
		if item.SatIPSourceID != "src1" {
			t.Errorf("got source %q, want src1", item.SatIPSourceID)
		}
	}
}

func TestBulkUpsert_NewItems(t *testing.T) {
	s := newTestStreamStore(t)
	ctx := context.Background()

	err := s.BulkUpsert(ctx, []models.Stream{
		{ID: "s1", Name: "Stream 1", ContentHash: "h1"},
		{ID: "s2", Name: "Stream 2", ContentHash: "h2"},
	})
	if err != nil {
		t.Fatal(err)
	}

	items, _ := s.List(ctx)
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	if items[0].CreatedAt.IsZero() {
		t.Error("CreatedAt should be set for new items")
	}
	if items[0].UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set for new items")
	}
}

func TestBulkUpsert_UpdateExisting(t *testing.T) {
	s := newTestStreamStore(t)
	ctx := context.Background()

	s.BulkUpsert(ctx, []models.Stream{
		{ID: "s1", Name: "Original", ContentHash: "h1"},
	})
	st1, _ := s.GetByID(ctx, "s1")
	origCreated := st1.CreatedAt

	time.Sleep(time.Millisecond)

	s.BulkUpsert(ctx, []models.Stream{
		{ID: "s1", Name: "Updated", ContentHash: "h1"},
	})
	st2, _ := s.GetByID(ctx, "s1")
	if st2.Name != "Updated" {
		t.Errorf("name not updated: got %q", st2.Name)
	}
	if !st2.CreatedAt.Equal(origCreated) {
		t.Error("CreatedAt should be preserved on update")
	}
	if !st2.UpdatedAt.After(origCreated) {
		t.Error("UpdatedAt should be newer than CreatedAt on update")
	}
}

func TestBulkUpsert_PreservesTMDBID(t *testing.T) {
	s := newTestStreamStore(t)
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

func TestBulkUpsert_PreservesTMDBManual(t *testing.T) {
	s := newTestStreamStore(t)
	ctx := context.Background()

	s.BulkUpsert(ctx, []models.Stream{{ID: "s1", Name: "Movie", ContentHash: "h1"}})
	s.SetTMDBManual(ctx, "s1", 999)

	s.BulkUpsert(ctx, []models.Stream{{ID: "s1", Name: "Movie", ContentHash: "h1"}})

	st, _ := s.GetByID(ctx, "s1")
	if st.TMDBID != 999 || !st.TMDBManual {
		t.Errorf("TMDBManual lost: TMDBID=%d Manual=%v", st.TMDBID, st.TMDBManual)
	}
}

func TestDeleteStaleByAccountID(t *testing.T) {
	s := newTestStreamStore(t)
	seedStreams(t, s)
	ctx := context.Background()

	deleted, err := s.DeleteStaleByAccountID(ctx, "acc1", []string{"m3u-1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(deleted) != 1 || deleted[0] != "m3u-2" {
		t.Errorf("deleted = %v, want [m3u-2]", deleted)
	}

	_, err = s.GetByID(ctx, "m3u-1")
	if err != nil {
		t.Error("m3u-1 should still exist (in keepIDs)")
	}
	_, err = s.GetByID(ctx, "m3u-2")
	if err == nil {
		t.Error("m3u-2 should be deleted (not in keepIDs)")
	}
	_, err = s.GetByID(ctx, "m3u-3")
	if err != nil {
		t.Error("m3u-3 should still exist (different account)")
	}
}

func TestDeleteByAccountID(t *testing.T) {
	s := newTestStreamStore(t)
	seedStreams(t, s)
	ctx := context.Background()

	s.DeleteByAccountID(ctx, "acc1")

	items, _ := s.ListByAccountID(ctx, "acc1")
	if len(items) != 0 {
		t.Errorf("acc1 should have 0 streams after delete, got %d", len(items))
	}
	items, _ = s.ListByAccountID(ctx, "acc2")
	if len(items) != 1 {
		t.Errorf("acc2 should be unaffected, got %d", len(items))
	}
}

func TestDeleteStaleBySatIPSourceID(t *testing.T) {
	s := newTestStreamStore(t)
	seedStreams(t, s)
	ctx := context.Background()

	deleted, err := s.DeleteStaleBySatIPSourceID(ctx, "src1", []string{"sat-1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(deleted) != 1 || deleted[0] != "sat-2" {
		t.Errorf("deleted = %v, want [sat-2]", deleted)
	}
	_, err = s.GetByID(ctx, "sat-1")
	if err != nil {
		t.Error("sat-1 should still exist")
	}
	_, err = s.GetByID(ctx, "sat-3")
	if err != nil {
		t.Error("sat-3 should still exist (different source)")
	}
}

func TestDeleteBySatIPSourceID(t *testing.T) {
	s := newTestStreamStore(t)
	seedStreams(t, s)
	ctx := context.Background()

	s.DeleteBySatIPSourceID(ctx, "src1")

	items, _ := s.ListBySatIPSourceID(ctx, "src1")
	if len(items) != 0 {
		t.Errorf("src1 should have 0 streams, got %d", len(items))
	}
	items, _ = s.ListBySatIPSourceID(ctx, "src2")
	if len(items) != 1 {
		t.Errorf("src2 should be unaffected, got %d", len(items))
	}
}

func TestDeleteOrphanedM3UStreams(t *testing.T) {
	s := newTestStreamStore(t)
	seedStreams(t, s)
	ctx := context.Background()

	deleted, err := s.DeleteOrphanedM3UStreams(ctx, []string{"acc1"})
	if err != nil {
		t.Fatal(err)
	}
	deletedSet := make(map[string]bool)
	for _, id := range deleted {
		deletedSet[id] = true
	}
	if !deletedSet["m3u-3"] {
		t.Error("m3u-3 (acc2) should be deleted as orphaned")
	}
	if !deletedSet["vod-1"] && !deletedSet["vod-2"] {
		t.Error("acc3 streams should be deleted as orphaned")
	}
	if deletedSet["m3u-1"] || deletedSet["m3u-2"] {
		t.Error("acc1 streams should NOT be deleted")
	}
	if deletedSet["sat-1"] || deletedSet["sat-2"] || deletedSet["sat-3"] {
		t.Error("SAT>IP streams should NOT be deleted (no M3UAccountID)")
	}
}

func TestDeleteOrphanedSatIPStreams(t *testing.T) {
	s := newTestStreamStore(t)
	seedStreams(t, s)
	ctx := context.Background()

	deleted, err := s.DeleteOrphanedSatIPStreams(ctx, []string{"src1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(deleted) != 1 || deleted[0] != "sat-3" {
		t.Errorf("deleted = %v, want [sat-3]", deleted)
	}
	_, err = s.GetByID(ctx, "sat-1")
	if err != nil {
		t.Error("sat-1 should still exist (src1 is known)")
	}
	_, err = s.GetByID(ctx, "m3u-1")
	if err != nil {
		t.Error("m3u streams should be unaffected")
	}
}

func TestDelete(t *testing.T) {
	s := newTestStreamStore(t)
	seedStreams(t, s)
	ctx := context.Background()

	s.Delete(ctx, "m3u-1")
	_, err := s.GetByID(ctx, "m3u-1")
	if err == nil {
		t.Error("m3u-1 should be deleted")
	}

	all, _ := s.List(ctx)
	if len(all) != 7 {
		t.Errorf("got %d items after delete, want 7", len(all))
	}
}

func TestClear(t *testing.T) {
	s := newTestStreamStore(t)
	seedStreams(t, s)

	s.Clear()

	items, _ := s.List(context.Background())
	if len(items) != 0 {
		t.Errorf("got %d items after clear, want 0", len(items))
	}
}

func TestUpdateTMDBID(t *testing.T) {
	s := newTestStreamStore(t)
	seedStreams(t, s)
	ctx := context.Background()

	s.UpdateTMDBID(ctx, "vod-1", 603)

	st, _ := s.GetByID(ctx, "vod-1")
	if st.TMDBID != 603 {
		t.Errorf("TMDBID = %d, want 603", st.TMDBID)
	}

	err := s.UpdateTMDBID(ctx, "nonexistent", 123)
	if err == nil {
		t.Error("expected error for nonexistent stream")
	}
}

func TestSetTMDBManual(t *testing.T) {
	s := newTestStreamStore(t)
	seedStreams(t, s)
	ctx := context.Background()

	s.SetTMDBManual(ctx, "vod-1", 999)

	st, _ := s.GetByID(ctx, "vod-1")
	if st.TMDBID != 999 || !st.TMDBManual {
		t.Errorf("SetTMDBManual: TMDBID=%d Manual=%v, want 999/true", st.TMDBID, st.TMDBManual)
	}
}

func TestClearAutoTMDBByAccountID(t *testing.T) {
	s := newTestStreamStore(t)
	seedStreams(t, s)
	ctx := context.Background()

	s.UpdateTMDBID(ctx, "m3u-1", 100)
	s.SetTMDBManual(ctx, "m3u-2", 200)
	s.UpdateTMDBID(ctx, "m3u-3", 300)

	s.ClearAutoTMDBByAccountID(ctx, "acc1")

	s1, _ := s.GetByID(ctx, "m3u-1")
	if s1.TMDBID != 0 {
		t.Errorf("auto match should be cleared: got %d", s1.TMDBID)
	}
	s2, _ := s.GetByID(ctx, "m3u-2")
	if s2.TMDBID != 200 {
		t.Errorf("manual match should be preserved: got %d", s2.TMDBID)
	}
	s3, _ := s.GetByID(ctx, "m3u-3")
	if s3.TMDBID != 300 {
		t.Errorf("other account should be untouched: got %d", s3.TMDBID)
	}
}

func TestUpdateWireGuardByAccountID(t *testing.T) {
	s := newTestStreamStore(t)
	seedStreams(t, s)
	ctx := context.Background()

	s.UpdateWireGuardByAccountID(ctx, "acc1", true)

	s1, _ := s.GetByID(ctx, "m3u-1")
	if !s1.UseWireGuard {
		t.Error("m3u-1 should have UseWireGuard=true")
	}
	s2, _ := s.GetByID(ctx, "m3u-2")
	if !s2.UseWireGuard {
		t.Error("m3u-2 should have UseWireGuard=true")
	}
	s3, _ := s.GetByID(ctx, "m3u-3")
	if s3.UseWireGuard {
		t.Error("m3u-3 (acc2) should NOT have UseWireGuard")
	}

	s.UpdateWireGuardByAccountID(ctx, "acc1", false)
	s1, _ = s.GetByID(ctx, "m3u-1")
	if s1.UseWireGuard {
		t.Error("m3u-1 should have UseWireGuard=false after toggle off")
	}
}

func TestSaveAndLoad(t *testing.T) {
	path := t.TempDir() + "/streams.gob"
	log := zerolog.Nop()

	s1 := NewStreamStore(path, log)
	ctx := context.Background()
	s1.BulkUpsert(ctx, []models.Stream{
		{ID: "s1", Name: "Saved Stream", M3UAccountID: "acc1", ContentHash: "h1", IsActive: true},
		{ID: "s2", Name: "Another Stream", SatIPSourceID: "src1", ContentHash: "h2", IsActive: true},
	})
	s1.UpdateTMDBID(ctx, "s1", 603)
	s1.Save()

	s2 := NewStreamStore(path, log)
	s2.Load()

	items, _ := s2.List(ctx)
	if len(items) != 2 {
		t.Fatalf("loaded %d items, want 2", len(items))
	}

	st, err := s2.GetByID(ctx, "s1")
	if err != nil {
		t.Fatal(err)
	}
	if st.Name != "Saved Stream" {
		t.Errorf("loaded name = %q, want Saved Stream", st.Name)
	}
	if st.TMDBID != 603 {
		t.Errorf("loaded TMDBID = %d, want 603", st.TMDBID)
	}
}

func TestETagBumpsOnWrite(t *testing.T) {
	s := newTestStreamStore(t)
	ctx := context.Background()

	etag1 := s.ETag()

	s.BulkUpsert(ctx, []models.Stream{{ID: "s1", Name: "Test", ContentHash: "h1"}})
	etag2 := s.ETag()
	if etag2 == etag1 {
		t.Error("ETag should change after BulkUpsert")
	}

	s.Delete(ctx, "s1")
	etag3 := s.ETag()
	if etag3 == etag2 {
		t.Error("ETag should change after Delete")
	}
}

func TestListEmpty(t *testing.T) {
	s := newTestStreamStore(t)

	items, err := s.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Errorf("empty store should return 0 items, got %d", len(items))
	}

	summaries, err := s.ListSummaries(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 0 {
		t.Errorf("empty store should return 0 summaries, got %d", len(summaries))
	}
}
