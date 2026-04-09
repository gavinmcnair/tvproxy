package handler

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/gavinmcnair/tvproxy/pkg/models"
	"github.com/gavinmcnair/tvproxy/pkg/service"
	"github.com/gavinmcnair/tvproxy/pkg/tvsatipscan"
)

func sanitizeHost(host string) string {
	for _, scheme := range []string{"http://", "https://", "rtsp://", "rtsps://"} {
		host = strings.TrimPrefix(host, scheme)
	}
	return strings.TrimRight(host, "/")
}

type SatIPHandler struct {
	satipService *service.SatIPService
}

func NewSatIPHandler(satipService *service.SatIPService) *SatIPHandler {
	return &SatIPHandler{satipService: satipService}
}

func (h *SatIPHandler) List(w http.ResponseWriter, r *http.Request) {
	sources, err := h.satipService.ListSources(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list satip sources")
		return
	}
	respondJSON(w, http.StatusOK, sources)
}

func (h *SatIPHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name            string `json:"name"`
		Host            string `json:"host"`
		HTTPPort        int    `json:"http_port"`
		IsEnabled       bool   `json:"is_enabled"`
		TransmitterFile string `json:"transmitter_file"`
		SourceProfileID string `json:"source_profile_id"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.Host == "" || req.TransmitterFile == "" {
		respondError(w, http.StatusBadRequest, "name, host, and transmitter_file are required")
		return
	}

	source := &models.SatIPSource{
		Name:            req.Name,
		Host:            sanitizeHost(req.Host),
		HTTPPort:        req.HTTPPort,
		IsEnabled:       req.IsEnabled,
		TransmitterFile: req.TransmitterFile,
		SourceProfileID: req.SourceProfileID,
	}

	if err := h.satipService.CreateSource(r.Context(), source); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			respondError(w, http.StatusConflict, err.Error())
		} else {
			respondError(w, http.StatusInternalServerError, "failed to create satip source")
		}
		return
	}

	respondJSON(w, http.StatusCreated, source)
}

func (h *SatIPHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	source, err := h.satipService.GetSource(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "satip source not found")
		return
	}

	respondJSON(w, http.StatusOK, source)
}

func (h *SatIPHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	source, err := h.satipService.GetSource(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "satip source not found")
		return
	}

	var req struct {
		Name            string `json:"name"`
		Host            string `json:"host"`
		HTTPPort        int    `json:"http_port"`
		IsEnabled       bool   `json:"is_enabled"`
		TransmitterFile string `json:"transmitter_file"`
		SourceProfileID string `json:"source_profile_id"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.TransmitterFile == "" {
		respondError(w, http.StatusBadRequest, "transmitter_file is required")
		return
	}

	if req.Name != "" {
		source.Name = req.Name
	}
	if req.Host != "" {
		source.Host = sanitizeHost(req.Host)
	}
	source.HTTPPort = req.HTTPPort
	source.IsEnabled = req.IsEnabled
	source.TransmitterFile = req.TransmitterFile
	source.SourceProfileID = req.SourceProfileID

	if err := h.satipService.UpdateSource(r.Context(), source); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update satip source")
		return
	}

	respondJSON(w, http.StatusOK, source)
}

func (h *SatIPHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.satipService.DeleteSource(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete satip source")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *SatIPHandler) Scan(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	go func() {
		if err := h.satipService.ScanSource(context.Background(), id); err != nil {
			h.satipService.Log().Error().Err(err).Str("source_id", id).Msg("background satip scan failed")
		}
	}()

	respondJSON(w, http.StatusAccepted, map[string]string{"message": "scan started"})
}

func (h *SatIPHandler) ScanStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	respondJSON(w, http.StatusOK, h.satipService.Get(id))
}

func (h *SatIPHandler) Signal(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if !strings.HasPrefix(url, "rtsp://") && !strings.HasPrefix(url, "rtsps://") {
		respondError(w, http.StatusBadRequest, "url must be an rtsp:// address")
		return
	}
	info, err := tvsatipscan.QuerySignal(url, 5*time.Second)
	if err != nil {
		respondError(w, http.StatusBadGateway, "signal query failed: "+err.Error())
		return
	}
	if info == nil {
		respondError(w, http.StatusNotFound, "no tuner signal data in response")
		return
	}
	type response struct {
		*tvsatipscan.SignalInfo
		LevelPct   int `json:"level_pct"`
		QualityPct int `json:"quality_pct"`
	}
	respondJSON(w, http.StatusOK, response{
		SignalInfo: info,
		LevelPct:   info.LevelPct(),
		QualityPct: info.QualityPct(),
	})
}

func (h *SatIPHandler) Clear(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.satipService.ClearSource(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to clear satip source streams")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "streams cleared"})
}

func (h *SatIPHandler) ListTransmitters(w http.ResponseWriter, r *http.Request) {
	system := r.URL.Query().Get("system")
	if system == "" {
		respondError(w, http.StatusBadRequest, "system parameter required (e.g. dvb-t, dvb-s, dvb-c)")
		return
	}

	names, err := tvsatipscan.ListTransmitters(system)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list transmitters: "+err.Error())
		return
	}

	type entry struct {
		Name string `json:"name"`
		File string `json:"file"`
	}
	result := make([]entry, len(names))
	for i, n := range names {
		result[i] = entry{Name: n, File: system + "/" + n}
	}
	respondJSON(w, http.StatusOK, result)
}
