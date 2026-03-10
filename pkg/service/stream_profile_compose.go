package service

import "strings"

// ComposeStreamProfileArgs builds the ffmpeg command and args from the
// user-facing dropdown values: sourceType, hwaccel, and videoCodec.
// Returns (command, args). For "direct" source type, both are empty.
func ComposeStreamProfileArgs(sourceType, hwaccel, videoCodec string) (string, string) {
	if sourceType == "direct" {
		return "", ""
	}

	var parts []string

	// Base flags per source type (everything before -i)
	switch sourceType {
	case "satip":
		parts = append(parts, "-hide_banner", "-loglevel", "warning")
	case "m3u":
		parts = append(parts, "-hide_banner", "-loglevel", "warning")
	}

	// HW accel flags (before -i)
	// VA-API: use -vaapi_device only (SW decode + HW encode is more compatible)
	// QSV/NVENC: use -hwaccel for full HW pipeline
	switch hwaccel {
	case "qsv":
		parts = append(parts, "-hwaccel", "qsv", "-hwaccel_output_format", "qsv")
	case "nvenc":
		parts = append(parts, "-hwaccel", "cuda", "-hwaccel_output_format", "cuda")
	case "vaapi":
		// init_hw_device + filter_hw_device: SW decode + HW encode (most compatible).
		// -vaapi_device was removed in ffmpeg 8.x; this is the modern equivalent.
		parts = append(parts, "-init_hw_device", "vaapi=va:/dev/dri/renderD128", "-filter_hw_device", "va")
	case "videotoolbox":
		parts = append(parts, "-hwaccel", "videotoolbox", "-hwaccel_output_format", "videotoolbox_vld")
	}

	// M3U-specific probe settings (before -i)
	if sourceType == "m3u" {
		parts = append(parts, "-analyzeduration", "1000000", "-probesize", "1000000")
	}

	// Input
	parts = append(parts, "-i", "{input}")

	// M3U-specific mapping
	if sourceType == "m3u" {
		parts = append(parts, "-map", "0:v:0", "-map", "0:a:0")
	}

	// VA-API needs frames uploaded to GPU via filter
	if hwaccel == "vaapi" && videoCodec != "copy" {
		parts = append(parts, "-vf", "format=nv12,hwupload")
	}

	// Video encoder
	parts = append(parts, encoderFlags(hwaccel, videoCodec)...)

	// Audio + output flags per source type
	switch sourceType {
	case "satip":
		parts = append(parts, "-c:a", "copy", "-bsf:v", "dump_extra", "-f", "mpegts", "pipe:1", "-rw_timeout", "5000000")
	case "m3u":
		parts = append(parts, "-c:a", "aac", "-b:a", "128k", "-ac", "2", "-f", "mpegts", "-fflags", "+genpts", "-copyts", "pipe:1")
	}

	return "ffmpeg", strings.Join(parts, " ")
}

// encoderFlags returns the -c:v flags for the given hwaccel + videoCodec combination.
func encoderFlags(hwaccel, videoCodec string) []string {
	switch videoCodec {
	case "copy":
		return []string{"-c:v", "copy"}
	case "h264":
		switch hwaccel {
		case "qsv":
			return []string{"-c:v", "h264_qsv", "-preset", "fast"}
		case "nvenc":
			return []string{"-c:v", "h264_nvenc", "-preset", "p4"}
		case "vaapi":
			return []string{"-c:v", "h264_vaapi"}
		case "videotoolbox":
			return []string{"-c:v", "h264_videotoolbox"}
		default:
			return []string{"-c:v", "libx264", "-preset", "fast"}
		}
	case "h265":
		switch hwaccel {
		case "qsv":
			return []string{"-c:v", "hevc_qsv", "-preset", "fast"}
		case "nvenc":
			return []string{"-c:v", "hevc_nvenc", "-preset", "p4"}
		case "vaapi":
			return []string{"-c:v", "hevc_vaapi"}
		case "videotoolbox":
			return []string{"-c:v", "hevc_videotoolbox"}
		default:
			return []string{"-c:v", "libx265", "-preset", "fast"}
		}
	case "av1":
		switch hwaccel {
		case "qsv":
			return []string{"-c:v", "av1_qsv", "-preset", "fast"}
		case "nvenc":
			return []string{"-c:v", "av1_nvenc", "-preset", "p4"}
		case "vaapi":
			return []string{"-c:v", "av1_vaapi", "-global_quality", "30", "-b:v", "0"}
		default:
			// videotoolbox has no AV1 encoder; falls back to software
			return []string{"-c:v", "libaom-av1"}
		}
	default:
		return []string{"-c:v", "copy"}
	}
}
