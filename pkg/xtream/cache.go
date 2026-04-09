package xtream

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type MovieMeta struct {
	StreamID     int      `json:"stream_id"`
	Name         string   `json:"name"`
	Plot         string   `json:"plot,omitempty"`
	Cast         string   `json:"cast,omitempty"`
	Director     string   `json:"director,omitempty"`
	Genre        string   `json:"genre,omitempty"`
	ReleaseDate  string   `json:"release_date,omitempty"`
	Rating       string   `json:"rating,omitempty"`
	PosterURL    string   `json:"poster_url,omitempty"`
	BackdropURL  string   `json:"backdrop_url,omitempty"`
	Duration     int      `json:"duration_secs,omitempty"`
	VideoCodec   string   `json:"video_codec,omitempty"`
	AudioCodec   string   `json:"audio_codec,omitempty"`
	Width        int      `json:"width,omitempty"`
	Height       int      `json:"height,omitempty"`
	Container    string   `json:"container,omitempty"`
	IsAdult      bool     `json:"is_adult,omitempty"`
	CategoryName string   `json:"category_name,omitempty"`
	Trailer      string   `json:"trailer,omitempty"`
}

type SeriesMeta struct {
	SeriesID    int             `json:"series_id"`
	Name        string          `json:"name"`
	Plot        string          `json:"plot,omitempty"`
	Cast        string          `json:"cast,omitempty"`
	Director    string          `json:"director,omitempty"`
	Genre       string          `json:"genre,omitempty"`
	ReleaseDate string          `json:"release_date,omitempty"`
	Rating      string          `json:"rating,omitempty"`
	PosterURL   string          `json:"poster_url,omitempty"`
	BackdropURL string          `json:"backdrop_url,omitempty"`
	Trailer     string          `json:"trailer,omitempty"`
	CategoryName string         `json:"category_name,omitempty"`
	Seasons     []SeasonMeta    `json:"seasons,omitempty"`
}

type SeasonMeta struct {
	SeasonNumber int    `json:"season_number"`
	Name         string `json:"name,omitempty"`
	AirDate      string `json:"air_date,omitempty"`
	EpisodeCount int    `json:"episode_count,omitempty"`
	CoverURL     string `json:"cover_url,omitempty"`
}

type EpisodeMeta struct {
	ID           int    `json:"id"`
	EpisodeNum   int    `json:"episode_num"`
	Season       int    `json:"season"`
	Title        string `json:"title"`
	Plot         string `json:"plot,omitempty"`
	Duration     int    `json:"duration_secs,omitempty"`
	DurationStr  string `json:"duration,omitempty"`
	VideoCodec   string `json:"video_codec,omitempty"`
	AudioCodec   string `json:"audio_codec,omitempty"`
	Width        int    `json:"width,omitempty"`
	Height       int    `json:"height,omitempty"`
	Container    string `json:"container,omitempty"`
	CoverURL     string `json:"cover_url,omitempty"`
}

type Cache struct {
	mu            sync.RWMutex
	movies        map[int]*MovieMeta
	series        map[int]*SeriesMeta
	seriesNameIdx map[string]*SeriesMeta
	episodes      map[int]*EpisodeMeta
	path          string
}

type cacheData struct {
	Movies   map[int]*MovieMeta   `json:"movies,omitempty"`
	Series   map[int]*SeriesMeta  `json:"series,omitempty"`
	Episodes map[int]*EpisodeMeta `json:"episodes,omitempty"`
}

func NewCache(baseDir string) *Cache {
	c := &Cache{
		movies:   make(map[int]*MovieMeta),
		series:   make(map[int]*SeriesMeta),
		episodes: make(map[int]*EpisodeMeta),
		path:     filepath.Join(baseDir, "xtream_cache.json"),
	}
	c.load()
	return c
}

func (c *Cache) GetMovie(streamID int) *MovieMeta {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.movies[streamID]
}

func (c *Cache) SetMovie(streamID int, m *MovieMeta) {
	c.mu.Lock()
	c.movies[streamID] = m
	c.mu.Unlock()
}

func (c *Cache) GetSeries(seriesID int) *SeriesMeta {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.series[seriesID]
}

func (c *Cache) FindSeriesByName(name string) *SeriesMeta {
	c.mu.Lock()
	if c.seriesNameIdx == nil {
		c.seriesNameIdx = make(map[string]*SeriesMeta, len(c.series)*2)
		for _, s := range c.series {
			c.seriesNameIdx[s.Name] = s
			if idx := strings.Index(s.Name, ":"); idx >= 2 && idx <= 6 {
				cleaned := strings.TrimSpace(s.Name[idx+1:])
				c.seriesNameIdx[cleaned] = s
			}
		}
	}
	result := c.seriesNameIdx[name]
	c.mu.Unlock()
	return result
}

func (c *Cache) SetSeries(seriesID int, s *SeriesMeta) {
	c.mu.Lock()
	c.series[seriesID] = s
	c.seriesNameIdx = nil
	c.mu.Unlock()
}

func (c *Cache) GetEpisode(episodeID int) *EpisodeMeta {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.episodes[episodeID]
}

func (c *Cache) SetEpisode(episodeID int, e *EpisodeMeta) {
	c.mu.Lock()
	c.episodes[episodeID] = e
	c.mu.Unlock()
}

func (c *Cache) Save() {
	c.mu.RLock()
	data, err := json.MarshalIndent(cacheData{
		Movies:   c.movies,
		Series:   c.series,
		Episodes: c.episodes,
	}, "", "  ")
	c.mu.RUnlock()
	if err != nil {
		return
	}
	os.MkdirAll(filepath.Dir(c.path), 0755)
	tmp := c.path + ".tmp"
	os.WriteFile(tmp, data, 0644)
	os.Rename(tmp, c.path)
}

func (c *Cache) load() {
	data, err := os.ReadFile(c.path)
	if err != nil {
		return
	}
	var d cacheData
	if err := json.Unmarshal(data, &d); err != nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if d.Movies != nil {
		c.movies = d.Movies
	}
	if d.Series != nil {
		c.series = d.Series
	}
	if d.Episodes != nil {
		c.episodes = d.Episodes
	}
}

func (c *Cache) PosterURLs() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var urls []string
	for _, m := range c.movies {
		if m.PosterURL != "" {
			urls = append(urls, m.PosterURL)
		}
		if m.BackdropURL != "" {
			urls = append(urls, m.BackdropURL)
		}
	}
	for _, s := range c.series {
		if s.PosterURL != "" {
			urls = append(urls, s.PosterURL)
		}
		if s.BackdropURL != "" {
			urls = append(urls, s.BackdropURL)
		}
	}
	return urls
}

func (c *Cache) Stats() (int, int, int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.movies), len(c.series), len(c.episodes)
}
