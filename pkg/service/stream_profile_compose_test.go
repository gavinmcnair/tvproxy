package service

import "testing"

func TestComposeStreamProfileArgs(t *testing.T) {
	tests := []struct {
		name       string
		sourceType string
		hwaccel    string
		videoCodec string
		container  string
		want       string
	}{
		// No HW accel - copy
		{
			name:       "satip copy no hwaccel",
			sourceType: "satip",
			hwaccel:    "none",
			videoCodec: "copy",
			container:  "mpegts",
			want:       "-hide_banner -loglevel warning -i {input} -c:v copy -c:a copy -bsf:v dump_extra -f mpegts pipe:1 -rw_timeout 5000000",
		},
		{
			name:       "m3u copy no hwaccel",
			sourceType: "m3u",
			hwaccel:    "none",
			videoCodec: "copy",
			container:  "mpegts",
			want:       "-hide_banner -loglevel warning -analyzeduration 1000000 -probesize 1000000 -i {input} -map 0:v -map 0:a:0 -c:v copy -c:a aac -b:a 192k -ac 2 -c:s copy -f mpegts -fflags +genpts -copyts pipe:1",
		},
		// Software encode
		{
			name:       "satip software h264",
			sourceType: "satip",
			hwaccel:    "none",
			videoCodec: "h264",
			container:  "mpegts",
			want:       "-hide_banner -loglevel warning -i {input} -c:v libx264 -preset fast -c:a copy -bsf:v dump_extra -f mpegts pipe:1 -rw_timeout 5000000",
		},
		{
			name:       "satip software h265",
			sourceType: "satip",
			hwaccel:    "none",
			videoCodec: "h265",
			container:  "mpegts",
			want:       "-hide_banner -loglevel warning -i {input} -c:v libx265 -preset fast -c:a copy -bsf:v dump_extra -f mpegts pipe:1 -rw_timeout 5000000",
		},
		{
			name:       "m3u software h265",
			sourceType: "m3u",
			hwaccel:    "none",
			videoCodec: "h265",
			container:  "mpegts",
			want:       "-hide_banner -loglevel warning -analyzeduration 1000000 -probesize 1000000 -i {input} -map 0:v:0 -map 0:a:0 -c:v libx265 -preset fast -c:a aac -b:a 192k -ac 2 -c:s copy -f mpegts -fflags +genpts -copyts pipe:1",
		},
		{
			name:       "m3u software av1 matroska",
			sourceType: "m3u",
			hwaccel:    "none",
			videoCodec: "av1",
			container:  "matroska",
			want:       "-hide_banner -loglevel warning -analyzeduration 1000000 -probesize 1000000 -i {input} -map 0:v:0 -map 0:a:0 -c:v libsvtav1 -preset 6 -crf 24 -pix_fmt yuv420p10le -c:a aac -b:a 192k -ac 2 -c:s copy -f matroska -fflags +genpts -copyts pipe:1",
		},
		// QSV
		{
			name:       "satip qsv h264",
			sourceType: "satip",
			hwaccel:    "qsv",
			videoCodec: "h264",
			container:  "mpegts",
			want:       "-hide_banner -loglevel warning -hwaccel qsv -i {input} -c:v h264_qsv -preset veryslow -global_quality 20 -c:a copy -bsf:v dump_extra -f mpegts pipe:1 -rw_timeout 5000000",
		},
		{
			name:       "m3u qsv h264",
			sourceType: "m3u",
			hwaccel:    "qsv",
			videoCodec: "h264",
			container:  "mpegts",
			want:       "-hide_banner -loglevel warning -hwaccel qsv -analyzeduration 1000000 -probesize 1000000 -i {input} -map 0:v:0 -map 0:a:0 -c:v h264_qsv -preset veryslow -global_quality 20 -c:a aac -b:a 192k -ac 2 -c:s copy -f mpegts -fflags +genpts -copyts pipe:1",
		},
		{
			name:       "satip qsv av1 matroska",
			sourceType: "satip",
			hwaccel:    "qsv",
			videoCodec: "av1",
			container:  "matroska",
			want:       "-hide_banner -loglevel warning -init_hw_device qsv=qs:hw,child_device_type=vaapi -hwaccel qsv -i {input} -c:v av1_qsv -preset veryslow -global_quality 25 -look_ahead 1 -pix_fmt p010le -c:a copy -f matroska pipe:1 -rw_timeout 5000000",
		},
		{
			name:       "m3u qsv av1 matroska",
			sourceType: "m3u",
			hwaccel:    "qsv",
			videoCodec: "av1",
			container:  "matroska",
			want:       "-hide_banner -loglevel warning -init_hw_device qsv=qs:hw,child_device_type=vaapi -hwaccel qsv -analyzeduration 1000000 -probesize 1000000 -i {input} -map 0:v:0 -map 0:a:0 -c:v av1_qsv -preset veryslow -global_quality 25 -look_ahead 1 -pix_fmt p010le -c:a aac -b:a 192k -ac 2 -c:s copy -f matroska -fflags +genpts -copyts pipe:1",
		},
		{
			name:       "satip qsv h265",
			sourceType: "satip",
			hwaccel:    "qsv",
			videoCodec: "h265",
			container:  "mpegts",
			want:       "-hide_banner -loglevel warning -hwaccel qsv -i {input} -c:v hevc_qsv -preset veryslow -global_quality 22 -pix_fmt p010le -c:a copy -bsf:v dump_extra -f mpegts pipe:1 -rw_timeout 5000000",
		},
		// NVENC
		{
			name:       "satip nvenc h264",
			sourceType: "satip",
			hwaccel:    "nvenc",
			videoCodec: "h264",
			container:  "mpegts",
			want:       "-hide_banner -loglevel warning -hwaccel cuda -hwaccel_output_format cuda -i {input} -c:v h264_nvenc -preset p4 -c:a copy -bsf:v dump_extra -f mpegts pipe:1 -rw_timeout 5000000",
		},
		{
			name:       "m3u nvenc h264",
			sourceType: "m3u",
			hwaccel:    "nvenc",
			videoCodec: "h264",
			container:  "mpegts",
			want:       "-hide_banner -loglevel warning -hwaccel cuda -hwaccel_output_format cuda -analyzeduration 1000000 -probesize 1000000 -i {input} -map 0:v:0 -map 0:a:0 -c:v h264_nvenc -preset p4 -c:a aac -b:a 192k -ac 2 -c:s copy -f mpegts -fflags +genpts -copyts pipe:1",
		},
		{
			name:       "satip nvenc av1 matroska",
			sourceType: "satip",
			hwaccel:    "nvenc",
			videoCodec: "av1",
			container:  "matroska",
			want:       "-hide_banner -loglevel warning -hwaccel cuda -hwaccel_output_format cuda -i {input} -c:v av1_nvenc -preset p4 -cq 24 -pix_fmt p010le -c:a copy -f matroska pipe:1 -rw_timeout 5000000",
		},
		{
			name:       "m3u nvenc av1 matroska",
			sourceType: "m3u",
			hwaccel:    "nvenc",
			videoCodec: "av1",
			container:  "matroska",
			want:       "-hide_banner -loglevel warning -hwaccel cuda -hwaccel_output_format cuda -analyzeduration 1000000 -probesize 1000000 -i {input} -map 0:v:0 -map 0:a:0 -c:v av1_nvenc -preset p4 -cq 24 -pix_fmt p010le -c:a aac -b:a 192k -ac 2 -c:s copy -f matroska -fflags +genpts -copyts pipe:1",
		},
		{
			name:       "satip nvenc h265",
			sourceType: "satip",
			hwaccel:    "nvenc",
			videoCodec: "h265",
			container:  "mpegts",
			want:       "-hide_banner -loglevel warning -hwaccel cuda -hwaccel_output_format cuda -i {input} -c:v hevc_nvenc -preset p4 -c:a copy -bsf:v dump_extra -f mpegts pipe:1 -rw_timeout 5000000",
		},
		// VA-API (SW decode + HW encode with hwupload filter)
		{
			name:       "satip vaapi h264",
			sourceType: "satip",
			hwaccel:    "vaapi",
			videoCodec: "h264",
			container:  "mpegts",
			want:       "-hide_banner -loglevel warning -init_hw_device vaapi=va:/dev/dri/renderD128 -filter_hw_device va -i {input} -vf format=nv12,hwupload -c:v h264_vaapi -c:a copy -bsf:v dump_extra -f mpegts pipe:1 -rw_timeout 5000000",
		},
		{
			name:       "m3u vaapi h265",
			sourceType: "m3u",
			hwaccel:    "vaapi",
			videoCodec: "h265",
			container:  "mpegts",
			want:       "-hide_banner -loglevel warning -init_hw_device vaapi=va:/dev/dri/renderD128 -filter_hw_device va -analyzeduration 1000000 -probesize 1000000 -i {input} -map 0:v:0 -map 0:a:0 -vf format=nv12,hwupload -c:v hevc_vaapi -c:a aac -b:a 192k -ac 2 -c:s copy -f mpegts -fflags +genpts -copyts pipe:1",
		},
		{
			name:       "satip vaapi av1 matroska",
			sourceType: "satip",
			hwaccel:    "vaapi",
			videoCodec: "av1",
			container:  "matroska",
			want:       "-hide_banner -loglevel warning -hwaccel vaapi -hwaccel_output_format vaapi -vaapi_device /dev/dri/renderD128 -i {input} -c:v av1_vaapi -rc_mode ICQ -global_quality 25 -c:a copy -f matroska pipe:1 -rw_timeout 5000000",
		},
		{
			name:       "m3u vaapi av1 matroska",
			sourceType: "m3u",
			hwaccel:    "vaapi",
			videoCodec: "av1",
			container:  "matroska",
			want:       "-hide_banner -loglevel warning -hwaccel vaapi -hwaccel_output_format vaapi -vaapi_device /dev/dri/renderD128 -analyzeduration 1000000 -probesize 1000000 -i {input} -map 0:v:0 -map 0:a:0 -c:v av1_vaapi -rc_mode ICQ -global_quality 25 -c:a aac -b:a 192k -ac 2 -c:s copy -f matroska -fflags +genpts -copyts pipe:1",
		},
		{
			name:       "m3u vaapi copy (no hwupload needed)",
			sourceType: "m3u",
			hwaccel:    "vaapi",
			videoCodec: "copy",
			container:  "mpegts",
			want:       "-hide_banner -loglevel warning -init_hw_device vaapi=va:/dev/dri/renderD128 -filter_hw_device va -analyzeduration 1000000 -probesize 1000000 -i {input} -map 0:v -map 0:a:0 -c:v copy -c:a aac -b:a 192k -ac 2 -c:s copy -f mpegts -fflags +genpts -copyts pipe:1",
		},
		// VideoToolbox
		{
			name:       "satip videotoolbox h264",
			sourceType: "satip",
			hwaccel:    "videotoolbox",
			videoCodec: "h264",
			container:  "mpegts",
			want:       "-hide_banner -loglevel warning -hwaccel videotoolbox -hwaccel_output_format videotoolbox_vld -i {input} -c:v h264_videotoolbox -c:a copy -bsf:v dump_extra -f mpegts pipe:1 -rw_timeout 5000000",
		},
		{
			name:       "m3u videotoolbox h265",
			sourceType: "m3u",
			hwaccel:    "videotoolbox",
			videoCodec: "h265",
			container:  "mpegts",
			want:       "-hide_banner -loglevel warning -hwaccel videotoolbox -hwaccel_output_format videotoolbox_vld -analyzeduration 1000000 -probesize 1000000 -i {input} -map 0:v:0 -map 0:a:0 -c:v hevc_videotoolbox -c:a aac -b:a 192k -ac 2 -c:s copy -f mpegts -fflags +genpts -copyts pipe:1",
		},
		{
			name:       "satip videotoolbox av1 falls back to software",
			sourceType: "satip",
			hwaccel:    "videotoolbox",
			videoCodec: "av1",
			container:  "matroska",
			want:       "-hide_banner -loglevel warning -hwaccel videotoolbox -hwaccel_output_format videotoolbox_vld -i {input} -c:v libsvtav1 -preset 6 -crf 24 -pix_fmt yuv420p10le -c:a copy -f matroska pipe:1 -rw_timeout 5000000",
		},
		// Container-specific tests
		{
			name:       "m3u copy mp4 fragmented",
			sourceType: "m3u",
			hwaccel:    "none",
			videoCodec: "copy",
			container:  "mp4",
			want:       "-hide_banner -loglevel warning -analyzeduration 1000000 -probesize 1000000 -i {input} -map 0:v -map 0:a:0 -c:v copy -c:a aac -b:a 192k -ac 2 -c:s copy -f mp4 -movflags frag_keyframe+empty_moov+default_base_moof -fflags +genpts pipe:1",
		},
		{
			name:       "m3u copy matroska",
			sourceType: "m3u",
			hwaccel:    "none",
			videoCodec: "copy",
			container:  "matroska",
			want:       "-hide_banner -loglevel warning -analyzeduration 1000000 -probesize 1000000 -i {input} -map 0:v -map 0:a:0 -c:v copy -c:a aac -b:a 192k -ac 2 -c:s copy -f matroska -fflags +genpts -copyts pipe:1",
		},
		{
			name:       "m3u h264 webm uses opus",
			sourceType: "m3u",
			hwaccel:    "none",
			videoCodec: "h264",
			container:  "webm",
			want:       "-hide_banner -loglevel warning -analyzeduration 1000000 -probesize 1000000 -i {input} -map 0:v:0 -map 0:a:0 -c:v libx264 -preset fast -c:a libopus -b:a 192k -ac 2 -c:s copy -f webm -fflags +genpts pipe:1",
		},
		{
			name:       "satip copy matroska no dump_extra",
			sourceType: "satip",
			hwaccel:    "none",
			videoCodec: "copy",
			container:  "matroska",
			want:       "-hide_banner -loglevel warning -i {input} -c:v copy -c:a copy -f matroska pipe:1 -rw_timeout 5000000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComposeStreamProfileArgs(tt.sourceType, tt.hwaccel, tt.videoCodec, tt.container)
			if got != tt.want {
				t.Errorf("got  = %q\nwant = %q", got, tt.want)
			}
		})
	}
}
