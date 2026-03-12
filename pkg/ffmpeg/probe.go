package ffmpeg

import (
	"context"
	"encoding/json"
	"math"
	"os/exec"
	"strconv"
	"time"
)

type ProbeResult struct {
	Duration float64 `json:"duration"`
	IsVOD    bool    `json:"is_vod"`
	Width    int     `json:"width"`
	Height   int     `json:"height"`
}

type ffprobeOutput struct {
	Format  ffprobeFormat   `json:"format"`
	Streams []ffprobeStream `json:"streams"`
}

type ffprobeFormat struct {
	Duration string `json:"duration"`
}

type ffprobeStream struct {
	CodecType string `json:"codec_type"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

func Probe(ctx context.Context, url, userAgent string) (*ProbeResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	args := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
	}
	if userAgent != "" {
		args = append(args, "-user_agent", userAgent)
	}
	args = append(args, url)

	cmd := exec.CommandContext(ctx, "ffprobe", args...)
	out, err := cmd.Output()
	if err != nil {
		return &ProbeResult{IsVOD: false}, nil
	}

	var probe ffprobeOutput
	if err := json.Unmarshal(out, &probe); err != nil {
		return &ProbeResult{IsVOD: false}, nil
	}

	result := &ProbeResult{}

	if probe.Format.Duration != "" {
		d, err := strconv.ParseFloat(probe.Format.Duration, 64)
		if err == nil && d > 0 && !math.IsInf(d, 0) && !math.IsNaN(d) {
			result.Duration = d
			result.IsVOD = true
		}
	}

	for _, s := range probe.Streams {
		if s.CodecType == "video" && s.Width > 0 {
			result.Width = s.Width
			result.Height = s.Height
			break
		}
	}

	return result, nil
}
