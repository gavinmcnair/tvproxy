package service

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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
	archive      bool
	archiveName  string
	archiveDir   string
	stopAt       time.Time
	mu           sync.Mutex
}

type ArchiveInfo struct {
	ProcessID   string  `json:"process_id"`
	ArchiveName string  `json:"archive_name"`
	Buffered    float64 `json:"buffered_secs"`
	StopAt      string  `json:"stop_at,omitempty"`
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

	args = InjectUserAgent(args, m.config.UserAgent)
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

	proc.mu.Lock()
	proc.archive = false
	proc.mu.Unlock()

	proc.cancel()
}

func (m *FFmpegManager) StopAndArchive(id string) {
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

func (m *FFmpegManager) MarkForArchival(id, archiveName, archiveDir string, stopAt time.Time) {
	m.mu.RLock()
	proc, ok := m.processes[id]
	m.mu.RUnlock()
	if !ok {
		return
	}

	proc.mu.Lock()
	proc.archive = true
	proc.archiveName = archiveName
	proc.archiveDir = archiveDir
	proc.stopAt = stopAt
	procCancel := proc.cancel
	proc.mu.Unlock()

	if !stopAt.IsZero() {
		deadlineCtx, deadlineCancel := context.WithDeadline(context.Background(), stopAt)
		go func() {
			select {
			case <-deadlineCtx.Done():
				deadlineCancel()
				m.log.Info().Str("process_id", proc.ID).Msg("recording deadline reached")
				procCancel()
			case <-proc.Ready:
				deadlineCancel()
			}
		}()
	}
}

func (m *FFmpegManager) CancelArchival(id string) {
	m.mu.RLock()
	proc, ok := m.processes[id]
	m.mu.RUnlock()
	if !ok {
		return
	}

	proc.mu.Lock()
	proc.archive = false
	proc.mu.Unlock()

	proc.cancel()
}

func (m *FFmpegManager) ListArchiving() []ArchiveInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []ArchiveInfo
	for _, proc := range m.processes {
		proc.mu.Lock()
		if proc.archive {
			info := ArchiveInfo{
				ProcessID:   proc.ID,
				ArchiveName: proc.archiveName,
				Buffered:    proc.BufferedSecs,
			}
			if !proc.stopAt.IsZero() {
				info.StopAt = proc.stopAt.Format(time.RFC3339)
			}
			list = append(list, info)
		}
		proc.mu.Unlock()
	}
	return list
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
	cmd.WaitDelay = 5 * time.Second

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

	waitErr := cmd.Wait()

	proc.mu.Lock()
	shouldArchive := proc.archive
	proc.mu.Unlock()

	if shouldArchive {
		m.log.Info().Str("process_id", proc.ID).Msg("ffmpeg stopped, finalizing archival")
		m.finalizeArchival(proc)
	} else if waitErr != nil && ctx.Err() == nil {
		proc.mu.Lock()
		proc.Error = fmt.Errorf("ffmpeg failed: %w", waitErr)
		proc.mu.Unlock()
	}

	close(proc.Ready)
}

var mgrNonAlphanumRe = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func (m *FFmpegManager) parseProgress(proc *ManagedProcess, r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "out_time_us=") {
			usStr := strings.TrimPrefix(line, "out_time_us=")
			us, err := strconv.ParseInt(usStr, 10, 64)
			if err == nil && us > 0 {
				secs := float64(us) / 1_000_000.0
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
			line != "" {
			m.log.Warn().Str("process_id", proc.ID).Str("ffmpeg", line).Msg("ffmpeg output")
		}
	}
}

func (m *FFmpegManager) finalizeArchival(proc *ManagedProcess) {
	info, err := os.Stat(proc.OutputPath)
	if os.IsNotExist(err) {
		m.log.Warn().Str("process_id", proc.ID).Str("path", proc.OutputPath).Msg("recording file not found, nothing to finalize")
		return
	}
	if err != nil {
		m.log.Error().Err(err).Str("process_id", proc.ID).Str("path", proc.OutputPath).Msg("failed to stat recording file")
		return
	}

	proc.mu.Lock()
	name := proc.archiveName
	destDir := proc.archiveDir
	proc.mu.Unlock()

	if destDir == "" {
		destDir = m.config.RecordDir
	}

	m.log.Info().Str("process_id", proc.ID).Int64("size_bytes", info.Size()).Str("src", proc.OutputPath).Str("dest_dir", destDir).Msg("finalizing recording")

	if err := os.MkdirAll(destDir, 0755); err != nil {
		m.log.Error().Err(err).Str("process_id", proc.ID).Str("record_dir", destDir).Msg("failed to create record directory")
		return
	}

	if name == "" {
		name = "recording_" + time.Now().Format("20060102_1504")
	}

	destPath := filepath.Join(destDir, name+".mp4")
	for i := 1; ; i++ {
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			break
		}
		destPath = filepath.Join(destDir, fmt.Sprintf("%s_%d.mp4", name, i))
	}

	if err := os.Rename(proc.OutputPath, destPath); err != nil {
		m.log.Info().Err(err).Str("process_id", proc.ID).Msg("rename failed (cross-volume), copying instead")
		if err := mgrCopyFile(proc.OutputPath, destPath); err != nil {
			m.log.Error().Err(err).Str("process_id", proc.ID).Str("dest", destPath).Msg("failed to copy recording to destination")
			return
		}
		os.Remove(proc.OutputPath)
	}

	m.log.Info().Str("process_id", proc.ID).Str("path", destPath).Int64("size_bytes", info.Size()).Msg("recording saved")
}

func mgrCopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func mgrSanitizeFilename(title string, t time.Time) string {
	name := mgrNonAlphanumRe.ReplaceAllString(title, "_")
	name = strings.Trim(name, "_")
	if len(name) > 60 {
		name = name[:60]
	}
	if name == "" {
		name = "recording"
	}
	return name + "_" + t.Format("20060102_1504")
}
