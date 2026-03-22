FROM golang:1.24-bookworm AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -ldflags="-s -w -X main.buildVersion=$VERSION" -o /tvproxy ./cmd/tvproxy/

# ffmpeg 8.x with --enable-libvpl (QSV/oneVPL), VA-API, NVENC, Vulkan,
# and all HW encoders (av1_qsv, av1_vaapi, h264_qsv, hevc_vaapi, etc.).
FROM linuxserver/ffmpeg:latest

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
    gosu \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /tvproxy /usr/local/bin/tvproxy
COPY pkg/defaults/clients.json /data/clients.json
COPY pkg/defaults/settings.json /data/settings.json

# Create tvproxy user at default UID 1000.
RUN (usermod -l tvproxy -d /home/tvproxy ubuntu 2>/dev/null && groupmod -n tvproxy ubuntu 2>/dev/null || useradd -m -u 1000 tvproxy)

COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

WORKDIR /data

ENV PUID=1000
ENV PGID=1000

# For Intel Arc GPU access, run with: --device /dev/dri:/dev/dri
# For NVIDIA GPU access, run with: --gpus all (requires nvidia-container-toolkit)
EXPOSE 8080

ENTRYPOINT ["entrypoint.sh"]
