package ffmpeg

import "strings"

// ComposeStreamProfileArgs builds the ffmpeg args string from the user-facing
// dropdown values. Returns empty string for "direct" source type.
func ComposeStreamProfileArgs(sourceType, hwaccel, videoCodec, container string) string {
	if sourceType == "direct" {
		return ""
	}

	var parts []string

	// Base flags
	parts = append(parts, "-hide_banner", "-loglevel", "warning")

	// HW accel flags (before -i) — some vary by codec
	switch hwaccel {
	case "qsv":
		if videoCodec == "av1" {
			parts = append(parts, "-init_hw_device", "qsv=qs:hw,child_device_type=vaapi", "-hwaccel", "qsv")
		} else {
			parts = append(parts, "-hwaccel", "qsv")
		}
	case "nvenc":
		parts = append(parts, "-hwaccel", "cuda", "-hwaccel_output_format", "cuda")
	case "vaapi":
		if videoCodec == "av1" {
			parts = append(parts, "-hwaccel", "vaapi", "-hwaccel_output_format", "vaapi", "-vaapi_device", "/dev/dri/renderD128")
		} else {
			parts = append(parts, "-init_hw_device", "vaapi=va:/dev/dri/renderD128", "-filter_hw_device", "va")
		}
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
	// copy: -map 0:v (all video streams, Plex/Threadfin compatible)
	// transcode: -map 0:v:0 (first video only, safe for HW encoder filter chains)
	if sourceType == "m3u" {
		if videoCodec == "copy" {
			parts = append(parts, "-map", "0:v", "-map", "0:a:0")
		} else {
			parts = append(parts, "-map", "0:v:0", "-map", "0:a:0")
		}
	}

	// VA-API with -init_hw_device needs frames uploaded to GPU via filter
	// (av1 uses -hwaccel vaapi which handles upload automatically)
	if hwaccel == "vaapi" && videoCodec != "copy" && videoCodec != "av1" {
		parts = append(parts, "-vf", "format=nv12,hwupload")
	}

	// Video encoder
	parts = append(parts, encoderFlags(hwaccel, videoCodec)...)

	// Audio codec: webm requires opus, everything else uses aac or copy
	switch sourceType {
	case "satip":
		parts = append(parts, "-c:a", "copy")
		if container == "mpegts" {
			parts = append(parts, "-bsf:v", "dump_extra")
		}
	case "m3u":
		if container == "webm" {
			parts = append(parts, "-c:a", "libopus", "-b:a", "192k", "-ac", "2")
		} else {
			parts = append(parts, "-c:a", "aac", "-b:a", "192k", "-ac", "2")
		}
		parts = append(parts, "-c:s", "copy")
	}

	// Container and output flags
	switch container {
	case "mp4":
		parts = append(parts, "-f", "mp4", "-movflags", "frag_keyframe+empty_moov+default_base_moof")
	default:
		parts = append(parts, "-f", container)
	}

	if sourceType == "m3u" {
		parts = append(parts, "-fflags", "+genpts")
		// copyts preserves original timestamps — fine for mpegts/matroska but
		// fragmented mp4/webm need timestamps starting near zero
		if container == "mpegts" || container == "matroska" {
			parts = append(parts, "-copyts")
		}
	}

	parts = append(parts, "pipe:1")

	if sourceType == "satip" {
		parts = append(parts, "-rw_timeout", "5000000")
	}

	return strings.Join(parts, " ")
}

// DefaultContainer returns the sensible default container for a video codec.
func DefaultContainer(videoCodec string) string {
	switch videoCodec {
	case "av1":
		return "matroska"
	default:
		return "mpegts"
	}
}

// encoderFlags returns the -c:v flags for the given hwaccel + videoCodec combination.
func encoderFlags(hwaccel, videoCodec string) []string {
	switch videoCodec {
	case "copy":
		return []string{"-c:v", "copy"}
	case "h264":
		switch hwaccel {
		case "qsv":
			return []string{"-c:v", "h264_qsv", "-preset", "veryslow", "-global_quality", "20"}
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
			return []string{"-c:v", "hevc_qsv", "-preset", "veryslow", "-global_quality", "22", "-pix_fmt", "p010le"}
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
			return []string{"-c:v", "av1_qsv", "-preset", "veryslow", "-global_quality", "25", "-look_ahead", "1", "-pix_fmt", "p010le"}
		case "nvenc":
			return []string{"-c:v", "av1_nvenc", "-preset", "p4", "-cq", "24", "-pix_fmt", "p010le"}
		case "vaapi":
			return []string{"-c:v", "av1_vaapi", "-rc_mode", "ICQ", "-global_quality", "25"}
		default:
			return []string{"-c:v", "libsvtav1", "-preset", "6", "-crf", "24", "-pix_fmt", "yuv420p10le"}
		}
	default:
		return []string{"-c:v", "copy"}
	}
}
