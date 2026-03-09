FROM golang:1.22-bookworm AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /tvproxy ./cmd/tvproxy/

FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive

# Enable multiverse repo and install ffmpeg with hardware acceleration support.
# Intel Arc A380 / DG2 drivers (AV1, H.264, HEVC) are installed on amd64 only.
RUN sed -i 's/Components: main restricted/Components: main restricted multiverse/' /etc/apt/sources.list.d/ubuntu.sources && \
    apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
    ffmpeg \
    vainfo \
    mesa-va-drivers \
    libva-drm2 \
    libva2 \
    && ARCH=$(dpkg --print-architecture) && \
    if [ "$ARCH" = "amd64" ]; then \
      apt-get install -y --no-install-recommends \
        intel-media-va-driver-non-free \
        i965-va-driver; \
    fi \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /tvproxy /usr/local/bin/tvproxy

# Use UID 1000 for compatibility with existing data volumes.
# Ubuntu 24.04 has 'ubuntu' user at UID 1000; reuse it and rename.
RUN usermod -l tvproxy -d /home/tvproxy ubuntu 2>/dev/null || useradd -m -u 1000 tvproxy
USER tvproxy
WORKDIR /data

# For Intel Arc GPU access, run with: --device /dev/dri:/dev/dri
# For NVIDIA GPU access, run with: --gpus all (requires nvidia-container-toolkit)
EXPOSE 8080

ENTRYPOINT ["tvproxy"]
