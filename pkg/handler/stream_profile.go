package handler

import (
	"net/http"

	"github.com/gavinmcnair/tvproxy/pkg/models"
	"github.com/gavinmcnair/tvproxy/pkg/repository"
	"github.com/gavinmcnair/tvproxy/pkg/service"
)

// StreamProfileHandler handles stream profile HTTP requests.
type StreamProfileHandler struct {
	repo *repository.StreamProfileRepository
}

// NewStreamProfileHandler creates a new StreamProfileHandler.
func NewStreamProfileHandler(repo *repository.StreamProfileRepository) *StreamProfileHandler {
	return &StreamProfileHandler{repo: repo}
}

var validSourceTypes = map[string]bool{"direct": true, "satip": true, "m3u": true}
var validHWAccels = map[string]bool{"none": true, "qsv": true, "nvenc": true, "vaapi": true, "videotoolbox": true}
var validVideoCodecs = map[string]bool{"copy": true, "h264": true, "h265": true, "av1": true}

// List returns all stream profiles.
func (h *StreamProfileHandler) List(w http.ResponseWriter, r *http.Request) {
	profiles, err := h.repo.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list stream profiles")
		return
	}

	respondJSON(w, http.StatusOK, profiles)
}

// Create creates a new stream profile.
func (h *StreamProfileHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name       string `json:"name"`
		SourceType string `json:"source_type"`
		HWAccel    string `json:"hwaccel"`
		VideoCodec string `json:"video_codec"`
		CustomArgs string `json:"custom_args"`
		IsDefault  bool   `json:"is_default"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}

	// Default dropdown values if not provided
	if req.SourceType == "" {
		req.SourceType = "direct"
	}
	if req.HWAccel == "" {
		req.HWAccel = "none"
	}
	if req.VideoCodec == "" {
		req.VideoCodec = "copy"
	}

	if !validSourceTypes[req.SourceType] {
		respondError(w, http.StatusBadRequest, "invalid source_type")
		return
	}
	if !validHWAccels[req.HWAccel] {
		respondError(w, http.StatusBadRequest, "invalid hwaccel")
		return
	}
	if !validVideoCodecs[req.VideoCodec] {
		respondError(w, http.StatusBadRequest, "invalid video_codec")
		return
	}

	var command, args string
	if req.CustomArgs != "" {
		command = "ffmpeg"
		args = req.CustomArgs
	} else {
		command, args = service.ComposeStreamProfileArgs(req.SourceType, req.HWAccel, req.VideoCodec)
	}

	profile := &models.StreamProfile{
		Name:       req.Name,
		SourceType: req.SourceType,
		HWAccel:    req.HWAccel,
		VideoCodec: req.VideoCodec,
		CustomArgs: req.CustomArgs,
		Command:    command,
		Args:       args,
		IsDefault:  req.IsDefault,
	}

	if err := h.repo.Create(r.Context(), profile); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create stream profile")
		return
	}

	respondJSON(w, http.StatusCreated, profile)
}

// Get returns a stream profile by ID.
func (h *StreamProfileHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamInt64(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid stream profile id")
		return
	}

	profile, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "stream profile not found")
		return
	}

	respondJSON(w, http.StatusOK, profile)
}

// Update updates a stream profile by ID.
func (h *StreamProfileHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamInt64(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid stream profile id")
		return
	}

	profile, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "stream profile not found")
		return
	}

	var req struct {
		Name       string `json:"name"`
		SourceType string `json:"source_type"`
		HWAccel    string `json:"hwaccel"`
		VideoCodec string `json:"video_codec"`
		CustomArgs string `json:"custom_args"`
		IsDefault  bool   `json:"is_default"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name != "" {
		profile.Name = req.Name
	}

	// Default dropdown values if not provided
	if req.SourceType == "" {
		req.SourceType = profile.SourceType
	}
	if req.HWAccel == "" {
		req.HWAccel = profile.HWAccel
	}
	if req.VideoCodec == "" {
		req.VideoCodec = profile.VideoCodec
	}

	if !validSourceTypes[req.SourceType] {
		respondError(w, http.StatusBadRequest, "invalid source_type")
		return
	}
	if !validHWAccels[req.HWAccel] {
		respondError(w, http.StatusBadRequest, "invalid hwaccel")
		return
	}
	if !validVideoCodecs[req.VideoCodec] {
		respondError(w, http.StatusBadRequest, "invalid video_codec")
		return
	}

	profile.SourceType = req.SourceType
	profile.HWAccel = req.HWAccel
	profile.VideoCodec = req.VideoCodec
	profile.CustomArgs = req.CustomArgs
	profile.IsDefault = req.IsDefault

	if req.CustomArgs != "" {
		profile.Command = "ffmpeg"
		profile.Args = req.CustomArgs
	} else {
		profile.Command, profile.Args = service.ComposeStreamProfileArgs(req.SourceType, req.HWAccel, req.VideoCodec)
	}

	if err := h.repo.Update(r.Context(), profile); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update stream profile")
		return
	}

	respondJSON(w, http.StatusOK, profile)
}

// Delete deletes a stream profile by ID.
func (h *StreamProfileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamInt64(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid stream profile id")
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete stream profile")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
