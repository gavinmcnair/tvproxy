package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/gavinmcnair/tvproxy/pkg/models"
	"github.com/gavinmcnair/tvproxy/pkg/repository"
)

type ChannelProfileHandler struct {
	repo *repository.ChannelProfileRepository
}

func NewChannelProfileHandler(repo *repository.ChannelProfileRepository) *ChannelProfileHandler {
	return &ChannelProfileHandler{repo: repo}
}

func (h *ChannelProfileHandler) List(w http.ResponseWriter, r *http.Request) {
	profiles, err := h.repo.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list channel profiles")
		return
	}

	respondJSON(w, http.StatusOK, profiles)
}

func (h *ChannelProfileHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name          string `json:"name"`
		StreamProfile string `json:"stream_profile"`
		SortOrder     int    `json:"sort_order"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}

	profile := &models.ChannelProfile{
		Name:          req.Name,
		StreamProfile: req.StreamProfile,
		SortOrder:     req.SortOrder,
	}

	if err := h.repo.Create(r.Context(), profile); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create channel profile")
		return
	}

	respondJSON(w, http.StatusCreated, profile)
}

func (h *ChannelProfileHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	profile, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "channel profile not found")
		return
	}

	respondJSON(w, http.StatusOK, profile)
}

func (h *ChannelProfileHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	profile, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "channel profile not found")
		return
	}

	var req struct {
		Name          string `json:"name"`
		StreamProfile string `json:"stream_profile"`
		SortOrder     int    `json:"sort_order"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name != "" {
		profile.Name = req.Name
	}
	profile.StreamProfile = req.StreamProfile
	profile.SortOrder = req.SortOrder

	if err := h.repo.Update(r.Context(), profile); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update channel profile")
		return
	}

	respondJSON(w, http.StatusOK, profile)
}

func (h *ChannelProfileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.repo.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete channel profile")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
