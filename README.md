# TVProxy

A media streaming hub that connects your content sources to your devices. Watch on Jellyfin clients, Plex, DLNA players (including VR headsets like Meta Quest), smart TVs, or any browser — TVProxy handles the transcoding, format negotiation, and delivery automatically.

Written in Go. Single binary. No external dependencies.

## What It Does

TVProxy sits between your media sources and your playback devices. It ingests streams from multiple sources, manages channels and EPG data, and serves content to clients in whatever format they need — transcoding with hardware acceleration when required, passing through untouched when possible.

### Client Integrations

- **Jellyfin** — Full Jellyfin API server on port 8096. Connect any Jellyfin client directly — phones, tablets, TVs, Meta Quest, Fire Stick, Apple TV.
- **DLNA** — MediaServer for network players. Works with VR headsets (Quest via Skybox/ALVR), smart TVs (LG, Samsung, Panasonic), and any UPnP/DLNA player.
- **HDHomeRun** — Emulates HDHR devices with SSDP discovery for native Plex/Emby/Jellyfin DVR integration. Multiple virtual tuners, each on its own port.
- **Plex/Emby** — Via HDHomeRun emulation. Shows up as a native DVR source with full guide data.
- **Browser** — Built-in HLS player with live TV, VOD, recording, and EPG guide.

### Source Integrations

- **M3U / Xtream Codes** — Import playlists or connect to Xtream APIs. Automatic periodic refresh.
- **SAT>IP** — DVB-T/T2, DVB-S/S2, DVB-C tuner integration via SAT>IP protocol.
- **TVProxy-streams** — Companion server for serving local media libraries with inline probe data.
- **WireGuard VPN** — Built-in tunnel with per-source routing. Transparent to ffmpeg via localhost proxy.

### Transcoding & Profiles

- **Source Stream Profiles** — Define expected codecs, transport and ffmpeg input settings per source.
- **Client Stream Profiles** — Define what each client needs. The system compares source vs client and automatically decides copy, remux, or transcode.
- **Hardware Acceleration** — Intel QSV, VA-API (Arc A380 etc), NVIDIA NVENC, Apple VideoToolbox. H.264, H.265/HEVC, AV1 output.
- **Live Dual-Output** — Single ffmpeg writes both HLS (for playback) and MP4 (for recording) simultaneously.
- **VOD Seeking** — Seekable HLS with hardware transcoding for on-demand content.

### Management

- **Channels** — Multi-stream failover, groups, per-channel profile assignment.
- **EPG** — XMLTV import, auto-match to channels, programme guide grid.
- **Recording** — One-click recording during live playback. Scheduled recordings via EPG.
- **TMDB** — Automatic poster art, metadata, ratings, and episode info.
- **Activity** — Real-time viewer tracking and session monitoring.
- **Multi-User** — JWT auth, invite tokens, role-based access.

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
