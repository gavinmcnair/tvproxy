package service

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/gavinmcnair/tvproxy/pkg/config"
)

type ManagedProcess struct {
	ID           string
	InputURL     string
	OutputPath   string
	TempDir      string
	BufferedSecs float64
	Error        error
	Ready        chan struct{}
	cancel       context.CancelFunc
	mu           sync.Mutex
}

type FFmpegManager struct {
	config    *config.Config
	log       zerolog.Logger
	mu        sync.RWMutex
	processes map[string]*ManagedProcess
}

func NewFFmpegManager(cfg *config.Config, log zerolog.Logger) *FFmpegManager {
	return &FFmpegManager{
		config:    cfg,
		log:       log.With().Str("component", "ffmpeg_manager").Logger(),
		processes: make(map[string]*ManagedProcess),
	}
}

func (m *FFmpegManager) Start(inputURL, outputPath, tempDir, command string, args []string) string {
	id := uuid.New().String()
	ctx, cancel := context.WithCancel(context.Background())

	proc := &ManagedProcess{
		ID:         id,
		InputURL:   inputURL,
		OutputPath: outputPath,
		TempDir:    tempDir,
		Ready:      make(chan struct{}),
		cancel:     cancel,
	}

	m.mu.Lock()
	m.processes[id] = proc
	m.mu.Unlock()

	if args == nil {
		args = ShellSplit("-hide_banner -loglevel warning -i {input} -c copy -f mp4 -movflags frag_keyframe+empty_moov+default_base_moof {output}")
	}

	for i, arg := range args {
		switch arg {
		case "{input}":
			args[i] = inputURL
		case "{output}", "pipe:1":
			args[i] = outputPath
		}
	}

	args = append([]string{"-y"}, args...)
	args = InjectUserAgent(args, m.config.UserAgent)
	delayMax, rwTimeout := 30, 30000000
	if m.config.Settings != nil {
		delayMax = m.config.Settings.Network.ReconnectDelayMax
		rwTimeout = m.config.Settings.Network.ReconnectRWTimeout
	}
	args = InjectReconnect(args, inputURL, delayMax, rwTimeout)
	args = append(args, "-progress", "pipe:2")

	go m.run(ctx, proc, command, args)

	return id
}

func (m *FFmpegManager) Stop(id string) {
	m.mu.RLock()
	proc, ok := m.processes[id]
	m.mu.RUnlock()
	if !ok {
		return
	}
	proc.cancel()
}

func (m *FFmpegManager) Wait(id string) {
	m.mu.RLock()
	proc, ok := m.processes[id]
	m.mu.RUnlock()
	if !ok {
		return
	}
	<-proc.Ready
}

func (m *FFmpegManager) GetBufferedSecs(id string) float64 {
	m.mu.RLock()
	proc, ok := m.processes[id]
	m.mu.RUnlock()
	if !ok {
		return 0
	}
	proc.mu.Lock()
	defer proc.mu.Unlock()
	return proc.BufferedSecs
}

func (m *FFmpegManager) IsReady(id string) bool {
	m.mu.RLock()
	proc, ok := m.processes[id]
	m.mu.RUnlock()
	if !ok {
		return true
	}
	select {
	case <-proc.Ready:
		return true
	default:
		return false
	}
}

func (m *FFmpegManager) GetError(id string) error {
	m.mu.RLock()
	proc, ok := m.processes[id]
	m.mu.RUnlock()
	if !ok {
		return nil
	}
	proc.mu.Lock()
	defer proc.mu.Unlock()
	return proc.Error
}

func (m *FFmpegManager) Remove(id string) {
	m.mu.Lock()
	delete(m.processes, id)
	m.mu.Unlock()
}

func (m *FFmpegManager) run(ctx context.Context, proc *ManagedProcess, command string, args []string) {
	m.log.Info().Str("process_id", proc.ID).Strs("args", args).Msg("starting ffmpeg")

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Cancel = func() error {
		return cmd.Process.Signal(syscall.SIGTERM)
	}
	waitDelay := 5 * time.Second
	if m.config.Settings != nil {
		waitDelay = m.config.Settings.FFmpeg.WaitDelay
	}
	cmd.WaitDelay = waitDelay

	stderr, err := cmd.StderrPipe()
	if err != nil {
		proc.mu.Lock()
		proc.Error = fmt.Errorf("creating stderr pipe: %w", err)
		proc.mu.Unlock()
		close(proc.Ready)
		return
	}

	if err := cmd.Start(); err != nil {
		proc.mu.Lock()
		proc.Error = fmt.Errorf("starting ffmpeg: %w", err)
		proc.mu.Unlock()
		close(proc.Ready)
		return
	}

	go m.parseProgress(proc, stderr)

	startupDur := 30 * time.Second
	if m.config.Settings != nil {
		startupDur = m.config.Settings.FFmpeg.StartupTimeout
	}
	startupTimeout := time.AfterFunc(startupDur, func() {
		proc.mu.Lock()
		buffered := proc.BufferedSecs
		proc.mu.Unlock()
		if buffered == 0 {
			m.log.Warn().Str("process_id", proc.ID).Dur("timeout", startupDur).Msg("ffmpeg startup timeout, no data received")
			proc.cancel()
		}
	})

	waitErr := cmd.Wait()
	startupTimeout.Stop()

	if waitErr != nil && ctx.Err() == nil {
		proc.mu.Lock()
		proc.Error = fmt.Errorf("ffmpeg failed: %w", waitErr)
		proc.mu.Unlock()
	}

	close(proc.Ready)
}

var nonAlphanumRe = regexp.MustCompile(`[^a-zA-Z0-9]+`)

var ffmpegNoisePatterns = []string{
	"non-existing PPS",
	"non-existing SPS",
	"no frame!",
	"skipping",
	"missing picture",
	"concealing",
	"decode_slice_header",
	"error while decoding",
	"missing reference picture",
	"reference picture reordering",
	"Last message repeated",
}

func isFFmpegNoise(line string) bool {
	for _, pattern := range ffmpegNoisePatterns {
		if strings.Contains(line, pattern) {
			return true
		}
	}
	return false
}

func (m *FFmpegManager) parseProgress(proc *ManagedProcess, r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "out_time_us=") {
			usStr := strings.TrimPrefix(line, "out_time_us=")
			us, err := strconv.ParseInt(usStr, 10, 64)
			if err == nil && us > 0 {
				secs := float64(us) / 1_000_000.0
				if secs > 172800 {
					m.log.Warn().Str("process_id", proc.ID).Float64("secs", secs).Msg("ffmpeg progress exceeds 48h cap")
					secs = 172800
				}
				proc.mu.Lock()
				proc.BufferedSecs = secs
				proc.mu.Unlock()
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
			!isFFmpegNoise(line) &&
			line != "" {
			m.log.Warn().Str("process_id", proc.ID).Str("ffmpeg", line).Msg("ffmpeg output")
		}
	}
}

func (m *FFmpegManager) ExtractSegment(inputPath, outputPath string, startSecs, endSecs float64) error {
	duration := endSecs - startSecs
	if duration <= 0 {
		return fmt.Errorf("invalid segment duration: %.1f", duration)
	}

	args := []string{
		"-hide_banner", "-loglevel", "warning",
		"-ss", fmt.Sprintf("%.6f", startSecs),
		"-i", inputPath,
		"-t", fmt.Sprintf("%.6f", duration),
		"-c", "copy",
		"-f", "mp4",
		"-movflags", "frag_keyframe+empty_moov+default_base_moof",
		outputPath,
	}

	m.log.Info().Str("input", inputPath).Str("output", outputPath).Float64("start", startSecs).Float64("end", endSecs).Msg("extracting segment")

	cmd := exec.Command("ffmpeg", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg extraction failed: %w: %s", err, string(output))
	}

	m.log.Info().Str("output", outputPath).Msg("segment extraction complete")
	return nil
}

func sanitizeFilename(title string, t time.Time) string {
	name := nonAlphanumRe.ReplaceAllString(title, "_")
	name = strings.Trim(name, "_")
	if len(name) > 60 {
		name = name[:60]
	}
	if name == "" {
		name = "recording"
	}
	return name + "_" + t.Format("20060102_1504")
}
