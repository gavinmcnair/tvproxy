package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/gavinmcnair/tvproxy/pkg/ffmpeg"
	"github.com/gavinmcnair/tvproxy/pkg/models"
	"github.com/gavinmcnair/tvproxy/pkg/repository"
)

type StreamProfileHandler struct {
	repo *repository.StreamProfileRepository
}

func NewStreamProfileHandler(repo *repository.StreamProfileRepository) *StreamProfileHandler {
	return &StreamProfileHandler{repo: repo}
}

var validStreamModes = map[string]bool{"direct": true, "proxy": true, "ffmpeg": true}
var validSourceTypes = map[string]bool{"satip": true, "m3u": true}
var validHWAccels = map[string]bool{"none": true, "qsv": true, "nvenc": true, "vaapi": true, "videotoolbox": true}
var validVideoCodecs = map[string]bool{"copy": true, "h264": true, "h265": true, "av1": true}
var validContainers = map[string]bool{"mpegts": true, "matroska": true, "mp4": true, "webm": true}
var validFPSModes = map[string]bool{"auto": true, "cfr": true}

func composeArgs(sourceType, hwaccel, videoCodec, container, fpsMode, customArgs string, deinterlace, useCustom bool) string {
	if useCustom {
		return customArgs
	}
	composed := ffmpeg.ComposeStreamProfileArgs(ffmpeg.ComposeOptions{
		SourceType:  sourceType,
		HWAccel:     hwaccel,
		VideoCodec:  videoCodec,
		Container:   container,
		Deinterlace: deinterlace,
		FPSMode:     fpsMode,
	})
	if customArgs != "" && composed != "" {
		return composed + " " + customArgs
	}
	return composed
}

func (h *StreamProfileHandler) List(w http.ResponseWriter, r *http.Request) {
	profiles, err := h.repo.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list stream profiles")
		return
	}

	respondJSON(w, http.StatusOK, profiles)
}

func (h *StreamProfileHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name          string `json:"name"`
		StreamMode    string `json:"stream_mode"`
		SourceType    string `json:"source_type"`
		HWAccel       string `json:"hwaccel"`
		VideoCodec    string `json:"video_codec"`
		Container     string `json:"container"`
		Deinterlace   bool   `json:"deinterlace"`
		FPSMode       string `json:"fps_mode"`
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

	if existing, _ := h.repo.GetByName(r.Context(), req.Name); existing != nil {
		respondError(w, http.StatusConflict, "stream profile name already exists")
		return
	}

	if req.StreamMode == "" {
		req.StreamMode = "ffmpeg"
	}
	if !validStreamModes[req.StreamMode] {
		respondError(w, http.StatusBadRequest, "invalid stream_mode")
		return
	}

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
	if req.FPSMode == "" {
		req.FPSMode = "auto"
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
	if !validFPSModes[req.FPSMode] {
		respondError(w, http.StatusBadRequest, "invalid fps_mode")
		return
	}

	fullArgs := composeArgs(req.SourceType, req.HWAccel, req.VideoCodec, req.Container, req.FPSMode, req.CustomArgs, req.Deinterlace, req.UseCustomArgs)

	profile := &models.StreamProfile{
		Name:          req.Name,
		StreamMode:    req.StreamMode,
		SourceType:    req.SourceType,
		HWAccel:       req.HWAccel,
		VideoCodec:    req.VideoCodec,
		Container:     req.Container,
		Deinterlace:   req.Deinterlace,
		FPSMode:       req.FPSMode,
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

func (h *StreamProfileHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	profile, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "stream profile not found")
		return
	}

	respondJSON(w, http.StatusOK, profile)
}

func (h *StreamProfileHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

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
		Deinterlace   bool   `json:"deinterlace"`
		FPSMode       string `json:"fps_mode"`
		UseCustomArgs bool   `json:"use_custom_args"`
		CustomArgs    string `json:"custom_args"`
		IsDefault     bool   `json:"is_default"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name != "" && req.Name != profile.Name {
		if profile.IsClient {
			respondError(w, http.StatusForbidden, "cannot rename client profile")
			return
		}
		if existing, _ := h.repo.GetByName(r.Context(), req.Name); existing != nil {
			respondError(w, http.StatusConflict, "stream profile name already exists")
			return
		}
		profile.Name = req.Name
	}

	if req.StreamMode == "" {
		req.StreamMode = profile.StreamMode
	}
	if !validStreamModes[req.StreamMode] {
		respondError(w, http.StatusBadRequest, "invalid stream_mode")
		return
	}

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
	if req.FPSMode == "" {
		req.FPSMode = profile.FPSMode
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
	if !validFPSModes[req.FPSMode] {
		respondError(w, http.StatusBadRequest, "invalid fps_mode")
		return
	}

	profile.StreamMode = req.StreamMode
	profile.SourceType = req.SourceType
	profile.HWAccel = req.HWAccel
	profile.VideoCodec = req.VideoCodec
	profile.Container = req.Container
	profile.Deinterlace = req.Deinterlace
	profile.FPSMode = req.FPSMode
	profile.IsDefault = req.IsDefault

	fullArgs := composeArgs(req.SourceType, req.HWAccel, req.VideoCodec, req.Container, req.FPSMode, req.CustomArgs, req.Deinterlace, req.UseCustomArgs)

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

func (h *StreamProfileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

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
