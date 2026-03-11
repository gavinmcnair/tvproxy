package handler

import (
	"net/http"

	"github.com/gavinmcnair/tvproxy/pkg/ffmpeg"
	"github.com/gavinmcnair/tvproxy/pkg/models"
	"github.com/gavinmcnair/tvproxy/pkg/repository"
)

// StreamProfileHandler handles stream profile HTTP requests.
type StreamProfileHandler struct {
	repo *repository.StreamProfileRepository
}

// NewStreamProfileHandler creates a new StreamProfileHandler.
func NewStreamProfileHandler(repo *repository.StreamProfileRepository) *StreamProfileHandler {
	return &StreamProfileHandler{repo: repo}
}

var validStreamModes = map[string]bool{"direct": true, "proxy": true, "ffmpeg": true}
var validSourceTypes = map[string]bool{"satip": true, "m3u": true}
var validHWAccels = map[string]bool{"none": true, "qsv": true, "nvenc": true, "vaapi": true, "videotoolbox": true}
var validVideoCodecs = map[string]bool{"copy": true, "h264": true, "h265": true, "av1": true}
var validContainers = map[string]bool{"mpegts": true, "matroska": true, "mp4": true, "webm": true}

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
		Name          string `json:"name"`
		StreamMode    string `json:"stream_mode"`
		SourceType    string `json:"source_type"`
		HWAccel       string `json:"hwaccel"`
		VideoCodec    string `json:"video_codec"`
		Container     string `json:"container"`
		UseCustomArgs bool   `json:"use_custom_args"`
		CustomArgs    string `json:"custom_args"`
		IsDefault     bool   `json:"is_default"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}

	// Default stream mode to "ffmpeg" if not provided
	if req.StreamMode == "" {
		req.StreamMode = "ffmpeg"
	}
	if !validStreamModes[req.StreamMode] {
		respondError(w, http.StatusBadRequest, "invalid stream_mode")
		return
	}

	// Default dropdown values if not provided
	if req.SourceType == "" {
		req.SourceType = "m3u"
	}
	if req.HWAccel == "" {
		req.HWAccel = "none"
	}
	if req.VideoCodec == "" {
		req.VideoCodec = "copy"
	}
	if req.Container == "" {
		req.Container = ffmpeg.DefaultContainer(req.VideoCodec)
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
	if !validContainers[req.Container] {
		respondError(w, http.StatusBadRequest, "invalid container")
		return
	}

	var fullArgs string
	if req.UseCustomArgs {
		fullArgs = req.CustomArgs
	} else {
		composed := ffmpeg.ComposeStreamProfileArgs(req.SourceType, req.HWAccel, req.VideoCodec, req.Container)
		fullArgs = composed
		if req.CustomArgs != "" && composed != "" {
			fullArgs = composed + " " + req.CustomArgs
		}
	}

	profile := &models.StreamProfile{
		Name:          req.Name,
		StreamMode:    req.StreamMode,
		SourceType:    req.SourceType,
		HWAccel:       req.HWAccel,
		VideoCodec:    req.VideoCodec,
		Container:     req.Container,
		UseCustomArgs: req.UseCustomArgs,
		CustomArgs:    req.CustomArgs,
		Command:       "ffmpeg",
		Args:          fullArgs,
		IsDefault:     req.IsDefault,
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

	if profile.IsSystem {
		respondError(w, http.StatusForbidden, "cannot edit system profile")
		return
	}

	var req struct {
		Name          string `json:"name"`
		StreamMode    string `json:"stream_mode"`
		SourceType    string `json:"source_type"`
		HWAccel       string `json:"hwaccel"`
		VideoCodec    string `json:"video_codec"`
		Container     string `json:"container"`
		UseCustomArgs bool   `json:"use_custom_args"`
		CustomArgs    string `json:"custom_args"`
		IsDefault     bool   `json:"is_default"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name != "" {
		profile.Name = req.Name
	}

	// Default stream mode to existing value if not provided
	if req.StreamMode == "" {
		req.StreamMode = profile.StreamMode
	}
	if !validStreamModes[req.StreamMode] {
		respondError(w, http.StatusBadRequest, "invalid stream_mode")
		return
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
	if req.Container == "" {
		req.Container = profile.Container
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
	if !validContainers[req.Container] {
		respondError(w, http.StatusBadRequest, "invalid container")
		return
	}

	profile.StreamMode = req.StreamMode
	profile.SourceType = req.SourceType
	profile.HWAccel = req.HWAccel
	profile.VideoCodec = req.VideoCodec
	profile.Container = req.Container
	profile.IsDefault = req.IsDefault

	var fullArgs string
	if req.UseCustomArgs {
		fullArgs = req.CustomArgs
	} else {
		composed := ffmpeg.ComposeStreamProfileArgs(req.SourceType, req.HWAccel, req.VideoCodec, req.Container)
		fullArgs = composed
		if req.CustomArgs != "" && composed != "" {
			fullArgs = composed + " " + req.CustomArgs
		}
	}

	profile.UseCustomArgs = req.UseCustomArgs
	profile.CustomArgs = req.CustomArgs
	profile.Command = "ffmpeg"
	profile.Args = fullArgs

	if err := h.repo.Update(r.Context(), profile); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update stream profile")
		return
	}

	respondJSON(w, http.StatusOK, profile)
}

// Delete deletes a stream profile by ID. System profiles cannot be deleted.
func (h *StreamProfileHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

	if profile.IsSystem || profile.IsClient {
		respondError(w, http.StatusForbidden, "cannot delete system or client profile")
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete stream profile")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
