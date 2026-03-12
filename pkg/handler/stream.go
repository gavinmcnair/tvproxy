package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/gavinmcnair/tvproxy/pkg/repository"
)

type StreamHandler struct {
	streamRepo *repository.StreamRepository
}

func NewStreamHandler(streamRepo *repository.StreamRepository) *StreamHandler {
	return &StreamHandler{streamRepo: streamRepo}
}

func (h *StreamHandler) List(w http.ResponseWriter, r *http.Request) {
	accountIDStr := r.URL.Query().Get("account_id")
	if accountIDStr != "" {
		streams, err := h.streamRepo.ListByAccountID(r.Context(), accountIDStr)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "failed to list streams")
			return
		}

		respondJSON(w, http.StatusOK, streams)
		return
	}

	if r.URL.Query().Get("full") == "true" {
		streams, err := h.streamRepo.List(r.Context())
		if err != nil {
			respondError(w, http.StatusInternalServerError, "failed to list streams")
			return
		}
		respondJSON(w, http.StatusOK, streams)
		return
	}

	// Default: lightweight summaries (id, name, group, account_id only)
	summaries, err := h.streamRepo.ListSummaries(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list streams")
		return
	}

	respondJSON(w, http.StatusOK, summaries)
}

func (h *StreamHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	stream, err := h.streamRepo.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "stream not found")
		return
	}

	respondJSON(w, http.StatusOK, stream)
}

func (h *StreamHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.streamRepo.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete stream")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
