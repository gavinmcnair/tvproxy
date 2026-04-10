# TVProxy

Media stream management and proxy server written in Go. Consolidates streaming sources, manages channels and EPG data, transcodes with hardware acceleration, and integrates with media servers via HDHomeRun emulation and Jellyfin API.

## Features

- **Source Stream Profiles** — Define expected codecs, transport and ffmpeg input settings per source. Supports HTTP, RTSP and local servers.
- **Client Stream Profiles** — Configure output codec, container and delivery per client. Auto copy/transcode based on source vs client comparison.
- **Hardware Transcoding** — Intel QSV, VA-API (Arc), NVIDIA NVENC, Apple VideoToolbox. H.264, H.265/HEVC, AV1 output.
- **Live Playback** — Single ffmpeg dual-output: HLS segments for browser playback + MP4 recording simultaneously. One upstream connection, multiple consumers.
- **VOD Playback** — Seekable HLS with hardware transcoding. WireGuard streams routed through localhost proxy for range request support.
- **Recording** — Click to record during live playback. Scheduled recordings via EPG. Recordings preserved on disk as MP4.
- **Channel Management** — Multi-stream failover, channel groups, per-channel profile assignment.
- **EPG Support** — XMLTV import, auto-match to channels, programme guide grid in the web UI.
- **HDHomeRun Emulation** — Virtual HDHR devices for native Plex/Emby/Jellyfin DVR integration with SSDP discovery.
- **Jellyfin API** — Native Jellyfin server emulation on port 8096 for direct client access.
- **DLNA** — MediaServer for network players (VR headsets, smart TVs, etc).
- **Client Detection** — Auto-detect players via HTTP headers and assign appropriate output profiles.
- **SAT>IP** — DVB terrestrial/satellite/cable tuner integration via SAT>IP protocol.
- **WireGuard VPN** — Built-in WireGuard tunnel with per-source routing.
- **TMDB Integration** — Automatic poster art, metadata, and episode info from The Movie Database.
- **Web Interface** — Built-in SPA with EPG guide, media libraries, in-browser HLS playback, activity monitoring.
- **Authentication** — JWT auth with invite tokens, multi-user support, role-based access.
- **Single Binary** — Everything including the web UI embedded in one Go binary. All data stored in JSON files.

## Quick Start

```bash
go build ./cmd/tvproxy/
TVPROXY_BASE_URL=http://192.168.1.100 ./tvproxy
```

The server starts on port 8080. Default credentials: `admin` / `admin`.

## Docker

```bash
docker run -p 8080:8080 -p 8096:8096 -p 47601-47610:47601-47610 \
  -e TVPROXY_BASE_URL=http://192.168.1.100 \
  -v tvproxy-data:/config -v tvproxy-recordings:/record \
  gavinmcnair/tvproxy:latest
```

For hardware transcoding, pass through the GPU:

```bash
# Intel Arc / QSV / VA-API
docker run ... --device /dev/dri:/dev/dri gavinmcnair/tvproxy:latest

# NVIDIA (requires nvidia-container-toolkit)
docker run ... --gpus all gavinmcnair/tvproxy:latest
```

## Configuration

Key environment variables:

| Variable | Default | Description |
|---|---|---|
| `TVPROXY_BASE_URL` | _(required)_ | Base URL (e.g. `http://192.168.1.100`) |
| `TVPROXY_PORT` | `8080` | Listen port |
| `TVPROXY_RECORD_DIR` | `/record` | Recording output directory |
| `TVPROXY_JWT_SECRET` | `change-me-in-production` | JWT signing secret |
| `TVPROXY_API_KEY` | _(empty)_ | Optional API key auth |
| `TVPROXY_USER_AGENT` | `TVProxy` | Upstream request User-Agent |
| `TVPROXY_LOG_LEVEL` | `info` | Log level |

Full API documentation at `/api/docs` (Swagger UI).

## Architecture

```
cmd/tvproxy/         — Entry point, DI wiring, routes
pkg/
  config/            — Environment-based configuration
  ffmpeg/            — FFmpeg argument composition, probe, hardware encoder mapping
  handler/           — HTTP handlers
  hls/               — HLS session management, playlist generation
  session/           — Kafka-style session manager (one ffmpeg per channel, consumer tracking)
  service/           — Business logic (strategy, proxy, VOD, M3U, EPG, recording)
  store/             — JSON-backed in-memory stores
  models/            — Data models (streams, channels, profiles, sources)
  worker/            — Background workers (refresh, SSDP, HDHR, scheduler)
  xtream/            — Xtream Codes API client and metadata cache
  tmdb/              — TMDB client, metadata store, image proxy
  jellyfin/          — Jellyfin API server emulation
  wireguard/         — WireGuard tunnel management
  logocache/         — Image caching proxy
web/dist/            — Vanilla JS SPA (embedded via Go embed.FS)
```

## Development

```bash
make build          # Local binary
make test           # Run all tests
make docker-build   # Multi-arch build (amd64+arm64) and push
make docker-local   # Build for current arch only
```

## Acknowledgements

Inspired by [Threadfin](https://github.com/Threadfin/Threadfin) and [Dispatcharr](https://github.com/Dispatcharr/Dispatcharr).
