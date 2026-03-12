package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/gavinmcnair/tvproxy/pkg/models"
	"github.com/gavinmcnair/tvproxy/pkg/repository"
)

type LogoHandler struct {
	repo *repository.LogoRepository
}

func NewLogoHandler(repo *repository.LogoRepository) *LogoHandler {
	return &LogoHandler{repo: repo}
}

func (h *LogoHandler) List(w http.ResponseWriter, r *http.Request) {
	logos, err := h.repo.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list logos")
		return
	}

	respondJSON(w, http.StatusOK, logos)
}

func (h *LogoHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.URL == "" {
		respondError(w, http.StatusBadRequest, "name and url are required")
		return
	}

	logo := &models.Logo{
		Name: req.Name,
		URL:  req.URL,
	}

	if err := h.repo.Create(r.Context(), logo); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create logo")
		return
	}

	respondJSON(w, http.StatusCreated, logo)
}

func (h *LogoHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	logo, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "logo not found")
		return
	}

	respondJSON(w, http.StatusOK, logo)
}

func (h *LogoHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	logo, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "logo not found")
		return
	}

	var req struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name != "" {
		logo.Name = req.Name
	}
	if req.URL != "" {
		logo.URL = req.URL
	}

	if err := h.repo.Update(r.Context(), logo); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update logo")
		return
	}

	respondJSON(w, http.StatusOK, logo)
}

func (h *LogoHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.repo.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete logo")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
