package service

import "testing"

func TestComposeStreamProfileArgs(t *testing.T) {
	tests := []struct {
		name       string
		sourceType string
		hwaccel    string
		videoCodec string
		wantCmd    string
		wantArgs   string
	}{
		{
			name:       "direct returns empty",
			sourceType: "direct",
			hwaccel:    "none",
			videoCodec: "copy",
			wantCmd:    "",
			wantArgs:   "",
		},
		// No HW accel - copy
		{
			name:       "satip copy no hwaccel",
			sourceType: "satip",
			hwaccel:    "none",
			videoCodec: "copy",
			wantCmd:    "ffmpeg",
			wantArgs:   "-hide_banner -loglevel warning -i {input} -c:v copy -c:a copy -bsf:v dump_extra -f mpegts pipe:1 -rw_timeout 5000000",
		},
		{
			name:       "m3u copy no hwaccel",
			sourceType: "m3u",
			hwaccel:    "none",
			videoCodec: "copy",
			wantCmd:    "ffmpeg",
			wantArgs:   "-hide_banner -loglevel warning -analyzeduration 1000000 -probesize 1000000 -i {input} -map 0:v:0 -map 0:a:0 -c:v copy -c:a aac -b:a 128k -ac 2 -f mpegts -fflags +genpts -copyts pipe:1",
		},
		// Software encode
		{
			name:       "satip software h264",
			sourceType: "satip",
			hwaccel:    "none",
			videoCodec: "h264",
			wantCmd:    "ffmpeg",
			wantArgs:   "-hide_banner -loglevel warning -i {input} -c:v libx264 -preset fast -c:a copy -bsf:v dump_extra -f mpegts pipe:1 -rw_timeout 5000000",
		},
		{
			name:       "satip software h265",
			sourceType: "satip",
			hwaccel:    "none",
			videoCodec: "h265",
			wantCmd:    "ffmpeg",
			wantArgs:   "-hide_banner -loglevel warning -i {input} -c:v libx265 -preset fast -c:a copy -bsf:v dump_extra -f mpegts pipe:1 -rw_timeout 5000000",
		},
		{
			name:       "m3u software h265",
			sourceType: "m3u",
			hwaccel:    "none",
			videoCodec: "h265",
			wantCmd:    "ffmpeg",
			wantArgs:   "-hide_banner -loglevel warning -analyzeduration 1000000 -probesize 1000000 -i {input} -map 0:v:0 -map 0:a:0 -c:v libx265 -preset fast -c:a aac -b:a 128k -ac 2 -f mpegts -fflags +genpts -copyts pipe:1",
		},
		{
			name:       "m3u software av1",
			sourceType: "m3u",
			hwaccel:    "none",
			videoCodec: "av1",
			wantCmd:    "ffmpeg",
			wantArgs:   "-hide_banner -loglevel warning -analyzeduration 1000000 -probesize 1000000 -i {input} -map 0:v:0 -map 0:a:0 -c:v libaom-av1 -c:a aac -b:a 128k -ac 2 -f mpegts -fflags +genpts -copyts pipe:1",
		},
		// QSV
		{
			name:       "satip qsv h264",
			sourceType: "satip",
			hwaccel:    "qsv",
			videoCodec: "h264",
			wantCmd:    "ffmpeg",
			wantArgs:   "-hide_banner -loglevel warning -hwaccel qsv -hwaccel_output_format qsv -i {input} -c:v h264_qsv -preset fast -c:a copy -bsf:v dump_extra -f mpegts pipe:1 -rw_timeout 5000000",
		},
		{
			name:       "m3u qsv h264",
			sourceType: "m3u",
			hwaccel:    "qsv",
			videoCodec: "h264",
			wantCmd:    "ffmpeg",
			wantArgs:   "-hide_banner -loglevel warning -hwaccel qsv -hwaccel_output_format qsv -analyzeduration 1000000 -probesize 1000000 -i {input} -map 0:v:0 -map 0:a:0 -c:v h264_qsv -preset fast -c:a aac -b:a 128k -ac 2 -f mpegts -fflags +genpts -copyts pipe:1",
		},
		{
			name:       "satip qsv av1",
			sourceType: "satip",
			hwaccel:    "qsv",
			videoCodec: "av1",
			wantCmd:    "ffmpeg",
			wantArgs:   "-hide_banner -loglevel warning -hwaccel qsv -hwaccel_output_format qsv -i {input} -c:v av1_qsv -preset fast -c:a copy -bsf:v dump_extra -f mpegts pipe:1 -rw_timeout 5000000",
		},
		{
			name:       "m3u qsv av1",
			sourceType: "m3u",
			hwaccel:    "qsv",
			videoCodec: "av1",
			wantCmd:    "ffmpeg",
			wantArgs:   "-hide_banner -loglevel warning -hwaccel qsv -hwaccel_output_format qsv -analyzeduration 1000000 -probesize 1000000 -i {input} -map 0:v:0 -map 0:a:0 -c:v av1_qsv -preset fast -c:a aac -b:a 128k -ac 2 -f mpegts -fflags +genpts -copyts pipe:1",
		},
		{
			name:       "satip qsv h265",
			sourceType: "satip",
			hwaccel:    "qsv",
			videoCodec: "h265",
			wantCmd:    "ffmpeg",
			wantArgs:   "-hide_banner -loglevel warning -hwaccel qsv -hwaccel_output_format qsv -i {input} -c:v hevc_qsv -preset fast -c:a copy -bsf:v dump_extra -f mpegts pipe:1 -rw_timeout 5000000",
		},
		// NVENC
		{
			name:       "satip nvenc h264",
			sourceType: "satip",
			hwaccel:    "nvenc",
			videoCodec: "h264",
			wantCmd:    "ffmpeg",
			wantArgs:   "-hide_banner -loglevel warning -hwaccel cuda -hwaccel_output_format cuda -i {input} -c:v h264_nvenc -preset p4 -c:a copy -bsf:v dump_extra -f mpegts pipe:1 -rw_timeout 5000000",
		},
		{
			name:       "m3u nvenc h264",
			sourceType: "m3u",
			hwaccel:    "nvenc",
			videoCodec: "h264",
			wantCmd:    "ffmpeg",
			wantArgs:   "-hide_banner -loglevel warning -hwaccel cuda -hwaccel_output_format cuda -analyzeduration 1000000 -probesize 1000000 -i {input} -map 0:v:0 -map 0:a:0 -c:v h264_nvenc -preset p4 -c:a aac -b:a 128k -ac 2 -f mpegts -fflags +genpts -copyts pipe:1",
		},
		{
			name:       "satip nvenc av1",
			sourceType: "satip",
			hwaccel:    "nvenc",
			videoCodec: "av1",
			wantCmd:    "ffmpeg",
			wantArgs:   "-hide_banner -loglevel warning -hwaccel cuda -hwaccel_output_format cuda -i {input} -c:v av1_nvenc -preset p4 -c:a copy -bsf:v dump_extra -f mpegts pipe:1 -rw_timeout 5000000",
		},
		{
			name:       "m3u nvenc av1",
			sourceType: "m3u",
			hwaccel:    "nvenc",
			videoCodec: "av1",
			wantCmd:    "ffmpeg",
			wantArgs:   "-hide_banner -loglevel warning -hwaccel cuda -hwaccel_output_format cuda -analyzeduration 1000000 -probesize 1000000 -i {input} -map 0:v:0 -map 0:a:0 -c:v av1_nvenc -preset p4 -c:a aac -b:a 128k -ac 2 -f mpegts -fflags +genpts -copyts pipe:1",
		},
		{
			name:       "satip nvenc h265",
			sourceType: "satip",
			hwaccel:    "nvenc",
			videoCodec: "h265",
			wantCmd:    "ffmpeg",
			wantArgs:   "-hide_banner -loglevel warning -hwaccel cuda -hwaccel_output_format cuda -i {input} -c:v hevc_nvenc -preset p4 -c:a copy -bsf:v dump_extra -f mpegts pipe:1 -rw_timeout 5000000",
		},
		// VA-API (SW decode + HW encode with hwupload filter)
		{
			name:       "satip vaapi h264",
			sourceType: "satip",
			hwaccel:    "vaapi",
			videoCodec: "h264",
			wantCmd:    "ffmpeg",
			wantArgs:   "-hide_banner -loglevel warning -init_hw_device vaapi=va:/dev/dri/renderD128 -filter_hw_device va -i {input} -vf format=nv12,hwupload -c:v h264_vaapi -c:a copy -bsf:v dump_extra -f mpegts pipe:1 -rw_timeout 5000000",
		},
		{
			name:       "m3u vaapi h265",
			sourceType: "m3u",
			hwaccel:    "vaapi",
			videoCodec: "h265",
			wantCmd:    "ffmpeg",
			wantArgs:   "-hide_banner -loglevel warning -init_hw_device vaapi=va:/dev/dri/renderD128 -filter_hw_device va -analyzeduration 1000000 -probesize 1000000 -i {input} -map 0:v:0 -map 0:a:0 -vf format=nv12,hwupload -c:v hevc_vaapi -c:a aac -b:a 128k -ac 2 -f mpegts -fflags +genpts -copyts pipe:1",
		},
		{
			name:       "satip vaapi av1",
			sourceType: "satip",
			hwaccel:    "vaapi",
			videoCodec: "av1",
			wantCmd:    "ffmpeg",
			wantArgs:   "-hide_banner -loglevel warning -init_hw_device vaapi=va:/dev/dri/renderD128 -filter_hw_device va -i {input} -vf format=nv12,hwupload -c:v av1_vaapi -global_quality 30 -b:v 0 -c:a copy -bsf:v dump_extra -f mpegts pipe:1 -rw_timeout 5000000",
		},
		{
			name:       "m3u vaapi av1",
			sourceType: "m3u",
			hwaccel:    "vaapi",
			videoCodec: "av1",
			wantCmd:    "ffmpeg",
			wantArgs:   "-hide_banner -loglevel warning -init_hw_device vaapi=va:/dev/dri/renderD128 -filter_hw_device va -analyzeduration 1000000 -probesize 1000000 -i {input} -map 0:v:0 -map 0:a:0 -vf format=nv12,hwupload -c:v av1_vaapi -global_quality 30 -b:v 0 -c:a aac -b:a 128k -ac 2 -f mpegts -fflags +genpts -copyts pipe:1",
		},
		{
			name:       "m3u vaapi copy (no hwupload needed)",
			sourceType: "m3u",
			hwaccel:    "vaapi",
			videoCodec: "copy",
			wantCmd:    "ffmpeg",
			wantArgs:   "-hide_banner -loglevel warning -init_hw_device vaapi=va:/dev/dri/renderD128 -filter_hw_device va -analyzeduration 1000000 -probesize 1000000 -i {input} -map 0:v:0 -map 0:a:0 -c:v copy -c:a aac -b:a 128k -ac 2 -f mpegts -fflags +genpts -copyts pipe:1",
		},
		// VideoToolbox
		{
			name:       "satip videotoolbox h264",
			sourceType: "satip",
			hwaccel:    "videotoolbox",
			videoCodec: "h264",
			wantCmd:    "ffmpeg",
			wantArgs:   "-hide_banner -loglevel warning -hwaccel videotoolbox -hwaccel_output_format videotoolbox_vld -i {input} -c:v h264_videotoolbox -c:a copy -bsf:v dump_extra -f mpegts pipe:1 -rw_timeout 5000000",
		},
		{
			name:       "m3u videotoolbox h265",
			sourceType: "m3u",
			hwaccel:    "videotoolbox",
			videoCodec: "h265",
			wantCmd:    "ffmpeg",
			wantArgs:   "-hide_banner -loglevel warning -hwaccel videotoolbox -hwaccel_output_format videotoolbox_vld -analyzeduration 1000000 -probesize 1000000 -i {input} -map 0:v:0 -map 0:a:0 -c:v hevc_videotoolbox -c:a aac -b:a 128k -ac 2 -f mpegts -fflags +genpts -copyts pipe:1",
		},
		{
			name:       "satip videotoolbox av1 falls back to software",
			sourceType: "satip",
			hwaccel:    "videotoolbox",
			videoCodec: "av1",
			wantCmd:    "ffmpeg",
			wantArgs:   "-hide_banner -loglevel warning -hwaccel videotoolbox -hwaccel_output_format videotoolbox_vld -i {input} -c:v libaom-av1 -c:a copy -bsf:v dump_extra -f mpegts pipe:1 -rw_timeout 5000000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCmd, gotArgs := ComposeStreamProfileArgs(tt.sourceType, tt.hwaccel, tt.videoCodec)
			if gotCmd != tt.wantCmd {
				t.Errorf("command = %q, want %q", gotCmd, tt.wantCmd)
			}
			if gotArgs != tt.wantArgs {
				t.Errorf("args = %q, want %q", gotArgs, tt.wantArgs)
			}
		})
	}
}
