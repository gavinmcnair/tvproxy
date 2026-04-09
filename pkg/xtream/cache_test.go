package xtream

import (
	"testing"
)

func TestCacheMovieCRUD(t *testing.T) {
	c := NewCache(t.TempDir())

	if m := c.GetMovie(603); m != nil {
		t.Fatal("expected nil")
	}

	c.SetMovie(603, &MovieMeta{StreamID: 603, Name: "The Matrix", PosterURL: "/poster.jpg", Rating: "8.7"})

	m := c.GetMovie(603)
	if m == nil || m.Name != "The Matrix" {
		t.Fatalf("expected The Matrix, got %v", m)
	}
	if m.Rating != "8.7" {
		t.Errorf("rating = %q, want 8.7", m.Rating)
	}
}

func TestCacheSeriesCRUD(t *testing.T) {
	c := NewCache(t.TempDir())

	c.SetSeries(1396, &SeriesMeta{
		SeriesID: 1396,
		Name:     "Breaking Bad",
		Genre:    "Drama",
		Plot:     "Walter White",
		Seasons:  []SeasonMeta{{SeasonNumber: 1, EpisodeCount: 7}},
	})

	s := c.GetSeries(1396)
	if s == nil || s.Name != "Breaking Bad" {
		t.Fatalf("expected Breaking Bad, got %v", s)
	}
	if len(s.Seasons) != 1 || s.Seasons[0].EpisodeCount != 7 {
		t.Errorf("seasons: %v", s.Seasons)
	}
}

func TestCacheEpisodeCRUD(t *testing.T) {
	c := NewCache(t.TempDir())

	c.SetEpisode(12345, &EpisodeMeta{
		ID: 12345, EpisodeNum: 1, Season: 1, Title: "Pilot",
		Duration: 3600, VideoCodec: "h264", Width: 1920, Height: 1080,
	})

	e := c.GetEpisode(12345)
	if e == nil || e.Title != "Pilot" {
		t.Fatalf("expected Pilot, got %v", e)
	}
	if e.VideoCodec != "h264" || e.Width != 1920 {
		t.Errorf("codec=%s width=%d", e.VideoCodec, e.Width)
	}
}

func TestCachePersistence(t *testing.T) {
	dir := t.TempDir()
	c1 := NewCache(dir)
	c1.SetMovie(1, &MovieMeta{Name: "Movie 1"})
	c1.SetSeries(2, &SeriesMeta{Name: "Series 1"})
	c1.SetEpisode(3, &EpisodeMeta{Title: "Episode 1"})
	c1.Save()

	c2 := NewCache(dir)
	if m := c2.GetMovie(1); m == nil || m.Name != "Movie 1" {
		t.Errorf("movie not persisted: %v", m)
	}
	if s := c2.GetSeries(2); s == nil || s.Name != "Series 1" {
		t.Errorf("series not persisted: %v", s)
	}
	if e := c2.GetEpisode(3); e == nil || e.Title != "Episode 1" {
		t.Errorf("episode not persisted: %v", e)
	}
}

func TestCacheStats(t *testing.T) {
	c := NewCache(t.TempDir())
	c.SetMovie(1, &MovieMeta{Name: "M1"})
	c.SetMovie(2, &MovieMeta{Name: "M2"})
	c.SetSeries(1, &SeriesMeta{Name: "S1"})
	c.SetEpisode(1, &EpisodeMeta{Title: "E1"})
	c.SetEpisode(2, &EpisodeMeta{Title: "E2"})
	c.SetEpisode(3, &EpisodeMeta{Title: "E3"})

	m, s, e := c.Stats()
	if m != 2 || s != 1 || e != 3 {
		t.Errorf("stats: movies=%d series=%d episodes=%d, want 2/1/3", m, s, e)
	}
}
