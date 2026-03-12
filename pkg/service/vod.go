package service

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/gavinmcnair/tvproxy/pkg/config"
	"github.com/gavinmcnair/tvproxy/pkg/ffmpeg"
	"github.com/gavinmcnair/tvproxy/pkg/repository"
)

type VODSession struct {
	ID           string
	StreamURL    string
	Duration     float64
	FilePath     string
	TempDir      string
	LastAccess   time.Time
	Ready        chan struct{}
	Error        error
	BufferedSecs float64
	mu           sync.Mutex
}

func (s *VODSession) GetBufferedSecs() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.BufferedSecs
}

type VODService struct {
	config            *config.Config
	channelRepo       *repository.ChannelRepository
	streamRepo        *repository.StreamRepository
	streamProfileRepo *repository.StreamProfileRepository
	log               zerolog.Logger
	mu                sync.RWMutex
	sessions          map[string]*VODSession
}

func NewVODService(
	channelRepo *repository.ChannelRepository,
	streamRepo *repository.StreamRepository,
	streamProfileRepo *repository.StreamProfileRepository,
	cfg *config.Config,
	log zerolog.Logger,
) *VODService {
	return &VODService{
		config:            cfg,
		channelRepo:       channelRepo,
		streamRepo:        streamRepo,
		streamProfileRepo: streamProfileRepo,
		log:               log.With().Str("service", "vod").Logger(),
		sessions:          make(map[string]*VODSession),
	}
}

func (s *VODService) ProbeStream(ctx context.Context, streamID int64) (*ffmpeg.ProbeResult, error) {
	stream, err := s.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		return nil, fmt.Errorf("stream not found: %w", err)
	}
	return ffmpeg.Probe(ctx, stream.URL, s.config.UserAgent)
}

func (s *VODService) CreateSession(ctx context.Context, streamID int64, profileName string) (*VODSession, error) {
	stream, err := s.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		return nil, fmt.Errorf("stream not found: %w", err)
	}
	if !stream.IsActive {
		return nil, fmt.Errorf("stream %d is inactive", streamID)
	}
	return s.createSessionForURL(ctx, stream.URL, stream.ID, profileName)
}

func (s *VODService) CreateSessionForChannel(ctx context.Context, channelID int64, profileName string) (*VODSession, error) {
	channel, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return nil, fmt.Errorf("channel not found: %w", err)
	}
	if !channel.IsEnabled {
		return nil, fmt.Errorf("channel %d is disabled", channelID)
	}

	channelStreams, err := s.channelRepo.GetStreams(ctx, channelID)
	if err != nil {
		return nil, fmt.Errorf("getting channel streams: %w", err)
	}

	for _, cs := range channelStreams {
		stream, err := s.streamRepo.GetByID(ctx, cs.StreamID)
		if err != nil || !stream.IsActive {
			continue
		}
		return s.createSessionForURL(ctx, stream.URL, stream.ID, profileName)
	}

	return nil, fmt.Errorf("no active streams for channel %d", channelID)
}

func (s *VODService) createSessionForURL(ctx context.Context, streamURL string, streamID int64, profileName string) (*VODSession, error) {
	var duration float64
	probe, err := ffmpeg.Probe(ctx, streamURL, s.config.UserAgent)
	if err == nil && probe.IsVOD {
		duration = probe.Duration
	}

	profileArgs := "-hide_banner -loglevel warning -i {input} -c copy -f mp4 -movflags frag_keyframe+empty_moov+default_base_moof pipe:1"
	command := "ffmpeg"
	if profileName != "" {
		sp, err := s.streamProfileRepo.GetByName(ctx, profileName)
		if err != nil {
			return nil, fmt.Errorf("profile %q not found: %w", profileName, err)
		}
		if sp.Args != "" {
			profileArgs = sp.Args
			command = sp.Command
		}
	}

	id := uuid.New().String()
	tempDir := filepath.Join(s.config.VODTempDir, id)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}

	filePath := filepath.Join(tempDir, "video.mp4")

	session := &VODSession{
		ID:         id,
		StreamURL:  streamURL,
		Duration:   duration,
		FilePath:   filePath,
		TempDir:    tempDir,
		LastAccess: time.Now(),
		Ready:      make(chan struct{}),
	}

	s.mu.Lock()
	s.sessions[id] = session
	s.mu.Unlock()

	go s.remux(session, command, profileArgs)

	probeDur := 0.0
	if probe != nil {
		probeDur = probe.Duration
	}
	s.log.Info().Str("session_id", id).Int64("stream_id", streamID).Float64("duration", probeDur).Msg("VOD session created, remux started")
	return session, nil
}

func (s *VODService) remux(session *VODSession, command, profileArgs string) {
	argsStr := strings.Replace(profileArgs, "{input}", session.StreamURL, 1)
	args := ShellSplit(argsStr)
	args = InjectUserAgent(args, s.config.UserAgent)

	for i, arg := range args {
		if arg == "pipe:1" {
			args[i] = session.FilePath
		}
	}

	args = append(args, "-progress", "pipe:2")

	s.log.Info().Str("session_id", session.ID).Strs("args", args).Msg("starting VOD remux")

	cmd := exec.Command(command, args...)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		session.Error = fmt.Errorf("creating stderr pipe: %w", err)
		close(session.Ready)
		return
	}

	if err := cmd.Start(); err != nil {
		session.Error = fmt.Errorf("starting ffmpeg: %w", err)
		close(session.Ready)
		return
	}

	go s.parseProgress(session, stderr)

	if err := cmd.Wait(); err != nil {
		session.Error = fmt.Errorf("ffmpeg remux failed: %w", err)
		os.Remove(session.FilePath)
		close(session.Ready)
		return
	}

	if session.Duration > 0 {
		session.mu.Lock()
		session.BufferedSecs = session.Duration
		session.mu.Unlock()
	}

	s.log.Info().Str("session_id", session.ID).Msg("remux complete")
	close(session.Ready)
}

func (s *VODService) parseProgress(session *VODSession, r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "out_time_us=") {
			usStr := strings.TrimPrefix(line, "out_time_us=")
			us, err := strconv.ParseInt(usStr, 10, 64)
			if err == nil && us > 0 {
				secs := float64(us) / 1_000_000.0
				session.mu.Lock()
				session.BufferedSecs = secs
				session.mu.Unlock()
			}
		} else if !strings.HasPrefix(line, "progress=") &&
			!strings.HasPrefix(line, "out_time_ms=") &&
			!strings.HasPrefix(line, "out_time=") &&
			!strings.HasPrefix(line, "frame=") &&
			!strings.HasPrefix(line, "fps=") &&
			!strings.HasPrefix(line, "stream_") &&
			!strings.HasPrefix(line, "bitrate=") &&
			!strings.HasPrefix(line, "total_size=") &&
			!strings.HasPrefix(line, "speed=") &&
			!strings.HasPrefix(line, "dup_frames=") &&
			!strings.HasPrefix(line, "drop_frames=") &&
			line != "" {
			s.log.Warn().Str("session_id", session.ID).Str("ffmpeg", line).Msg("vod ffmpeg output")
		}
	}
}

func (s *VODService) StreamSeek(ctx context.Context, session *VODSession, offsetSecs float64) (io.ReadCloser, error) {
	session.mu.Lock()
	buffered := session.BufferedSecs
	session.mu.Unlock()

	if offsetSecs > buffered {
		return nil, fmt.Errorf("offset %.1fs exceeds buffered %.1fs", offsetSecs, buffered)
	}

	args := []string{
		"-hide_banner", "-loglevel", "warning",
		"-ss", fmt.Sprintf("%.6f", offsetSecs),
		"-i", session.FilePath,
		"-c", "copy",
		"-f", "mp4",
		"-movflags", "frag_keyframe+empty_moov+default_base_moof",
		"-fflags", "+genpts",
		"pipe:1",
	}

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdout pipe: %w", err)
	}
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting seek ffmpeg: %w", err)
	}

	s.log.Info().Str("session_id", session.ID).Float64("offset", offsetSecs).Msg("serving seek stream")

	return &seekReadCloser{ReadCloser: stdout, cmd: cmd}, nil
}

type seekReadCloser struct {
	io.ReadCloser
	cmd *exec.Cmd
}

func (s *seekReadCloser) Close() error {
	s.ReadCloser.Close()
	return s.cmd.Wait()
}

func (s *VODService) GetSession(id string) (*VODSession, bool) {
	s.mu.RLock()
	session, ok := s.sessions[id]
	s.mu.RUnlock()
	if ok {
		session.mu.Lock()
		session.LastAccess = time.Now()
		session.mu.Unlock()
	}
	return session, ok
}

func (s *VODService) DeleteSession(id string) {
	s.mu.Lock()
	session, ok := s.sessions[id]
	if ok {
		delete(s.sessions, id)
	}
	s.mu.Unlock()

	if ok {
		os.RemoveAll(session.TempDir)
		s.log.Info().Str("session_id", id).Msg("VOD session deleted")
	}
}

func (s *VODService) CleanupExpired() {
	s.mu.Lock()
	var expired []string
	for id, session := range s.sessions {
		session.mu.Lock()
		if time.Since(session.LastAccess) > s.config.VODSessionTimeout {
			expired = append(expired, id)
		}
		session.mu.Unlock()
	}
	for _, id := range expired {
		session := s.sessions[id]
		delete(s.sessions, id)
		os.RemoveAll(session.TempDir)
		s.log.Info().Str("session_id", id).Msg("expired VOD session cleaned up")
	}
	s.mu.Unlock()
}
