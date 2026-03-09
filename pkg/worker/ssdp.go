package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/koron/go-ssdp"
	"github.com/rs/zerolog"

	"github.com/gavinmcnair/tvproxy/pkg/models"
	"github.com/gavinmcnair/tvproxy/pkg/repository"
)

// SSDPWorker advertises HDHR devices via SSDP so Plex/Emby/Jellyfin
// can auto-discover them on the local network.
type SSDPWorker struct {
	hdhrDeviceRepo *repository.HDHRDeviceRepository
	baseURL        string
	log            zerolog.Logger
}

// NewSSDPWorker creates a new SSDP discovery worker.
func NewSSDPWorker(hdhrDeviceRepo *repository.HDHRDeviceRepository, baseURL string, log zerolog.Logger) *SSDPWorker {
	return &SSDPWorker{
		hdhrDeviceRepo: hdhrDeviceRepo,
		baseURL:        baseURL,
		log:            log.With().Str("worker", "ssdp").Logger(),
	}
}

// Run starts the SSDP advertiser. It finds the first enabled HDHR device and
// advertises it as a UPnP root device. The FriendlyName shown in Plex/Emby
// matches the device Name from the HDHR settings (e.g., "tvproxy (Movies)").
func (w *SSDPWorker) Run(ctx context.Context) {
	// Wait briefly for the HTTP server to start
	select {
	case <-time.After(2 * time.Second):
	case <-ctx.Done():
		return
	}

	device := w.findEnabledDevice(ctx)
	if device == nil {
		w.log.Info().Msg("no enabled HDHR devices found, SSDP not started")
		return
	}

	location := fmt.Sprintf("%s/hdhr/device.xml", w.baseURL)
	usn := fmt.Sprintf("uuid:%s::upnp:rootdevice", device.DeviceID)

	w.log.Info().
		Str("device", device.Name).
		Str("device_id", device.DeviceID).
		Str("location", location).
		Msg("starting SSDP advertiser")

	ad, err := ssdp.Advertise(
		"upnp:rootdevice",
		usn,
		location,
		device.Name, // FriendlyName shown during discovery
		1800,        // cache-control max-age
	)
	if err != nil {
		w.log.Error().Err(err).Msg("failed to start SSDP advertiser")
		return
	}
	defer ad.Close()

	// Send alive every 300 seconds until shutdown
	ticker := time.NewTicker(300 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if err := ad.Bye(); err != nil {
				w.log.Warn().Err(err).Msg("SSDP bye failed")
			}
			w.log.Info().Msg("SSDP advertiser stopped")
			return
		case <-ticker.C:
			if err := ad.Alive(); err != nil {
				w.log.Warn().Err(err).Msg("SSDP alive failed")
			}
		}
	}
}

func (w *SSDPWorker) findEnabledDevice(ctx context.Context) *models.HDHRDevice {
	devices, err := w.hdhrDeviceRepo.List(ctx)
	if err != nil {
		w.log.Error().Err(err).Msg("failed to list HDHR devices for SSDP")
		return nil
	}
	for i := range devices {
		if devices[i].IsEnabled {
			return &devices[i]
		}
	}
	return nil
}
