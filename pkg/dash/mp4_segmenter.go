package dash

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

type MP4Segmenter struct {
	index     *MP4Index
	filePath  string
	duration  float64
	startTime time.Time
	done      chan struct{}
	doneOnce  sync.Once
	ready     chan struct{}
	readyOnce sync.Once
	log       zerolog.Logger
}

func NewMP4Segmenter(filePath string, duration float64, log zerolog.Logger) *MP4Segmenter {
	return &MP4Segmenter{
		filePath:  filePath,
		duration:  duration,
		startTime: time.Now().UTC(),
		done:      make(chan struct{}),
		ready:     make(chan struct{}),
		log:       log.With().Str("component", "mp4_segmenter").Logger(),
	}
}

func (s *MP4Segmenter) Start(ctx context.Context) error {
	waitCtx, waitCancel := context.WithTimeout(ctx, 30*time.Second)
	defer waitCancel()
	for {
		info, err := os.Stat(s.filePath)
		if err == nil && info.Size() > 4096 {
			break
		}
		select {
		case <-waitCtx.Done():
			return fmt.Errorf("input file not ready: %w", waitCtx.Err())
		case <-time.After(500 * time.Millisecond):
		}
	}

	idx := NewMP4Index(s.filePath, s.log)
	if err := idx.Start(); err != nil {
		return err
	}
	s.index = idx

	go s.waitForFragments()

	return nil
}

func (s *MP4Segmenter) waitForFragments() {
	for {
		select {
		case <-s.done:
			return
		default:
			if s.index.FragmentCount() > 0 {
				s.readyOnce.Do(func() {
					s.log.Info().Int("fragments", s.index.FragmentCount()).Msg("mp4 segmenter ready")
					close(s.ready)
				})
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
}

func (s *MP4Segmenter) WaitReady(ctx context.Context) error {
	select {
	case <-s.ready:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-s.done:
		return fmt.Errorf("segmenter stopped before ready")
	}
}

func (s *MP4Segmenter) Stop() {
	s.doneOnce.Do(func() {
		close(s.done)
	})
	if s.index != nil {
		s.index.MarkDone()
		s.index.Stop()
	}
}

func (s *MP4Segmenter) IsDone() bool {
	select {
	case <-s.done:
		return true
	default:
		return false
	}
}

func (s *MP4Segmenter) ServeInit() ([]byte, error) {
	data := s.index.InitData()
	if data == nil {
		return nil, fmt.Errorf("init segment not available")
	}
	return data, nil
}

func (s *MP4Segmenter) ServeSegment(number int) ([]byte, error) {
	frag, ok := s.index.Fragment(number)
	if !ok {
		return nil, fmt.Errorf("segment %d not found", number)
	}

	f, err := os.Open(s.filePath)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	if _, err := f.Seek(frag.FileOffset, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seeking to segment: %w", err)
	}

	data := make([]byte, frag.Size)
	if _, err := io.ReadFull(f, data); err != nil {
		return nil, fmt.Errorf("reading segment: %w", err)
	}

	return data, nil
}

func (s *MP4Segmenter) GenerateManifest(duration float64, bufferedSecs float64) []byte {
	tracks := s.index.Tracks()
	frags := s.index.Fragments()

	isComplete := s.index.IsDone() && duration > 0
	isDynamic := !isComplete

	timescale := s.index.VideoTimescale()
	if timescale == 0 {
		timescale = 90000
	}

	var videoCodec, audioCodec string
	for _, t := range tracks {
		if t.HandlerType == "vide" && videoCodec == "" {
			videoCodec = fullCodecString(t.Codec)
		}
		if t.HandlerType == "soun" && audioCodec == "" {
			audioCodec = fullCodecString(t.Codec)
		}
	}
	if videoCodec == "" {
		videoCodec = "avc1.640028"
	}
	if audioCodec == "" {
		audioCodec = "mp4a.40.2"
	}

	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")

	if isDynamic {
		ast := s.startTime
		if bufferedSecs > 0 {
			ast = time.Now().UTC().Add(-time.Duration(bufferedSecs+30) * time.Second)
		}
		buf.WriteString(fmt.Sprintf(`<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" type="dynamic" minimumUpdatePeriod="PT2S" availabilityStartTime="%s"`,
			ast.Format(time.RFC3339)))
		if duration > 0 {
			buf.WriteString(fmt.Sprintf(` mediaPresentationDuration="%s"`, formatISODur(duration)))
		}
		buf.WriteString(` minBufferTime="PT2S" profiles="urn:mpeg:dash:profile:isoff-live:2011">` + "\n")
	} else {
		buf.WriteString(fmt.Sprintf(`<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" type="static" mediaPresentationDuration="%s" minBufferTime="PT2S" profiles="urn:mpeg:dash:profile:isoff-on-demand:2011">`+"\n",
			formatISODur(duration)))
	}

	buf.WriteString(`  <Period id="0" start="PT0S">` + "\n")

	buf.WriteString(`    <AdaptationSet id="0" contentType="video" mimeType="video/mp4" segmentAlignment="true">` + "\n")
	buf.WriteString(fmt.Sprintf(`      <Representation id="v0" codecs="%s" bandwidth="2000000">`+"\n", videoCodec))
	writeTimeline(&buf, timescale, frags)
	buf.WriteString(`      </Representation>` + "\n")
	buf.WriteString(`    </AdaptationSet>` + "\n")

	buf.WriteString(`    <AdaptationSet id="1" contentType="audio" mimeType="audio/mp4" segmentAlignment="true">` + "\n")
	buf.WriteString(fmt.Sprintf(`      <Representation id="a0" codecs="%s" bandwidth="128000">`+"\n", audioCodec))
	writeTimeline(&buf, timescale, frags)
	buf.WriteString(`      </Representation>` + "\n")
	buf.WriteString(`    </AdaptationSet>` + "\n")

	buf.WriteString(`  </Period>` + "\n")
	buf.WriteString(`</MPD>` + "\n")

	return buf.Bytes()
}

func writeTimeline(buf *bytes.Buffer, timescale uint32, frags []FragmentEntry) {
	buf.WriteString(fmt.Sprintf(`        <SegmentTemplate timescale="%d" initialization="init.mp4" media="seg_$Number$.m4s" startNumber="0">`+"\n",
		timescale))
	buf.WriteString(`          <SegmentTimeline>` + "\n")

	type sEntry struct {
		t uint64
		d uint64
		r int
	}

	var entries []sEntry
	for _, f := range frags {
		if len(entries) > 0 {
			last := &entries[len(entries)-1]
			if f.Duration == last.d {
				last.r++
				continue
			}
		}
		entries = append(entries, sEntry{t: f.DecodeTime, d: f.Duration, r: 0})
	}

	for _, e := range entries {
		if e.r > 0 {
			buf.WriteString(fmt.Sprintf(`            <S t="%d" d="%d" r="%d"/>`+"\n", e.t, e.d, e.r))
		} else {
			buf.WriteString(fmt.Sprintf(`            <S t="%d" d="%d"/>`+"\n", e.t, e.d))
		}
	}

	buf.WriteString(`          </SegmentTimeline>` + "\n")
	buf.WriteString(`        </SegmentTemplate>` + "\n")
}

func fullCodecString(codec string) string {
	switch codec {
	case "avc1":
		return "avc1.640028"
	case "hev1":
		return "hev1.1.6.L120.90"
	case "av01":
		return "av01.0.08M.08"
	case "mp4a":
		return "mp4a.40.2"
	case "ac-3":
		return "ac-3"
	case "ec-3":
		return "ec-3"
	default:
		return codec
	}
}

func formatISODur(seconds float64) string {
	h := int(math.Floor(seconds / 3600))
	m := int(math.Floor(math.Mod(seconds, 3600) / 60))
	s := math.Mod(seconds, 60)
	var parts []string
	parts = append(parts, "PT")
	if h > 0 {
		parts = append(parts, fmt.Sprintf("%dH", h))
	}
	if m > 0 {
		parts = append(parts, fmt.Sprintf("%dM", m))
	}
	parts = append(parts, fmt.Sprintf("%.1fS", s))
	return strings.Join(parts, "")
}
