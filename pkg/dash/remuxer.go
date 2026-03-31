package dash

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog"
)

type logWriter struct {
	log    zerolog.Logger
	prefix string
	buf    bytes.Buffer
}

func (w *logWriter) Write(p []byte) (int, error) {
	w.buf.Write(p)
	for {
		line, err := w.buf.ReadString('\n')
		if err != nil {
			w.buf.WriteString(line)
			break
		}
		w.log.Warn().Str("src", w.prefix).Msg(line[:len(line)-1])
	}
	return len(p), nil
}

type Remuxer struct {
	inputPath    string
	outputDir    string
	manifestPath string
	cmd          *exec.Cmd
	cancel       context.CancelFunc
	done         chan struct{}
	ready        chan struct{}
	readyOnce    sync.Once
	err          error
	log          zerolog.Logger
}

func NewRemuxer(inputPath, outputDir string, log zerolog.Logger) *Remuxer {
	return &Remuxer{
		inputPath:    inputPath,
		outputDir:    outputDir,
		manifestPath: filepath.Join(outputDir, "manifest.mpd"),
		done:         make(chan struct{}),
		ready:        make(chan struct{}),
		log:          log,
	}
}

func (r *Remuxer) Start(ctx context.Context) error {
	if err := os.MkdirAll(r.outputDir, 0755); err != nil {
		return fmt.Errorf("creating dash output dir: %w", err)
	}

	// Wait for the upstream to write enough fMP4 data for MP4Box to read.
	waitCtx, waitCancel := context.WithTimeout(ctx, 30*time.Second)
	defer waitCancel()
	for {
		info, err := os.Stat(r.inputPath)
		if err == nil && info.Size() > 4096 {
			break
		}
		select {
		case <-waitCtx.Done():
			return fmt.Errorf("upstream file not ready: %w", waitCtx.Err())
		case <-time.After(500 * time.Millisecond):
		}
	}

	rctx, cancel := context.WithCancel(ctx)
	r.cancel = cancel

	// Use MP4Box -ddbg-live (debug live, no time regulation) which reads
	// the growing fMP4 and produces segments as fast as data arrives.
	// Separate video/audio inputs to avoid multiplexed representations.
	r.cmd = exec.CommandContext(rctx, "MP4Box",
		"-ddbg-live", "2000",
		"-rap",
		"-profile", "live",
		"-segment-name", "seg-",
		"-segment-timeline",
		"-time-shift", "30",
		"-min-buffer", "2000",
		"-ast-offset", "-800",
		"-out", r.manifestPath,
		r.inputPath+"#video",
		r.inputPath+"#audio",
	)
	r.cmd.Cancel = func() error {
		return r.cmd.Process.Signal(syscall.SIGTERM)
	}
	r.cmd.WaitDelay = 5 * time.Second
	r.cmd.Stderr = &logWriter{log: r.log, prefix: "mp4box"}
	r.cmd.Stdout = &logWriter{log: r.log, prefix: "mp4box"}

	if err := r.cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("starting MP4Box: %w", err)
	}

	go r.run()
	go r.waitForManifest()

	return nil
}

func (r *Remuxer) run() {
	defer close(r.done)
	err := r.cmd.Wait()
	if err != nil && r.cancel != nil {
		r.err = err
	}
	r.readyOnce.Do(func() { close(r.ready) })
}

func (r *Remuxer) waitForManifest() {
	for {
		select {
		case <-r.done:
			return
		default:
			if _, err := os.Stat(r.manifestPath); err == nil {
				r.readyOnce.Do(func() {
					r.log.Info().Str("manifest", r.manifestPath).Msg("dash manifest ready")
					close(r.ready)
				})
				return
			}
			time.Sleep(200 * time.Millisecond)
		}
	}
}

func (r *Remuxer) WaitReady(ctx context.Context) error {
	select {
	case <-r.ready:
		return r.err
	case <-ctx.Done():
		return ctx.Err()
	case <-r.done:
		if r.err != nil {
			return r.err
		}
		return fmt.Errorf("MP4Box exited before manifest was ready")
	}
}

func (r *Remuxer) ManifestPath() string { return r.manifestPath }
func (r *Remuxer) OutputDir() string    { return r.outputDir }

func (r *Remuxer) IsDone() bool {
	select {
	case <-r.done:
		return true
	default:
		return false
	}
}

func (r *Remuxer) Stop() {
	if r.cancel != nil {
		r.cancel()
	}
	<-r.done
}
