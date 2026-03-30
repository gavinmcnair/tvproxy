package ffmpeg

import (
	"strings"
)

func isRTSPURL(url string) bool {
	return strings.HasPrefix(url, "rtsp://") || strings.HasPrefix(url, "rtsps://")
}

func isHTTPURL(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}

func isInterlaced(fieldOrder string) bool {
	switch fieldOrder {
	case "tt", "bb", "tb", "bt":
		return true
	}
	return false
}

func audioEncoder(probe *ProbeResult) []string {
	s := settings()
	if probe == nil || len(probe.AudioTracks) == 0 {
		return []string{"-c:a", "aac", "-b:a", s.AudioBitrate}
	}
	switch probe.AudioTracks[0].Codec {
	case "aac":
		return []string{"-c:a", "copy"}
	default:
		return []string{"-c:a", "aac", "-b:a", s.AudioBitrate}
	}
}
