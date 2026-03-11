Certainly! Here’s a concise and clear **AGENT.md** document—suitable as a protocol/design reference for a generic agent emulating an HDHomeRun tuner on the network for local clients such as Plex. This is tailored based on your packet capture and includes connection expectations, request/response patterns, and sample headers.

---

# AGENT.md

## Overview

This agent emulates an HDHomeRun network TV tuner on a local network. It provides HTTP endpoints for auto-discovery, status and control, lineup querying, and live MPEG-TS streaming. The goal is compatibility with clients such as Plex, using simple, connection-per-request HTTP API patterns.

---

## API Endpoints & Protocol

### 1. **Discovery and Device Information**

#### `GET /discover.json`

**Purpose:**  
Clients auto-discover agent device properties.

**Sample Request Headers:**
```
GET /discover.json HTTP/1.1
Host: <agent-ip>
User-Agent: PlexMediaServer/1.43.0.10492-121068a07
Accept: */*
Accept-Encoding: gzip
```

**Sample Response:**
```json
{
 "FriendlyName": "HDHomeRun CONNECT DUO",
 "ModelNumber": "HDHR5-2DT",
 "FirmwareName": "hdhomerun5_dvbt",
 "FirmwareVersion": "20250815",
 "DeviceID": "124205A5",
 "DeviceAuth": "string",
 "BaseURL": "http://<agent-ip>",
 "LineupURL": "http://<agent-ip>/lineup.json",
 "TunerCount": 2
}
```
**Headers:**
- Content-Type: application/json; charset="utf-8"
- Access-Control-Allow-Origin: *

### 2. **Lineup (Channel List) & Status**

#### `GET /lineup.json`  |  `GET /lineup.xml`

**Purpose:**  
Expose available channels and stream URLs.

**Sample Request:**
```
GET /lineup.json HTTP/1.1
Host: <agent-ip>
User-Agent: PlexMediaServer/...
Accept: */*
Accept-Encoding: gzip
```

**Sample JSON Response:**
```json
[
  {"GuideNumber":"1","GuideName":"BBC ONE Lon","VideoCodec":"MPEG2","AudioCodec":"MPEG","SignalStrength":98,"SignalQuality":100,"URL":"http://<agent-ip>:5004/auto/v1"}
  // ...
]
```
**Headers:**
- Content-Type: application/json; charset="utf-8"
- Access-Control-Allow-Origin: *

#### `GET /lineup_status.json`

**Purpose:**  
Provide scan/progress information.

**Sample Response:**
```json
{
 "ScanInProgress":1,
 "ScanPossible":1,
 "Source":"Antenna",
 "SourceList":["Antenna","Cable"],
 "NetworkID":0,
 "NetworkIDList":[]
}
```

#### `POST /lineup.post?scan=start&source=Antenna`

**Purpose:**  
Trigger a scan for channels.  
**Request Body:** empty  
**Headers:** Content-Type: application/x-www-form-urlencoded

**Response:** 200 OK, Content-Length: 0

---

### 3. **Streaming**

#### `GET /auto/vN` (Usually on port 5004)

**Purpose:**  
Initiate a live MPEG-TS stream for channel `N`.

**Sample Headers:**
```
GET /auto/vN HTTP/1.1
Host: <agent-ip>:5004
User-Agent: Lavf/60.16.100
Accept: */*
Range: bytes=0-
Connection: close
Icy-MetaData: 1
```

**Success Response:**
- HTTP/1.1 200 OK
- Content-Type: video/MP2T or application/octet-stream
- Connection: close
- Body: Raw MPEG-TS

**Failure (e.g., all tuners busy):**
- HTTP/1.1 503 Service Unavailable
- X-HDHomeRun-Error: 803 System Busy

---

## Connection Style

- All requests use `Connection: close`. No HTTP Keep-Alive or pipelining.
- No authentication expected.
- CORS header `Access-Control-Allow-Origin: *` returned on all JSON/XML endpoints.

---

## Error Handling

- For system busy/tuner exhausted, respond with `503` **and** an `X-HDHomeRun-Error` header.

---

## Compatibility Hints

- User-Agent in discovery/lineup: PlexMediaServer/… (OK if you use your own string).
- User-Agent for streaming: Lavf/…, ffmpeg/libav style string expected.
- `Range: bytes=0-` is required for most clients.
- `Icy-MetaData: 1` can be present and safely ignored.

---

## Example Minimal Request/Response Table

| Type            | Method/URL          | Key Request Headers                 | Key Response Headers / Status       |
|-----------------|--------------------|-------------------------------------|-------------------------------------|
| Discovery       | GET /discover.json  | See above                           | 200, JSON, CORS, close             |
| Lineup          | GET /lineup.json    | See above                           | 200, JSON, CORS, close             |
| Stream channel  | GET /auto/vN:5004   | Host:…:5004, User-Agent: Lavf…, etc | 200 + MPEG-TS body OR 503 + error   |
| Scan command    | POST lineup.post... | Content-Type, no body               | 200, CORS, close                    |

---

## Notes

- Always format JSON with proper headers and `charset="utf-8"`.
- Return `Cache-Control: no-cache` on all API responses.
- Date header should be in RFC 1123 (`Date: Tue, 10 Mar 2026 15:12:46 GMT`).

---

## Example Emulation Sequence

1. Client gets `/discover.json`.
2. Client fetches `/lineup.json` to enumerate channels and their streaming URLs.
3. Client starts streaming via `GET /auto/vN` on port 5004.
4. If a scan is needed, client uses `POST /lineup.post?scan=start`.
5. Client may poll `/lineup_status.json` for scan progress.

---

**For best compatibility, all above behaviors should be mimicked as closely as possible.**

---

*End of AGENT.md*
