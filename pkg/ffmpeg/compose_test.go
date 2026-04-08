package ffmpeg

import (
	"testing"
)

func TestBuild(t *testing.T) {
	probeH264Progressive := &ProbeResult{
		HasVideo:    true,
		Video:       &VideoInfo{Codec: "h264", FieldOrder: "progressive"},
		AudioTracks: []AudioTrack{{Codec: "aac"}},
	}
	probeH264Interlaced := &ProbeResult{
		HasVideo:    true,
		Video:       &VideoInfo{Codec: "h264", FieldOrder: "tt"},
		AudioTracks: []AudioTrack{{Codec: "aac"}},
	}
	probeMpeg2 := &ProbeResult{
		HasVideo:    true,
		Video:       &VideoInfo{Codec: "mpeg2video", FieldOrder: "progressive"},
		AudioTracks: []AudioTrack{{Codec: "mp2"}},
	}
	probeMpeg2Interlaced := &ProbeResult{
		HasVideo:    true,
		Video:       &VideoInfo{Codec: "mpeg2video", FieldOrder: "tt"},
		AudioTracks: []AudioTrack{{Codec: "mp2"}},
	}
	probeHevcProgressive := &ProbeResult{
		HasVideo:    true,
		Video:       &VideoInfo{Codec: "hevc", FieldOrder: "progressive"},
		AudioTracks: []AudioTrack{{Codec: "aac"}},
	}
	probeHevcInterlaced := &ProbeResult{
		HasVideo:    true,
		Video:       &VideoInfo{Codec: "hevc", FieldOrder: "tt"},
		AudioTracks: []AudioTrack{{Codec: "aac"}},
	}
	probeRadio := &ProbeResult{
		HasVideo:    false,
		AudioTracks: []AudioTrack{{Codec: "aac"}},
	}

	tests := []struct {
		name    string
		opts    BuildOptions
		wantCmd string
		want    string
	}{
		{
			name:    "custom command passthrough",
			opts:    BuildOptions{CustomCommand: "-i {input} -c copy pipe:1"},
			wantCmd: "ffmpeg",
			want:    "-i {input} -c copy pipe:1",
		},
		{
			name: "RTSP copy nil probe mpegts",
			opts: BuildOptions{StreamURL: "rtsp://example.com/stream", VideoCodec: "copy", Container: "mpegts"},
			want: "-hide_banner -loglevel warning -nostdin -rtsp_transport tcp -analyzeduration 3000000 -probesize 2000000 -max_delay 500000 -err_detect ignore_err -fflags +genpts+discardcorrupt -i {input} -map 0:v:0? -map 0:a:0? -max_muxing_queue_size 4096 -c:v copy -g 50 -keyint_min 50 -c:a aac -ac 2 -b:a 192k -af aresample=async=1000:first_pts=0 -output_ts_offset 0 -f mpegts pipe:1",
		},
		{
			name: "HTTP copy nil probe mpegts",
			opts: BuildOptions{StreamURL: "http://example.com/stream", VideoCodec: "copy", Container: "mpegts"},
			want: "-hide_banner -loglevel warning -nostdin -analyzeduration 1000000 -probesize 1000000 -err_detect ignore_err -fflags +genpts+discardcorrupt -i {input} -map 0:v:0? -map 0:a:0? -max_muxing_queue_size 4096 -c:v copy -g 50 -keyint_min 50 -c:a aac -ac 2 -b:a 192k -output_ts_offset 0 -f mpegts pipe:1",
		},
		{
			name: "HTTP explicit h264 software nil probe",
			opts: BuildOptions{StreamURL: "http://example.com/stream", VideoCodec: "h264", Container: "mpegts"},
			want: "-hide_banner -loglevel warning -nostdin -analyzeduration 1000000 -probesize 1000000 -err_detect ignore_err -fflags +genpts+discardcorrupt -i {input} -map 0:v:0? -map 0:a:0? -max_muxing_queue_size 4096 -c:v libx264 -tune zerolatency -preset fast -g 50 -keyint_min 50 -c:a aac -ac 2 -b:a 192k -output_ts_offset 0 -f mpegts pipe:1",
		},
		{
			name: "HTTP explicit h265 software nil probe",
			opts: BuildOptions{StreamURL: "http://example.com/stream", VideoCodec: "h265", Container: "mpegts"},
			want: "-hide_banner -loglevel warning -nostdin -analyzeduration 1000000 -probesize 1000000 -err_detect ignore_err -fflags +genpts+discardcorrupt -i {input} -map 0:v:0? -map 0:a:0? -max_muxing_queue_size 4096 -c:v libx265 -preset fast -g 50 -keyint_min 50 -c:a aac -ac 2 -b:a 192k -output_ts_offset 0 -f mpegts pipe:1",
		},
		{
			name: "RTSP explicit h265 software nil probe",
			opts: BuildOptions{StreamURL: "rtsp://example.com/stream", VideoCodec: "h265", Container: "mpegts"},
			want: "-hide_banner -loglevel warning -nostdin -rtsp_transport tcp -analyzeduration 3000000 -probesize 2000000 -max_delay 500000 -err_detect ignore_err -fflags +genpts+discardcorrupt -i {input} -map 0:v:0? -map 0:a:0? -max_muxing_queue_size 4096 -c:v libx265 -preset fast -g 50 -keyint_min 50 -c:a aac -ac 2 -b:a 192k -af aresample=async=1000:first_pts=0 -output_ts_offset 0 -f mpegts pipe:1",
		},
		{
			name: "probe mpeg2video forces h264 transcode",
			opts: BuildOptions{StreamURL: "http://example.com/stream", Probe: probeMpeg2, VideoCodec: "copy", Container: "mpegts"},
			want: "-hide_banner -loglevel warning -nostdin -analyzeduration 1000000 -probesize 1000000 -err_detect ignore_err -fflags +genpts+discardcorrupt -i {input} -map 0:v:0? -map 0:a:0? -max_muxing_queue_size 4096 -c:v libx264 -tune zerolatency -preset fast -g 50 -keyint_min 50 -c:a aac -ac 2 -b:a 192k -output_ts_offset 0 -f mpegts pipe:1",
		},
		{
			name: "probe h264 progressive stays copy",
			opts: BuildOptions{StreamURL: "http://example.com/stream", Probe: probeH264Progressive, VideoCodec: "copy", Container: "mpegts"},
			want: "-hide_banner -loglevel warning -nostdin -analyzeduration 1000000 -probesize 1000000 -err_detect ignore_err -fflags +genpts+discardcorrupt -i {input} -map 0:v:0? -map 0:a:0? -max_muxing_queue_size 4096 -c:v copy -g 50 -keyint_min 50 -c:a copy -output_ts_offset 0 -f mpegts pipe:1",
		},
		{
			name: "probe h264 interlaced forces h264 plus yadif",
			opts: BuildOptions{StreamURL: "http://example.com/stream", Probe: probeH264Interlaced, VideoCodec: "copy", Container: "mpegts"},
			want: "-hide_banner -loglevel warning -nostdin -analyzeduration 1000000 -probesize 1000000 -err_detect ignore_err -fflags +genpts+discardcorrupt -i {input} -map 0:v:0? -map 0:a:0? -max_muxing_queue_size 4096 -vf yadif -c:v libx264 -tune zerolatency -preset fast -g 50 -keyint_min 50 -c:a copy -output_ts_offset 0 -f mpegts pipe:1",
		},
		{
			name: "probe hevc progressive stays copy",
			opts: BuildOptions{StreamURL: "rtsp://example.com/stream", Probe: probeHevcProgressive, VideoCodec: "copy", Container: "mpegts"},
			want: "-hide_banner -loglevel warning -nostdin -rtsp_transport tcp -analyzeduration 3000000 -probesize 2000000 -max_delay 500000 -err_detect ignore_err -fflags +genpts+discardcorrupt -i {input} -map 0:v:0? -map 0:a:0? -max_muxing_queue_size 4096 -c:v copy -g 50 -keyint_min 50 -c:a copy -af aresample=async=1000:first_pts=0 -output_ts_offset 0 -f mpegts pipe:1",
		},
		{
			name: "probe hevc interlaced forces h265 plus yadif",
			opts: BuildOptions{StreamURL: "rtsp://example.com/stream", Probe: probeHevcInterlaced, VideoCodec: "copy", Container: "mpegts"},
			want: "-hide_banner -loglevel warning -nostdin -rtsp_transport tcp -analyzeduration 3000000 -probesize 2000000 -max_delay 500000 -err_detect ignore_err -fflags +genpts+discardcorrupt -i {input} -map 0:v:0? -map 0:a:0? -max_muxing_queue_size 4096 -vf yadif -c:v libx265 -preset fast -g 50 -keyint_min 50 -c:a copy -af aresample=async=1000:first_pts=0 -output_ts_offset 0 -f mpegts pipe:1",
		},
		{
			name: "radio stream omits video map",
			opts: BuildOptions{StreamURL: "http://example.com/stream", Probe: probeRadio, VideoCodec: "copy", Container: "mpegts"},
			want: "-hide_banner -loglevel warning -nostdin -analyzeduration 1000000 -probesize 1000000 -err_detect ignore_err -fflags +genpts+discardcorrupt -i {input} -map 0:a:0? -max_muxing_queue_size 4096 -c:a aac -ac 2 -b:a 192k -output_ts_offset 0 -f adts pipe:1",
		},
		{
			name: "webm container uses opus audio",
			opts: BuildOptions{StreamURL: "http://example.com/stream", VideoCodec: "h264", Container: "webm"},
			want: "-hide_banner -loglevel warning -nostdin -analyzeduration 1000000 -probesize 1000000 -err_detect ignore_err -fflags +genpts+discardcorrupt -i {input} -map 0:v:0? -map 0:a:0? -max_muxing_queue_size 4096 -c:v libx264 -tune zerolatency -preset fast -g 50 -keyint_min 50 -c:a libopus -b:a 192k -output_ts_offset 0 -f webm pipe:1",
		},
		{
			name: "mp4 container uses fragmented movflags",
			opts: BuildOptions{StreamURL: "http://example.com/stream", VideoCodec: "copy", Container: "mp4"},
			want: "-hide_banner -loglevel warning -nostdin -analyzeduration 1000000 -probesize 1000000 -err_detect ignore_err -fflags +genpts+discardcorrupt -i {input} -map 0:v:0? -map 0:a:0? -max_muxing_queue_size 4096 -c:v copy -g 50 -keyint_min 50 -c:a aac -ac 2 -b:a 192k -output_ts_offset 0 -f mp4 -movflags frag_keyframe+empty_moov+default_base_moof pipe:1",
		},
		{
			name: "matroska container",
			opts: BuildOptions{StreamURL: "http://example.com/stream", VideoCodec: "copy", Container: "matroska"},
			want: "-hide_banner -loglevel warning -nostdin -analyzeduration 1000000 -probesize 1000000 -err_detect ignore_err -fflags +genpts+discardcorrupt -i {input} -map 0:v:0? -map 0:a:0? -max_muxing_queue_size 4096 -c:v copy -g 50 -keyint_min 50 -c:a aac -ac 2 -b:a 192k -output_ts_offset 0 -f matroska pipe:1",
		},
		{
			name: "user agent included before input",
			opts: BuildOptions{StreamURL: "http://example.com/stream", UserAgent: "TestAgent/1.0", VideoCodec: "copy", Container: "mpegts"},
			want: "-hide_banner -loglevel warning -nostdin -analyzeduration 1000000 -probesize 1000000 -user_agent TestAgent/1.0 -err_detect ignore_err -fflags +genpts+discardcorrupt -i {input} -map 0:v:0? -map 0:a:0? -max_muxing_queue_size 4096 -c:v copy -g 50 -keyint_min 50 -c:a aac -ac 2 -b:a 192k -output_ts_offset 0 -f mpegts pipe:1",
		},
		{
			name: "no URL no transport flags",
			opts: BuildOptions{VideoCodec: "copy", Container: "mpegts"},
			want: "-hide_banner -loglevel warning -nostdin -err_detect ignore_err -fflags +genpts+discardcorrupt -i {input} -map 0:v:0? -map 0:a:0? -max_muxing_queue_size 4096 -c:v copy -g 50 -keyint_min 50 -c:a aac -ac 2 -b:a 192k -output_ts_offset 0 -f mpegts pipe:1",
		},
		{
			name: "QSV h264 RTSP",
			opts: BuildOptions{StreamURL: "rtsp://example.com/stream", HWAccel: "qsv", VideoCodec: "h264", Container: "mpegts"},
			want: "-hide_banner -loglevel warning -nostdin -init_hw_device vaapi=va:/dev/dri/renderD128 -init_hw_device qsv=qs@va -hwaccel qsv -hwaccel_output_format qsv -rtsp_transport tcp -analyzeduration 3000000 -probesize 2000000 -max_delay 500000 -err_detect ignore_err -fflags +genpts+discardcorrupt -i {input} -map 0:v:0? -map 0:a:0? -max_muxing_queue_size 4096 -c:v h264_qsv -preset veryslow -global_quality 20 -g 50 -keyint_min 50 -c:a aac -ac 2 -b:a 192k -af aresample=async=1000:first_pts=0 -output_ts_offset 0 -f mpegts pipe:1",
		},
		{
			name: "QSV h264 HTTP",
			opts: BuildOptions{StreamURL: "http://example.com/stream", HWAccel: "qsv", VideoCodec: "h264", Container: "mpegts"},
			want: "-hide_banner -loglevel warning -nostdin -init_hw_device vaapi=va:/dev/dri/renderD128 -init_hw_device qsv=qs@va -hwaccel qsv -hwaccel_output_format qsv -analyzeduration 1000000 -probesize 1000000 -err_detect ignore_err -fflags +genpts+discardcorrupt -i {input} -map 0:v:0? -map 0:a:0? -max_muxing_queue_size 4096 -c:v h264_qsv -preset veryslow -global_quality 20 -g 50 -keyint_min 50 -c:a aac -ac 2 -b:a 192k -output_ts_offset 0 -f mpegts pipe:1",
		},
		{
			name: "QSV h265 RTSP",
			opts: BuildOptions{StreamURL: "rtsp://example.com/stream", HWAccel: "qsv", VideoCodec: "h265", Container: "mpegts"},
			want: "-hide_banner -loglevel warning -nostdin -init_hw_device vaapi=va:/dev/dri/renderD128 -init_hw_device qsv=qs@va -hwaccel qsv -hwaccel_output_format qsv -rtsp_transport tcp -analyzeduration 3000000 -probesize 2000000 -max_delay 500000 -err_detect ignore_err -fflags +genpts+discardcorrupt -i {input} -map 0:v:0? -map 0:a:0? -max_muxing_queue_size 4096 -c:v hevc_qsv -preset veryslow -global_quality 22 -g 50 -keyint_min 50 -c:a aac -ac 2 -b:a 192k -af aresample=async=1000:first_pts=0 -output_ts_offset 0 -f mpegts pipe:1",
		},
		{
			name: "QSV av1 matroska RTSP",
			opts: BuildOptions{StreamURL: "rtsp://example.com/stream", HWAccel: "qsv", VideoCodec: "av1", Container: "matroska"},
			want: "-hide_banner -loglevel warning -nostdin -init_hw_device vaapi=va:/dev/dri/renderD128 -init_hw_device qsv=qs@va -hwaccel qsv -hwaccel_output_format qsv -rtsp_transport tcp -analyzeduration 3000000 -probesize 2000000 -max_delay 500000 -err_detect ignore_err -fflags +genpts+discardcorrupt -i {input} -map 0:v:0? -map 0:a:0? -max_muxing_queue_size 4096 -c:v av1_qsv -preset veryslow -global_quality 25 -g 50 -keyint_min 50 -c:a aac -ac 2 -b:a 192k -af aresample=async=1000:first_pts=0 -output_ts_offset 0 -f matroska pipe:1",
		},
		{
			name: "NVENC h264 RTSP",
			opts: BuildOptions{StreamURL: "rtsp://example.com/stream", HWAccel: "nvenc", VideoCodec: "h264", Container: "mpegts"},
			want: "-hide_banner -loglevel warning -nostdin -hwaccel cuda -hwaccel_output_format cuda -rtsp_transport tcp -analyzeduration 3000000 -probesize 2000000 -max_delay 500000 -err_detect ignore_err -fflags +genpts+discardcorrupt -i {input} -map 0:v:0? -map 0:a:0? -max_muxing_queue_size 4096 -c:v h264_nvenc -preset p4 -g 50 -keyint_min 50 -c:a aac -ac 2 -b:a 192k -af aresample=async=1000:first_pts=0 -output_ts_offset 0 -f mpegts pipe:1",
		},
		{
			name: "NVENC h265 RTSP",
			opts: BuildOptions{StreamURL: "rtsp://example.com/stream", HWAccel: "nvenc", VideoCodec: "h265", Container: "mpegts"},
			want: "-hide_banner -loglevel warning -nostdin -hwaccel cuda -hwaccel_output_format cuda -rtsp_transport tcp -analyzeduration 3000000 -probesize 2000000 -max_delay 500000 -err_detect ignore_err -fflags +genpts+discardcorrupt -i {input} -map 0:v:0? -map 0:a:0? -max_muxing_queue_size 4096 -c:v hevc_nvenc -preset p4 -g 50 -keyint_min 50 -c:a aac -ac 2 -b:a 192k -af aresample=async=1000:first_pts=0 -output_ts_offset 0 -f mpegts pipe:1",
		},
		{
			name: "NVENC av1 matroska RTSP",
			opts: BuildOptions{StreamURL: "rtsp://example.com/stream", HWAccel: "nvenc", VideoCodec: "av1", Container: "matroska"},
			want: "-hide_banner -loglevel warning -nostdin -hwaccel cuda -hwaccel_output_format cuda -rtsp_transport tcp -analyzeduration 3000000 -probesize 2000000 -max_delay 500000 -err_detect ignore_err -fflags +genpts+discardcorrupt -i {input} -map 0:v:0? -map 0:a:0? -max_muxing_queue_size 4096 -c:v av1_nvenc -preset p4 -cq 24 -pix_fmt p010le -g 50 -keyint_min 50 -c:a aac -ac 2 -b:a 192k -af aresample=async=1000:first_pts=0 -output_ts_offset 0 -f matroska pipe:1",
		},
		{
			name: "VAAPI h264 HTTP with hwupload filter",
			opts: BuildOptions{StreamURL: "http://example.com/stream", HWAccel: "vaapi", VideoCodec: "h264", Container: "mpegts"},
			want: "-hide_banner -loglevel warning -nostdin -init_hw_device vaapi=va:/dev/dri/renderD128 -filter_hw_device va -analyzeduration 1000000 -probesize 1000000 -err_detect ignore_err -fflags +genpts+discardcorrupt -i {input} -map 0:v:0? -map 0:a:0? -max_muxing_queue_size 4096 -vf format=nv12,hwupload -c:v h264_vaapi -g 50 -keyint_min 50 -c:a aac -ac 2 -b:a 192k -output_ts_offset 0 -f mpegts pipe:1",
		},
		{
			name: "VAAPI h265 HTTP with hwupload filter",
			opts: BuildOptions{StreamURL: "http://example.com/stream", HWAccel: "vaapi", VideoCodec: "h265", Container: "mpegts"},
			want: "-hide_banner -loglevel warning -nostdin -init_hw_device vaapi=va:/dev/dri/renderD128 -filter_hw_device va -analyzeduration 1000000 -probesize 1000000 -err_detect ignore_err -fflags +genpts+discardcorrupt -i {input} -map 0:v:0? -map 0:a:0? -max_muxing_queue_size 4096 -vf format=nv12,hwupload -c:v hevc_vaapi -g 50 -keyint_min 50 -c:a aac -ac 2 -b:a 192k -output_ts_offset 0 -f mpegts pipe:1",
		},
		{
			name: "VAAPI av1 matroska uses software decode with hwupload filter",
			opts: BuildOptions{StreamURL: "rtsp://example.com/stream", HWAccel: "vaapi", VideoCodec: "av1", Container: "matroska"},
			want: "-hide_banner -loglevel warning -nostdin -init_hw_device vaapi=va:/dev/dri/renderD128 -filter_hw_device va -rtsp_transport tcp -analyzeduration 3000000 -probesize 2000000 -max_delay 500000 -err_detect ignore_err -fflags +genpts+discardcorrupt -i {input} -map 0:v:0? -map 0:a:0? -max_muxing_queue_size 4096 -vf format=nv12,hwupload -c:v av1_vaapi -global_quality 25 -rc_mode ICQ -g 50 -keyint_min 50 -c:a aac -ac 2 -b:a 192k -af aresample=async=1000:first_pts=0 -output_ts_offset 0 -f matroska pipe:1",
		},
		{
			name: "VideoToolbox h264 RTSP software decode hardware encode",
			opts: BuildOptions{StreamURL: "rtsp://example.com/stream", HWAccel: "videotoolbox", VideoCodec: "h264", Container: "mpegts"},
			want: "-hide_banner -loglevel warning -nostdin -rtsp_transport tcp -analyzeduration 3000000 -probesize 2000000 -max_delay 500000 -err_detect ignore_err -fflags +genpts+discardcorrupt -i {input} -map 0:v:0? -map 0:a:0? -max_muxing_queue_size 4096 -c:v h264_videotoolbox -realtime 1 -g 50 -keyint_min 50 -c:a aac -ac 2 -b:a 192k -af aresample=async=1000:first_pts=0 -output_ts_offset 0 -f mpegts pipe:1",
		},
		{
			name: "VideoToolbox h265 RTSP software decode hardware encode",
			opts: BuildOptions{StreamURL: "rtsp://example.com/stream", HWAccel: "videotoolbox", VideoCodec: "h265", Container: "mpegts"},
			want: "-hide_banner -loglevel warning -nostdin -rtsp_transport tcp -analyzeduration 3000000 -probesize 2000000 -max_delay 500000 -err_detect ignore_err -fflags +genpts+discardcorrupt -i {input} -map 0:v:0? -map 0:a:0? -max_muxing_queue_size 4096 -c:v hevc_videotoolbox -g 50 -keyint_min 50 -c:a aac -ac 2 -b:a 192k -af aresample=async=1000:first_pts=0 -output_ts_offset 0 -f mpegts pipe:1",
		},
		{
			name: "VideoToolbox av1 falls back to software encode no hwdownload",
			opts: BuildOptions{StreamURL: "rtsp://example.com/stream", HWAccel: "videotoolbox", VideoCodec: "av1", Container: "matroska"},
			want: "-hide_banner -loglevel warning -nostdin -rtsp_transport tcp -analyzeduration 3000000 -probesize 2000000 -max_delay 500000 -err_detect ignore_err -fflags +genpts+discardcorrupt -i {input} -map 0:v:0? -map 0:a:0? -max_muxing_queue_size 4096 -c:v libsvtav1 -preset 6 -crf 24 -pix_fmt yuv420p10le -g 50 -keyint_min 50 -c:a aac -ac 2 -b:a 192k -af aresample=async=1000:first_pts=0 -output_ts_offset 0 -f matroska pipe:1",
		},
		{
			name: "VAAPI h264 auto-detect interlaced adds yadif before hwupload",
			opts: BuildOptions{StreamURL: "http://example.com/stream", HWAccel: "vaapi", Probe: probeH264Interlaced, VideoCodec: "copy", Container: "mpegts"},
			want: "-hide_banner -loglevel warning -nostdin -init_hw_device vaapi=va:/dev/dri/renderD128 -filter_hw_device va -analyzeduration 1000000 -probesize 1000000 -err_detect ignore_err -fflags +genpts+discardcorrupt -i {input} -map 0:v:0? -map 0:a:0? -max_muxing_queue_size 4096 -vf yadif,format=nv12,hwupload -c:v h264_vaapi -g 50 -keyint_min 50 -c:a copy -output_ts_offset 0 -f mpegts pipe:1",
		},
		{
			name: "QSV hevc auto-detect interlaced uses vpp_qsv",
			opts: BuildOptions{StreamURL: "http://example.com/stream", HWAccel: "qsv", Probe: probeHevcInterlaced, VideoCodec: "copy", Container: "mpegts"},
			want: "-hide_banner -loglevel warning -nostdin -init_hw_device vaapi=va:/dev/dri/renderD128 -init_hw_device qsv=qs@va -hwaccel qsv -hwaccel_output_format qsv -analyzeduration 1000000 -probesize 1000000 -err_detect ignore_err -fflags +genpts+discardcorrupt -i {input} -map 0:v:0? -map 0:a:0? -max_muxing_queue_size 4096 -vf vpp_qsv=deinterlace_mode=advanced -c:v hevc_qsv -preset veryslow -global_quality 22 -g 50 -keyint_min 50 -c:a copy -output_ts_offset 0 -f mpegts pipe:1",
		},
		{
			name: "NVENC h264 auto-detect interlaced uses yadif_cuda",
			opts: BuildOptions{StreamURL: "rtsp://example.com/stream", HWAccel: "nvenc", Probe: probeH264Interlaced, VideoCodec: "copy", Container: "mpegts"},
			want: "-hide_banner -loglevel warning -nostdin -hwaccel cuda -hwaccel_output_format cuda -rtsp_transport tcp -analyzeduration 3000000 -probesize 2000000 -max_delay 500000 -err_detect ignore_err -fflags +genpts+discardcorrupt -i {input} -map 0:v:0? -map 0:a:0? -max_muxing_queue_size 4096 -vf yadif_cuda -c:v h264_nvenc -preset p4 -g 50 -keyint_min 50 -c:a copy -af aresample=async=1000:first_pts=0 -output_ts_offset 0 -f mpegts pipe:1",
		},
		{
			name: "VideoToolbox av1 interlaced uses software decode then yadif",
			opts: BuildOptions{StreamURL: "rtsp://example.com/stream", HWAccel: "videotoolbox", Probe: probeMpeg2Interlaced, VideoCodec: "av1", Container: "matroska"},
			want: "-hide_banner -loglevel warning -nostdin -rtsp_transport tcp -analyzeduration 3000000 -probesize 2000000 -max_delay 500000 -err_detect ignore_err -fflags +genpts+discardcorrupt -i {input} -map 0:v:0? -map 0:a:0? -max_muxing_queue_size 4096 -vf yadif -c:v libsvtav1 -preset 6 -crf 24 -pix_fmt yuv420p10le -g 50 -keyint_min 50 -c:a aac -ac 2 -b:a 192k -af aresample=async=1000:first_pts=0 -output_ts_offset 0 -f matroska pipe:1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, got := Build(tt.opts)
			if tt.wantCmd != "" && cmd != tt.wantCmd {
				t.Errorf("command: got %q, want %q", cmd, tt.wantCmd)
			}
			if got != tt.want {
				t.Errorf("got  = %q\nwant = %q", got, tt.want)
			}
		})
	}
}

func TestResolveOutputCodec(t *testing.T) {
	tests := []struct {
		name       string
		probe      *ProbeResult
		videoCodec string
		want       string
	}{
		{"explicit h264 wins over probe", nil, "h264", "h264"},
		{"explicit h265 wins over probe", nil, "h265", "h265"},
		{"explicit av1 wins over probe", nil, "av1", "av1"},
		{"nil probe copy stays copy", nil, "copy", "copy"},
		{"nil probe empty stays copy", nil, "", "copy"},
		{"no video in probe stays copy", &ProbeResult{HasVideo: false}, "copy", "copy"},
		{"mpeg2video forces h264", &ProbeResult{HasVideo: true, Video: &VideoInfo{Codec: "mpeg2video"}}, "copy", "h264"},
		{"h264 progressive stays copy", &ProbeResult{HasVideo: true, Video: &VideoInfo{Codec: "h264", FieldOrder: "progressive"}}, "copy", "copy"},
		{"h264 interlaced forces h264", &ProbeResult{HasVideo: true, Video: &VideoInfo{Codec: "h264", FieldOrder: "tt"}}, "copy", "h264"},
		{"h264 interlaced bb forces h264", &ProbeResult{HasVideo: true, Video: &VideoInfo{Codec: "h264", FieldOrder: "bb"}}, "copy", "h264"},
		{"hevc progressive stays copy", &ProbeResult{HasVideo: true, Video: &VideoInfo{Codec: "hevc", FieldOrder: "progressive"}}, "copy", "copy"},
		{"hevc interlaced forces h265", &ProbeResult{HasVideo: true, Video: &VideoInfo{Codec: "hevc", FieldOrder: "tt"}}, "copy", "h265"},
		{"unknown codec stays copy", &ProbeResult{HasVideo: true, Video: &VideoInfo{Codec: "vp9"}}, "copy", "copy"},
		{"explicit codec overrides interlaced probe", &ProbeResult{HasVideo: true, Video: &VideoInfo{Codec: "h264", FieldOrder: "tt"}}, "h265", "h265"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveOutputCodec(tt.probe, tt.videoCodec)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
