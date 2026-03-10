package service

import "github.com/gavinmcnair/tvproxy/pkg/ffmpeg"

// ComposeStreamProfileArgs delegates to the ffmpeg package.
func ComposeStreamProfileArgs(sourceType, hwaccel, videoCodec, container string) string {
	return ffmpeg.ComposeStreamProfileArgs(sourceType, hwaccel, videoCodec, container)
}
